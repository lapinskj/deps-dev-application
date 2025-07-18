package handlers

import (
	"bytes"
	"context"
	"deps-dev/config"
	"deps-dev/storage"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

// Mock Implementations
type mockStore struct {
	ListFilteredFn func(ctx context.Context, name string, minScore *float64) ([]storage.Dependency, error)
	GetFn          func(context.Context, string, string, string) (storage.Dependency, error)
	UpsertFn       func(context.Context, storage.Dependency) error
	DeleteFn       func(context.Context, string, string, string) error
}

func (m *mockStore) ListDependenciesFiltered(ctx context.Context, name string, minScore *float64) ([]storage.Dependency, error) {
	return m.ListFilteredFn(ctx, name, minScore)
}
func (m *mockStore) GetDependency(ctx context.Context, system, name, version string) (storage.Dependency, error) {
	return m.GetFn(ctx, system, name, version)
}
func (m *mockStore) UpsertDependency(ctx context.Context, dep storage.Dependency) error {
	return m.UpsertFn(ctx, dep)
}
func (m *mockStore) DeleteDependency(ctx context.Context, system, name, version string) error {
	return m.DeleteFn(ctx, system, name, version)
}

type mockManager struct {
	RefreshFn func(context.Context, string, string, string) error
}

func (m *mockManager) RefreshDependencies(ctx context.Context, system, name, version string) error {
	return m.RefreshFn(ctx, system, name, version)
}

// Tests
func TestListDependencies(t *testing.T) {
	tests := []struct {
		name           string
		url            string
		mockListFn     func(ctx context.Context, name string, minScore *float64) ([]storage.Dependency, error)
		expectedStatus int
		expectedBody   string
	}{
		{
			name: "no filters (success)",
			url:  "/dependencies",
			mockListFn: func(ctx context.Context, name string, minScore *float64) ([]storage.Dependency, error) {
				assert.Equal(t, "", name)
				assert.Nil(t, minScore)
				return []storage.Dependency{
					{System: "npm", Name: "react", Version: "18.2.0"},
				}, nil
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `[{"system":"npm","name":"react","version":"18.2.0"}]` + "\n",
		},
		{
			name: "filter by name",
			url:  "/dependencies?name=react",
			mockListFn: func(ctx context.Context, name string, minScore *float64) ([]storage.Dependency, error) {
				assert.Equal(t, "react", name)
				assert.Nil(t, minScore)
				return []storage.Dependency{
					{System: "npm", Name: "react", Version: "18.2.0"},
				}, nil
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `[{"system":"npm","name":"react","version":"18.2.0"}]` + "\n",
		},
		{
			name: "filter by name and min_score",
			url:  "/dependencies?name=react&min_score=8.5",
			mockListFn: func(ctx context.Context, name string, minScore *float64) ([]storage.Dependency, error) {
				assert.Equal(t, "react", name)
				assert.NotNil(t, minScore)
				assert.Equal(t, 8.5, *minScore)
				return []storage.Dependency{
					{System: "npm", Name: "react", Version: "18.2.0", OpenSSFScore: float64Ptr(9.1)},
				}, nil
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `[{"system":"npm","name":"react","version":"18.2.0","openssf_score":9.1}]` + "\n",
		},
		{
			name: "invalid min_score",
			url:  "/dependencies?min_score=not-a-number",
			mockListFn: func(ctx context.Context, name string, minScore *float64) ([]storage.Dependency, error) {
				t.Fatal("should not call mock on invalid input")
				return nil, nil
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "invalid min_score value\n",
		},
		{
			name: "store error with filters",
			url:  "/dependencies?name=react",
			mockListFn: func(ctx context.Context, name string, minScore *float64) ([]storage.Dependency, error) {
				return nil, errors.New("db error")
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   "internal server error\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &mockStore{
				ListFilteredFn: tt.mockListFn,
			}
			handler := &Handler{
				Store: store,
				Log:   logrus.New(),
			}

			req := httptest.NewRequest(http.MethodGet, tt.url, nil)
			rr := httptest.NewRecorder()

			handler.ListDependencies(rr, req)

			assert.Equal(t, tt.expectedStatus, rr.Code)
			assert.Equal(t, tt.expectedBody, rr.Body.String())
		})
	}
}

func TestGetDependency(t *testing.T) {
	tests := []struct {
		name           string
		system         string
		packageName    string
		version        string
		mockGetFn      func(ctx context.Context, system, name, version string) (storage.Dependency, error)
		expectedStatus int
		expectedBody   string
	}{
		{
			name:        "valid dependency",
			system:      "npm",
			packageName: "react",
			version:     "18.2.0",
			mockGetFn: func(ctx context.Context, system, name, version string) (storage.Dependency, error) {
				return storage.Dependency{
					System:  system,
					Name:    name,
					Version: version,
				}, nil
			},
			expectedStatus: http.StatusOK,
			expectedBody:   `{"system":"npm","name":"react","version":"18.2.0"}` + "\n",
		},
		{
			name:           "missing path parameters",
			system:         "",
			packageName:    "react",
			version:        "18.2.0",
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "missing path parameters\n",
		},
		{
			name:        "dependency not found",
			system:      "npm",
			packageName: "react",
			version:     "99.9.9",
			mockGetFn: func(ctx context.Context, system, name, version string) (storage.Dependency, error) {
				return storage.Dependency{}, errors.New("not found")
			},
			expectedStatus: http.StatusNotFound,
			expectedBody:   "dependency not found\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &mockStore{
				GetFn: tt.mockGetFn,
			}

			handler := &Handler{
				Store: store,
				Log:   logrus.New(),
			}

			r := chi.NewRouter()
			r.Get("/dependencies/{system}/{name}/{version}", handler.GetDependency)

			url := fmt.Sprintf("/dependencies/%s/%s/%s", tt.system, tt.packageName, tt.version)
			req := httptest.NewRequest(http.MethodGet, url, nil)
			rr := httptest.NewRecorder()

			r.ServeHTTP(rr, req)

			assert.Equal(t, tt.expectedStatus, rr.Code)
			assert.Equal(t, tt.expectedBody, rr.Body.String())
		})
	}
}

func TestCreateDependency(t *testing.T) {
	tests := []struct {
		name           string
		body           string
		existingDep    *storage.Dependency
		getErr         error
		upsertErr      error
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "invalid JSON body",
			body:           `invalid-json`,
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "invalid JSON body\n",
		},
		{
			name:           "missing required fields",
			body:           `{ "system": "npm", "name": "", "version": "" }`,
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "system, name, and version are required\n",
		},
		{
			name: "dependency already exists",
			body: `{ "system": "npm", "name": "react", "version": "18.2.0" }`,
			existingDep: &storage.Dependency{
				System:  "npm",
				Name:    "react",
				Version: "18.2.0",
			},
			getErr:         nil,
			expectedStatus: http.StatusConflict,
			expectedBody:   "dependency already exists\n",
		},
		{
			name:           "upsert failure",
			body:           `{ "system": "npm", "name": "react", "version": "18.2.0" }`,
			getErr:         errors.New("not found"),
			upsertErr:      errors.New("db write failed"),
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   "failed to create dependency\n",
		},
		{
			name:           "success",
			body:           `{ "system": "npm", "name": "react", "version": "18.2.0" }`,
			getErr:         errors.New("not found"),
			upsertErr:      nil,
			expectedStatus: http.StatusCreated,
			expectedBody:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &mockStore{
				GetFn: func(ctx context.Context, system, name, version string) (storage.Dependency, error) {
					if tt.existingDep != nil {
						return *tt.existingDep, tt.getErr
					}
					return storage.Dependency{}, tt.getErr
				},
				UpsertFn: func(ctx context.Context, dep storage.Dependency) error {
					return tt.upsertErr
				},
			}

			handler := &Handler{
				Store: store,
				Log:   logrus.New(),
			}

			req := httptest.NewRequest(http.MethodPost, "/dependencies", bytes.NewBufferString(tt.body))
			rr := httptest.NewRecorder()

			handler.CreateDependency(rr, req)

			assert.Equal(t, tt.expectedStatus, rr.Code)
			assert.Equal(t, tt.expectedBody, rr.Body.String())
		})
	}
}

