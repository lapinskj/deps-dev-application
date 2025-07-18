package data_test

import (
	"context"
	"deps-dev/data"
	"deps-dev/depsdev"
	"deps-dev/storage"
	"errors"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

type mockDepsDevAPI struct {
	GetDependencyGraphFn func(ctx context.Context, system, name, version string) (*depsdev.DependencyGraph, error)
	GetPackageMetadataFn func(ctx context.Context, vk depsdev.VersionKey) (*depsdev.PackageVersionMetadata, error)
	GetScorecardDataFn   func(ctx context.Context, meta *depsdev.PackageVersionMetadata) depsdev.ScorecardInfo
}

func (m *mockDepsDevAPI) GetDependencyGraph(ctx context.Context, system, name, version string) (*depsdev.DependencyGraph, error) {
	return m.GetDependencyGraphFn(ctx, system, name, version)
}
func (m *mockDepsDevAPI) GetPackageMetadata(ctx context.Context, vk depsdev.VersionKey) (*depsdev.PackageVersionMetadata, error) {
	return m.GetPackageMetadataFn(ctx, vk)
}
func (m *mockDepsDevAPI) GetScorecardData(ctx context.Context, meta *depsdev.PackageVersionMetadata) depsdev.ScorecardInfo {
	return m.GetScorecardDataFn(ctx, meta)
}

type mockStorage struct {
	GetMapFn   func(ctx context.Context, deps []storage.Dependency) (map[string]storage.Dependency, error)
	UpsertFn   func(ctx context.Context, deps []storage.Dependency) error
	Upserted   []storage.Dependency
	LastMerged *[]storage.Dependency
}

func (m *mockStorage) GetDependenciesMap(ctx context.Context, deps []storage.Dependency) (map[string]storage.Dependency, error) {
	return m.GetMapFn(ctx, deps)
}
func (m *mockStorage) UpsertDependencies(ctx context.Context, deps []storage.Dependency) error {
	m.Upserted = deps
	if m.LastMerged != nil {
		*m.LastMerged = deps
	}
	return m.UpsertFn(ctx, deps)
}

func TestRefreshDependencies_Success(t *testing.T) {
	score := 9.5
	api := &mockDepsDevAPI{
		GetDependencyGraphFn: func(ctx context.Context, system, name, version string) (*depsdev.DependencyGraph, error) {
			return &depsdev.DependencyGraph{
				Nodes: []depsdev.DependencyNode{
					{
						VersionKey: depsdev.VersionKey{System: "npm", Name: "react", Version: "18.2.0"},
						Relation:   "SELF",
					},
				},
			}, nil
		},
		GetPackageMetadataFn: func(ctx context.Context, vk depsdev.VersionKey) (*depsdev.PackageVersionMetadata, error) {
			return &depsdev.PackageVersionMetadata{
				RelatedProjects: []depsdev.RelatedProject{
					{
						ProjectKey:   depsdev.ProjectKey{ID: "github.com/facebook/react"},
						RelationType: "SOURCE_REPO",
					},
				},
			}, nil
		},
		GetScorecardDataFn: func(ctx context.Context, meta *depsdev.PackageVersionMetadata) depsdev.ScorecardInfo {
			return depsdev.ScorecardInfo{
				SourceRepo:   "github.com/facebook/react",
				OpenSSFScore: &score,
			}
		},
	}

	var capturedMerged []storage.Dependency

	store := &mockStorage{
		GetMapFn: func(ctx context.Context, deps []storage.Dependency) (map[string]storage.Dependency, error) {
			return map[string]storage.Dependency{}, nil
		},
		UpsertFn: func(ctx context.Context, deps []storage.Dependency) error {
			capturedMerged = deps
			return nil
		},
		LastMerged: &capturedMerged,
	}

	manager := &data.DataManager{
		API:           api,
		Store:         store,
		Log:           logrus.New(),
		MaxConcurrent: 5,
	}

	err := manager.RefreshDependencies(context.Background(), "npm", "react", "18.2.0")
	assert.NoError(t, err)

	assert.Len(t, capturedMerged, 1)
	dep := capturedMerged[0]
	assert.Equal(t, "npm", dep.System)
	assert.Equal(t, "react", dep.Name)
	assert.Equal(t, "18.2.0", dep.Version)
	assert.Equal(t, "SELF", dep.Relation)
	assert.Equal(t, "github.com/facebook/react", dep.SourceRepo)
	assert.NotNil(t, dep.OpenSSFScore)
	assert.Equal(t, 9.5, *dep.OpenSSFScore)
}

func TestRefreshDependencies_MergesExistingFields(t *testing.T) {
	score := 6.6

	api := &mockDepsDevAPI{
		GetDependencyGraphFn: func(ctx context.Context, system, name, version string) (*depsdev.DependencyGraph, error) {
			return &depsdev.DependencyGraph{
				Nodes: []depsdev.DependencyNode{
					{
						VersionKey: depsdev.VersionKey{System: "npm", Name: "react", Version: "18.2.0"},
						Relation:   "direct",
					},
				},
			}, nil
		},
		GetPackageMetadataFn: func(ctx context.Context, vk depsdev.VersionKey) (*depsdev.PackageVersionMetadata, error) {
			return &depsdev.PackageVersionMetadata{
				RelatedProjects: []depsdev.RelatedProject{
					{
						ProjectKey:   depsdev.ProjectKey{ID: "github.com/facebook/react"},
						RelationType: "SOURCE_REPO",
					},
				},
			}, nil
		},
		GetScorecardDataFn: func(ctx context.Context, meta *depsdev.PackageVersionMetadata) depsdev.ScorecardInfo {
			return depsdev.ScorecardInfo{
				SourceRepo:   "github.com/facebook/react",
				OpenSSFScore: &score,
			}
		},
	}

	existing := storage.Dependency{
		System:  "npm",
		Name:    "react",
		Version: "18.2.0",
	}

	store := &mockStorage{
		GetMapFn: func(ctx context.Context, deps []storage.Dependency) (map[string]storage.Dependency, error) {
			key := "npm|react|18.2.0"
			return map[string]storage.Dependency{key: existing}, nil
		},
		UpsertFn: func(ctx context.Context, deps []storage.Dependency) error {
			assert.Len(t, deps, 1)
			dep := deps[0]
			assert.Equal(t, "direct", dep.Relation)
			assert.Equal(t, "github.com/facebook/react", dep.SourceRepo)
			assert.NotNil(t, dep.OpenSSFScore)
			assert.Equal(t, 6.6, *dep.OpenSSFScore)
			return nil
		},
	}

	manager := &data.DataManager{
		API:           api,
		Store:         store,
		Log:           logrus.New(),
		MaxConcurrent: 5,
	}

	err := manager.RefreshDependencies(context.Background(), "npm", "react", "18.2.0")
	assert.NoError(t, err)
}

func TestRefreshDependencies_GraphError(t *testing.T) {
	api := &mockDepsDevAPI{
		GetDependencyGraphFn: func(ctx context.Context, system, name, version string) (*depsdev.DependencyGraph, error) {
			return nil, errors.New("graph fetch failed")
		},
	}

	store := &mockStorage{}

	manager := &data.DataManager{
		API:           api,
		Store:         store,
		Log:           logrus.New(),
		MaxConcurrent: 5,
	}

	err := manager.RefreshDependencies(context.Background(), "npm", "react", "18.2.0")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "graph fetch failed")
}

