package data

import (
	"context"
	"deps-dev/depsdev"
	"deps-dev/storage"
	"fmt"
	"sync"

	"github.com/sirupsen/logrus"
)

type Storage interface {
	UpsertDependencies(ctx context.Context, deps []storage.Dependency) error
	GetDependenciesMap(ctx context.Context, deps []storage.Dependency) (map[string]storage.Dependency, error)
}

type DepsDevAPI interface {
	GetDependencyGraph(ctx context.Context, system, name, version string) (*depsdev.DependencyGraph, error)
	GetPackageMetadata(ctx context.Context, vk depsdev.VersionKey) (*depsdev.PackageVersionMetadata, error)
	GetScorecardData(ctx context.Context, meta *depsdev.PackageVersionMetadata) depsdev.ScorecardInfo
}

type DataManager struct {
	Store         Storage
	API           DepsDevAPI
	Log           *logrus.Logger
	MaxConcurrent int
}

func (dm *DataManager) RefreshDependencies(ctx context.Context, system, name, version string) error {
	dm.Log.Infof("Fetching dependencies for %s/%s@%s", system, name, version)

	// Fetch data from deps.dev
	fetchedDeps, err := dm.fetchDependenciesWithScores(ctx, system, name, version)
	if err != nil {
		dm.Log.WithError(err).Error("failed to fetch dependencies")
		return err
	}

	// Fetch existing records from DB
	existingMap, err := dm.Store.GetDependenciesMap(ctx, fetchedDeps)
	if err != nil {
		dm.Log.WithError(err).Error("failed to get existing dependencies")
		return err
	}

	// Merge only non-empty fields from incoming
	var mergedDeps []storage.Dependency
	for _, incoming := range fetchedDeps {
		key := fmt.Sprintf("%s|%s|%s", incoming.System, incoming.Name, incoming.Version)

		existing, found := existingMap[key]
		if !found {
			mergedDeps = append(mergedDeps, incoming)
			continue
		}

		// Merge non-empty fields from incoming into existing
		merged := existing
		if incoming.Relation != "" {
			merged.Relation = incoming.Relation
		}
		if incoming.SourceRepo != "" {
			merged.SourceRepo = incoming.SourceRepo
		}
		if incoming.OpenSSFScore != nil {
			merged.OpenSSFScore = incoming.OpenSSFScore
		}

		mergedDeps = append(mergedDeps, merged)
	}

	// Upsert into DB
	if err := dm.Store.UpsertDependencies(ctx, mergedDeps); err != nil {
		dm.Log.WithError(err).Error("failed to upsert dependencies to database")
		return err
	}

	dm.Log.Infof("Successfully upserted %d dependencies", len(mergedDeps))
	return nil
}

func (dm *DataManager) fetchDependenciesWithScores(ctx context.Context, system, name, version string) ([]storage.Dependency, error) {
	graph, err := dm.API.GetDependencyGraph(ctx, system, name, version)
	if err != nil {
		return nil, err
	}

	var (
		results []storage.Dependency
		mu      sync.Mutex
		wg      sync.WaitGroup
		sem     = make(chan struct{}, 10)
	)

	for _, node := range graph.Nodes {
		wg.Add(1)
		go func(node depsdev.DependencyNode) {
			defer wg.Done()

			select {
			case sem <- struct{}{}:
				defer func() { <-sem }()
			case <-ctx.Done():
				return
			}

			meta, err := dm.API.GetPackageMetadata(ctx, node.VersionKey)
			if err != nil {
				return
			}

			scorecard := dm.API.GetScorecardData(ctx, meta)

			mu.Lock()
			results = append(results, storage.Dependency{
				System:       node.VersionKey.System,
				Name:         node.VersionKey.Name,
				Version:      node.VersionKey.Version,
				Relation:     node.Relation,
				SourceRepo:   scorecard.SourceRepo,
				OpenSSFScore: scorecard.OpenSSFScore,
			})
			mu.Unlock()
		}(node)
	}

	wg.Wait()
	return results, nil
}