func TestUpdateDependency(t *testing.T) {
	tests := []struct {
		name           string
		system         string
		packageName    string
		version        string
		body           string
		getFn          func(ctx context.Context, system, name, version string) (storage.Dependency, error)
		upsertFn       func(ctx context.Context, dep storage.Dependency) error
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "invalid JSON body",
			system:         "npm",
			packageName:    "react",
			version:        "18.2.0",
			body:           "not-json",
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "invalid JSON body\n",
		},
		{
			name:        "dependency not found",
			system:      "npm",
			packageName: "react",
			version:     "18.2.0",
			body:        `{}`,
			getFn: func(ctx context.Context, system, name, version string) (storage.Dependency, error) {
				return storage.Dependency{}, errors.New("not found")
			},
			expectedStatus: http.StatusNotFound,
			expectedBody:   "dependency not found\n",
		},
		{
			name:        "upsert failure",
			system:      "npm",
			packageName: "react",
			version:     "18.2.0",
			body:        `{"relation":"direct"}`,
			getFn: func(ctx context.Context, system, name, version string) (storage.Dependency, error) {
				return storage.Dependency{System: system, Name: name, Version: version}, nil
			},
			upsertFn: func(ctx context.Context, dep storage.Dependency) error {
				return errors.New("db error")
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   "failed to update dependency\n",
		},
		{
			name:        "successful update",
			system:      "npm",
			packageName: "react",
			version:     "18.2.0",
			body:        `{"relation":"direct","source_repo":"https://github.com/facebook/react","openssf_score":7.5}`,
			getFn: func(ctx context.Context, system, name, version string) (storage.Dependency, error) {
				return storage.Dependency{System: system, Name: name, Version: version}, nil
			},
			upsertFn: func(ctx context.Context, dep storage.Dependency) error {
				assert.Equal(t, "direct", dep.Relation)
				assert.Equal(t, "https://github.com/facebook/react", dep.SourceRepo)
				assert.NotNil(t, dep.OpenSSFScore)
				assert.Equal(t, 7.5, *dep.OpenSSFScore)
				return nil
			},
			expectedStatus: http.StatusOK,
			expectedBody:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &mockStore{
				GetFn:    tt.getFn,
				UpsertFn: tt.upsertFn,
			}
			handler := &Handler{
				Store: store,
				Log:   logrus.New(),
			}

			req := httptest.NewRequest(http.MethodPut, "/", bytes.NewBufferString(tt.body))
			rr := httptest.NewRecorder()

			routeCtx := chi.NewRouteContext()
			routeCtx.URLParams.Add("system", tt.system)
			routeCtx.URLParams.Add("name", tt.packageName)
			routeCtx.URLParams.Add("version", tt.version)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, routeCtx))

			handler.UpdateDependency(rr, req)

			assert.Equal(t, tt.expectedStatus, rr.Code)
			assert.Equal(t, tt.expectedBody, rr.Body.String())
		})
	}
}

