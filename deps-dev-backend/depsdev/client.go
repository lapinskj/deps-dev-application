package depsdev

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

type DepsDevClient struct {
	BaseURL    string
	HTTPClient *http.Client
}

// Fetch dependency graph
func (c *DepsDevClient) GetDependencyGraph(ctx context.Context, system, name, version string) (*DependencyGraph, error) {
	u := fmt.Sprintf("%s/systems/%s/packages/%s/versions/%s:dependencies",
		c.BaseURL, system, url.PathEscape(name), url.PathEscape(version))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch dependency graph: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("dependency graph request failed: %s", resp.Status)
	}

	var graph DependencyGraph
	if err := json.NewDecoder(resp.Body).Decode(&graph); err != nil {
		return nil, fmt.Errorf("failed to decode dependency graph: %w", err)
	}
	return &graph, nil
}

// Fetch metadata for a single dependency
func (c *DepsDevClient) GetPackageMetadata(ctx context.Context, vk VersionKey) (*PackageVersionMetadata, error) {
	u := fmt.Sprintf("%s/systems/%s/packages/%s/versions/%s",
		c.BaseURL, vk.System, url.PathEscape(vk.Name), url.PathEscape(vk.Version))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch package metadata for %s: %w", vk.Name, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("package metadata request failed for %s: %s", vk.Name, resp.Status)
	}

	var meta PackageVersionMetadata
	if err := json.NewDecoder(resp.Body).Decode(&meta); err != nil {
		return nil, fmt.Errorf("failed to decode package metadata: %w", err)
	}
	return &meta, nil
}

// Fetch scorecard data for single project
func (c *DepsDevClient) GetScorecardData(ctx context.Context, meta *PackageVersionMetadata) ScorecardInfo {
	var projectID string
	for _, proj := range meta.RelatedProjects {
		if proj.RelationType == "SOURCE_REPO" {
			projectID = proj.ProjectKey.ID
			break
		}
	}

	var score *float64
	if projectID != "" {
		projectURL := fmt.Sprintf("%s/projects/%s", c.BaseURL, url.PathEscape(projectID))
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, projectURL, nil)
		if err == nil {
			resp, err := c.HTTPClient.Do(req)
			if err == nil && resp.StatusCode == http.StatusOK {
				defer resp.Body.Close()

				var projMeta ProjectMetadata
				if err := json.NewDecoder(resp.Body).Decode(&projMeta); err == nil {
					score = &projMeta.Scorecard.OverallScore
				}
			}
		}
	}

	return ScorecardInfo{
		SourceRepo:   projectID,
		OpenSSFScore: score,
	}
}
