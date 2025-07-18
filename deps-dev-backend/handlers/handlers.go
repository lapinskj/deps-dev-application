package handlers

import (
	"context"
	"deps-dev/config"
	"deps-dev/storage"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/sirupsen/logrus"
)

type Storage interface {
	ListDependenciesFiltered(ctx context.Context, name string, minScore *float64) ([]storage.Dependency, error)
	GetDependency(ctx context.Context, system, name, version string) (storage.Dependency, error)
	UpsertDependency(ctx context.Context, dep storage.Dependency) error
	DeleteDependency(ctx context.Context, system, name, version string) error
}

type DataManager interface {
	RefreshDependencies(ctx context.Context, system, name, version string) error
}

type Handler struct {
	Store       Storage
	DataManager DataManager
	Log         *logrus.Logger
}

func (h *Handler) ListDependencies(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")
	minScoreStr := r.URL.Query().Get("min_score")

	var minScore *float64
	if minScoreStr != "" {
		if score, err := strconv.ParseFloat(minScoreStr, 64); err == nil {
			minScore = &score
		} else {
			http.Error(w, "invalid min_score value", http.StatusBadRequest)
			return
		}
	}

	deps, err := h.Store.ListDependenciesFiltered(r.Context(), name, minScore)
	if err != nil {
		h.Log.WithError(err).Error("listing dependencies with filters")
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(deps); err != nil {
		h.Log.WithError(err).Error("encoding dependencies list response")
	}
}

func (h *Handler) GetDependency(w http.ResponseWriter, r *http.Request) {
	system := chi.URLParam(r, "system")
	name := chi.URLParam(r, "name")
	version := chi.URLParam(r, "version")

	if system == "" || name == "" || version == "" {
		http.Error(w, "missing path parameters", http.StatusBadRequest)
		return
	}

	dep, err := h.Store.GetDependency(r.Context(), system, name, version)
	if err != nil {
		h.Log.WithFields(logrus.Fields{
			"system":  system,
			"name":    name,
			"version": version,
		}).WithError(err).Error("fetching dependency")
		http.Error(w, "dependency not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(dep); err != nil {
		h.Log.WithError(err).Error("encoding single dependency response")
	}
}

func (h *Handler) CreateDependency(w http.ResponseWriter, r *http.Request) {
	var dep storage.Dependency
	if err := json.NewDecoder(r.Body).Decode(&dep); err != nil {
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return
	}

	if dep.System == "" || dep.Name == "" || dep.Version == "" {
		http.Error(w, "system, name, and version are required", http.StatusBadRequest)
		return
	}

	existing, err := h.Store.GetDependency(r.Context(), dep.System, dep.Name, dep.Version)
	if err == nil && existing.Name != "" {
		http.Error(w, "dependency already exists", http.StatusConflict)
		return
	}

	if err := h.Store.UpsertDependency(r.Context(), dep); err != nil {
		h.Log.WithError(err).Error("creating dependency")
		http.Error(w, "failed to create dependency", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

type DependencyUpdateRequest struct {
	Relation     *string  `json:"relation,omitempty"`
	SourceRepo   *string  `json:"source_repo,omitempty"`
	OpenSSFScore *float64 `json:"openssf_score,omitempty"`
}

func (h *Handler) UpdateDependency(w http.ResponseWriter, r *http.Request) {
	system := chi.URLParam(r, "system")
	name := chi.URLParam(r, "name")
	version := chi.URLParam(r, "version")

	var input DependencyUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return
	}

	current, err := h.Store.GetDependency(r.Context(), system, name, version)
	if err != nil {
		http.Error(w, "dependency not found", http.StatusNotFound)
		return
	}

	if input.Relation != nil {
		current.Relation = *input.Relation
	}
	if input.SourceRepo != nil {
		current.SourceRepo = *input.SourceRepo
	}
	if input.OpenSSFScore != nil {
		current.OpenSSFScore = input.OpenSSFScore
	}

	if err := h.Store.UpsertDependency(r.Context(), current); err != nil {
		h.Log.WithError(err).Error("updating dependency")
		http.Error(w, "failed to update dependency", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *Handler) DeleteDependency(w http.ResponseWriter, r *http.Request) {
	system := chi.URLParam(r, "system")
	name := chi.URLParam(r, "name")
	version := chi.URLParam(r, "version")

	if system == "" || name == "" || version == "" {
		http.Error(w, "missing path parameters", http.StatusBadRequest)
		return
	}

	if err := h.Store.DeleteDependency(r.Context(), system, name, version); err != nil {
		h.Log.WithError(err).Error("deleting dependency")
		http.Error(w, "failed to delete dependency", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) RefreshHandler(w http.ResponseWriter, r *http.Request) {
	err := h.DataManager.RefreshDependencies(r.Context(), config.DefaultSystem, config.DefaultPackage, config.DefaultVersion)
	if err != nil {
		h.Log.WithError(err).Error("failed to refresh dependencies")
		http.Error(w, "failed to refresh dependencies", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
