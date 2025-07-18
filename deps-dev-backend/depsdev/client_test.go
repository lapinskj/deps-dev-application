package depsdev

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestGetDependencyGraph(t *testing.T) {
	tests := []struct {
		name          string
		statusCode    int
		body          any
		expectError   bool
		expectedGraph *DependencyGraph
	}{
		{
			name:       "Valid response",
			statusCode: http.StatusOK,
			body: DependencyGraph{
				Nodes: []DependencyNode{
					{VersionKey: VersionKey{System: "npm", Name: "pkg", Version: "1.0.0"}},
				},
			},
			expectError: false,
			expectedGraph: &DependencyGraph{
				Nodes: []DependencyNode{
					{VersionKey: VersionKey{System: "npm", Name: "pkg", Version: "1.0.0"}},
				},
			},
		},
		{
			name:          "Non-200 status",
			statusCode:    http.StatusNotFound,
			body:          nil,
			expectError:   true,
			expectedGraph: nil,
		},
		{
			name:          "Invalid JSON",
			statusCode:    http.StatusOK,
			body:          "invalid-json",
			expectError:   true,
			expectedGraph: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				if tt.body != nil {
					switch v := tt.body.(type) {
					case string:
						fmt.Fprint(w, v)
					default:
						_ = json.NewEncoder(w).Encode(v)
					}
				}
			}))
			defer server.Close()

			client := &DepsDevClient{
				BaseURL:    server.URL,
				HTTPClient: http.DefaultClient,
			}

			graph, err := client.GetDependencyGraph(context.Background(), "npm", "react", "18.2.0")

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
				if graph != nil {
					t.Errorf("expected nil graph, got %v", graph)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
					return
				}
				if !reflect.DeepEqual(graph, tt.expectedGraph) {
					t.Errorf("expected graph %+v, got %+v", tt.expectedGraph, graph)
				}
			}
		})
	}
}

func TestGetPackageMetadata(t *testing.T) {
	tests := []struct {
		name             string
		statusCode       int
		body             any
		expectError      bool
		expectedMetadata *PackageVersionMetadata
	}{
		{
			name:       "Valid metadata",
			statusCode: http.StatusOK,
			body: PackageVersionMetadata{
				RelatedProjects: []RelatedProject{
					{
						ProjectKey: struct {
							ID string `json:"id"`
						}{ID: "github.com/facebook/react"},
						RelationType: "ISSUE_TRACKER",
					},
					{
						ProjectKey: struct {
							ID string `json:"id"`
						}{ID: "github.com/facebook/react"},
						RelationType: "SOURCE_REPO",
					},
				},
			},
			expectError: false,
			expectedMetadata: &PackageVersionMetadata{
				RelatedProjects: []RelatedProject{
					{
						ProjectKey: struct {
							ID string `json:"id"`
						}{ID: "github.com/facebook/react"},
						RelationType: "ISSUE_TRACKER",
					},
					{
						ProjectKey: struct {
							ID string `json:"id"`
						}{ID: "github.com/facebook/react"},
						RelationType: "SOURCE_REPO",
					},
				},
			},
		},
		{
			name:             "Invalid JSON",
			statusCode:       http.StatusOK,
			body:             "bad-json",
			expectError:      true,
			expectedMetadata: nil,
		},
		{
			name:             "Not found",
			statusCode:       http.StatusNotFound,
			body:             nil,
			expectError:      true,
			expectedMetadata: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				if tt.body != nil {
					switch v := tt.body.(type) {
					case string:
						fmt.Fprint(w, v)
					default:
						_ = json.NewEncoder(w).Encode(v)
					}
				}
			}))
			defer server.Close()

			client := &DepsDevClient{
				BaseURL:    server.URL,
				HTTPClient: http.DefaultClient,
			}

			meta, err := client.GetPackageMetadata(context.Background(), VersionKey{
				System:  "npm",
				Name:    "react",
				Version: "18.2.0",
			})

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
				if meta != nil {
					t.Errorf("expected nil metadata, got %v", meta)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
					return
				}
				if !reflect.DeepEqual(meta, tt.expectedMetadata) {
					t.Errorf("expected metadata %+v, got %+v", tt.expectedMetadata, meta)
				}
			}
		})
	}
}

func TestGetScorecardData(t *testing.T) {
	projectID := "github.com/facebook/react"

	tests := []struct {
		name             string
		statusCode       int
		body             any
		expectedScore    *float64
		expectedMetadata *PackageVersionMetadata
	}{
		{
			name:       "Valid project with score",
			statusCode: http.StatusOK,
			body: ProjectMetadata{
				Scorecard: struct {
					OverallScore float64 `json:"overallScore"`
				}{OverallScore: 9.1},
			},
			expectedScore: float64Ptr(9.1),
			expectedMetadata: &PackageVersionMetadata{
				RelatedProjects: []RelatedProject{
					{
						ProjectKey:   ProjectKey{ID: projectID},
						RelationType: "ISSUE_TRACKER",
					},
					{
						ProjectKey:   ProjectKey{ID: projectID},
						RelationType: "SOURCE_REPO", // This is the one used
					},
				},
			},
		},
		{
			name:          "Project not found",
			statusCode:    http.StatusNotFound,
			body:          nil,
			expectedScore: nil,
			expectedMetadata: &PackageVersionMetadata{
				RelatedProjects: []RelatedProject{
					{
						ProjectKey:   ProjectKey{ID: projectID},
						RelationType: "SOURCE_REPO",
					},
				},
			},
		},
		{
			name:          "Invalid JSON response",
			statusCode:    http.StatusOK,
			body:          "bad-json",
			expectedScore: nil,
			expectedMetadata: &PackageVersionMetadata{
				RelatedProjects: []RelatedProject{
					{
						ProjectKey:   ProjectKey{ID: projectID},
						RelationType: "SOURCE_REPO",
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != fmt.Sprintf("/projects/%s", projectID) {
					t.Errorf("unexpected request path: %s", r.URL.Path)
				}
				w.WriteHeader(tt.statusCode)
				if tt.body != nil {
					switch v := tt.body.(type) {
					case string:
						fmt.Fprint(w, v)
					default:
						_ = json.NewEncoder(w).Encode(v)
					}
				}
			}))
			defer server.Close()

			client := &DepsDevClient{
				BaseURL:    server.URL,
				HTTPClient: http.DefaultClient,
			}

			result := client.GetScorecardData(context.Background(), tt.expectedMetadata)

			if tt.expectedScore == nil && result.OpenSSFScore != nil {
				t.Errorf("expected nil score, got %v", *result.OpenSSFScore)
			}
			if tt.expectedScore != nil {
				if result.OpenSSFScore == nil {
					t.Errorf("expected score %v, got nil", *tt.expectedScore)
				} else if *result.OpenSSFScore != *tt.expectedScore {
					t.Errorf("expected score %v, got %v", *tt.expectedScore, *result.OpenSSFScore)
				}
			}
		})
	}
}

func float64Ptr(f float64) *float64 {
	return &f
}