func TestRefreshDependencies_GetMapError(t *testing.T) {
	score := 8.0

	api := &mockDepsDevAPI{
		GetDependencyGraphFn: func(ctx context.Context, system, name, version string) (*depsdev.DependencyGraph, error) {
			return &depsdev.DependencyGraph{
				Nodes: []depsdev.DependencyNode{
					{
						VersionKey: depsdev.VersionKey{System: "npm", Name: "react", Version: "18.2.0"},
					},
				},
			}, nil
		},
		GetPackageMetadataFn: func(ctx context.Context, vk depsdev.VersionKey) (*depsdev.PackageVersionMetadata, error) {
			return &depsdev.PackageVersionMetadata{
				RelatedProjects: []depsdev.RelatedProject{
					{
						ProjectKey:   depsdev.ProjectKey{ID: "github.com/facebook/react"},
						RelationType: "SOURCE_REPO",
					},
				},
			}, nil
		},
		GetScorecardDataFn: func(ctx context.Context, meta *depsdev.PackageVersionMetadata) depsdev.ScorecardInfo {
			return depsdev.ScorecardInfo{
				SourceRepo:   "github.com/facebook/react",
				OpenSSFScore: &score,
			}
		},
	}

	store := &mockStorage{
		GetMapFn: func(ctx context.Context, deps []storage.Dependency) (map[string]storage.Dependency, error) {
			return nil, errors.New("db map error")
		},
	}

	manager := &data.DataManager{
		API:           api,
		Store:         store,
		Log:           logrus.New(),
		MaxConcurrent: 5,
	}

	err := manager.RefreshDependencies(context.Background(), "npm", "react", "18.2.0")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "db map error")
}

func TestRefreshDependencies_UpsertError(t *testing.T) {
	score := 7.2

	api := &mockDepsDevAPI{
		GetDependencyGraphFn: func(ctx context.Context, system, name, version string) (*depsdev.DependencyGraph, error) {
			return &depsdev.DependencyGraph{
				Nodes: []depsdev.DependencyNode{
					{
						VersionKey: depsdev.VersionKey{System: "npm", Name: "react", Version: "18.2.0"},
					},
				},
			}, nil
		},
		GetPackageMetadataFn: func(ctx context.Context, vk depsdev.VersionKey) (*depsdev.PackageVersionMetadata, error) {
			return &depsdev.PackageVersionMetadata{
				RelatedProjects: []depsdev.RelatedProject{
					{
						ProjectKey:   depsdev.ProjectKey{ID: "github.com/facebook/react"},
						RelationType: "SOURCE_REPO",
					},
				},
			}, nil
		},
		GetScorecardDataFn: func(ctx context.Context, meta *depsdev.PackageVersionMetadata) depsdev.ScorecardInfo {
			return depsdev.ScorecardInfo{
				SourceRepo:   "github.com/facebook/react",
				OpenSSFScore: &score,
			}
		},
	}

	store := &mockStorage{
		GetMapFn: func(ctx context.Context, deps []storage.Dependency) (map[string]storage.Dependency, error) {
			return map[string]storage.Dependency{}, nil
		},
		UpsertFn: func(ctx context.Context, deps []storage.Dependency) error {
			return errors.New("upsert failed")
		},
	}

	manager := &data.DataManager{
		API:           api,
		Store:         store,
		Log:           logrus.New(),
		MaxConcurrent: 5,
	}

	err := manager.RefreshDependencies(context.Background(), "npm", "react", "18.2.0")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "upsert failed")
}
