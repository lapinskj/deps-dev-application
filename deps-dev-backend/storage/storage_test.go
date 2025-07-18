package storage_test

import (
	"context"
	"database/sql"
	"deps-dev/storage"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
)

func setupTestDB(t *testing.T) (*sql.DB, *storage.Storage) {
	db, err := sql.Open("sqlite3", ":memory:")
	assert.NoError(t, err)

	store := &storage.Storage{DB: db}
	err = store.InitSchema(context.Background())
	assert.NoError(t, err)

	return db, store
}

func TestUpsertAndGetDependency(t *testing.T) {
	_, store := setupTestDB(t)

	dep := storage.Dependency{
		System:       "npm",
		Name:         "react",
		Version:      "18.2.0",
		Relation:     "direct",
		SourceRepo:   "github.com/facebook/react",
		OpenSSFScore: floatPtr(9.1),
	}

	err := store.UpsertDependency(context.Background(), dep)
	assert.NoError(t, err)

	got, err := store.GetDependency(context.Background(), "npm", "react", "18.2.0")
	assert.NoError(t, err)
	assert.Equal(t, dep.System, got.System)
	assert.Equal(t, *dep.OpenSSFScore, *got.OpenSSFScore)
}

func TestListDependencies(t *testing.T) {
	_, store := setupTestDB(t)

	score := 9.1
	deps := []storage.Dependency{
		{System: "npm", Name: "react", Version: "18.2.0", OpenSSFScore: &score},
		{System: "npm", Name: "express", Version: "4.17.1"},
	}

	for _, d := range deps {
		assert.NoError(t, store.UpsertDependency(context.Background(), d))
	}

	t.Run("list all dependencies", func(t *testing.T) {
		list, err := store.ListDependenciesFiltered(context.Background(), "", nil)
		assert.NoError(t, err)
		assert.Len(t, list, 2)
	})

	t.Run("filter by name", func(t *testing.T) {
		list, err := store.ListDependenciesFiltered(context.Background(), "react", nil)
		assert.NoError(t, err)
		assert.Len(t, list, 1)
		assert.Equal(t, "react", list[0].Name)
	})

	t.Run("filter by min_score", func(t *testing.T) {
		min := 8.0
		list, err := store.ListDependenciesFiltered(context.Background(), "", &min)
		assert.NoError(t, err)
		assert.Len(t, list, 1)
		assert.Equal(t, "react", list[0].Name)
	})

	t.Run("filter by name and min_score", func(t *testing.T) {
		min := 8.0
		list, err := store.ListDependenciesFiltered(context.Background(), "react", &min)
		assert.NoError(t, err)
		assert.Len(t, list, 1)
		assert.Equal(t, "react", list[0].Name)
	})

	t.Run("no match for filters", func(t *testing.T) {
		min := 9.5
		list, err := store.ListDependenciesFiltered(context.Background(), "nonexistent", &min)
		assert.NoError(t, err)
		assert.Len(t, list, 0)
	})
}

func TestUpsertDependencies(t *testing.T) {
	_, store := setupTestDB(t)

	deps := []storage.Dependency{
		{System: "npm", Name: "axios", Version: "1.3.0", SourceRepo: "gh/axios/axios"},
		{System: "npm", Name: "lodash", Version: "4.17.21", Relation: "indirect"},
	}

	err := store.UpsertDependencies(context.Background(), deps)
	assert.NoError(t, err)

	list, err := store.ListDependenciesFiltered(context.Background(), "", nil)
	assert.NoError(t, err)
	assert.Len(t, list, 2)
}

func TestDeleteDependency(t *testing.T) {
	_, store := setupTestDB(t)

	dep := storage.Dependency{System: "npm", Name: "vue", Version: "3.0.0"}
	assert.NoError(t, store.UpsertDependency(context.Background(), dep))

	err := store.DeleteDependency(context.Background(), "npm", "vue", "3.0.0")
	assert.NoError(t, err)

	_, err = store.GetDependency(context.Background(), "npm", "vue", "3.0.0")
	assert.Error(t, err)
}

func TestGetDependenciesMap(t *testing.T) {
	_, store := setupTestDB(t)

	dbDep := storage.Dependency{
		System:       "npm",
		Name:         "react",
		Version:      "18.2.0",
		SourceRepo:   "gh/facebook/react",
		OpenSSFScore: floatPtr(7.8),
	}
	assert.NoError(t, store.UpsertDependency(context.Background(), dbDep))

	input := []storage.Dependency{
		{System: "npm", Name: "react", Version: "18.2.0"},
		{System: "npm", Name: "nonexistent", Version: "1.0.0"},
	}

	m, err := store.GetDependenciesMap(context.Background(), input)
	assert.NoError(t, err)
	assert.Len(t, m, 1)

	key := "npm|react|18.2.0"
	assert.Equal(t, dbDep.SourceRepo, m[key].SourceRepo)
}

func floatPtr(f float64) *float64 {
	return &f
}