func TestDeleteDependency(t *testing.T) {
	tests := []struct {
		name           string
		system         string
		packageName    string
		version        string
		deleteFn       func(ctx context.Context, system, name, version string) error
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "missing path parameters",
			system:         "",
			packageName:    "react",
			version:        "18.2.0",
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "missing path parameters\n",
		},
		{
			name:        "delete fails",
			system:      "npm",
			packageName: "react",
			version:     "18.2.0",
			deleteFn: func(ctx context.Context, system, name, version string) error {
				return errors.New("delete error")
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   "failed to delete dependency\n",
		},
		{
			name:        "successful delete",
			system:      "npm",
			packageName: "react",
			version:     "18.2.0",
			deleteFn: func(ctx context.Context, system, name, version string) error {
				return nil
			},
			expectedStatus: http.StatusNoContent,
			expectedBody:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &mockStore{
				DeleteFn: tt.deleteFn,
			}
			handler := &Handler{
				Store: store,
				Log:   logrus.New(),
			}

			req := httptest.NewRequest(http.MethodDelete, "/", nil)
			rr := httptest.NewRecorder()

			routeCtx := chi.NewRouteContext()
			routeCtx.URLParams.Add("system", tt.system)
			routeCtx.URLParams.Add("name", tt.packageName)
			routeCtx.URLParams.Add("version", tt.version)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, routeCtx))

			handler.DeleteDependency(rr, req)

			assert.Equal(t, tt.expectedStatus, rr.Code)
			assert.Equal(t, tt.expectedBody, rr.Body.String())
		})
	}
}

func TestRefreshHandler(t *testing.T) {
	tests := []struct {
		name           string
		refreshFn      func(ctx context.Context, system, name, version string) error
		expectedStatus int
		expectedBody   string
	}{
		{
			name: "refresh fails",
			refreshFn: func(ctx context.Context, system, name, version string) error {
				assert.Equal(t, config.DefaultSystem, system)
				assert.Equal(t, config.DefaultPackage, name)
				assert.Equal(t, config.DefaultVersion, version)
				return errors.New("refresh error")
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   "failed to refresh dependencies\n",
		},
		{
			name: "refresh succeeds",
			refreshFn: func(ctx context.Context, system, name, version string) error {
				assert.Equal(t, config.DefaultSystem, system)
				assert.Equal(t, config.DefaultPackage, name)
				assert.Equal(t, config.DefaultVersion, version)
				return nil
			},
			expectedStatus: http.StatusOK,
			expectedBody:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := &mockManager{
				RefreshFn: tt.refreshFn,
			}

			handler := &Handler{
				DataManager: manager,
				Log:         logrus.New(),
			}

			req := httptest.NewRequest(http.MethodPost, "/refresh", nil)
			rr := httptest.NewRecorder()

			handler.RefreshHandler(rr, req)

			assert.Equal(t, tt.expectedStatus, rr.Code)
			assert.Equal(t, tt.expectedBody, rr.Body.String())
		})
	}
}

func float64Ptr(f float64) *float64 {
	return &f
}
