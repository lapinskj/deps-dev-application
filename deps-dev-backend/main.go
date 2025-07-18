package main

import (
	"context"
	"database/sql"
	"net/http"
	"os"
	"time"

	"deps-dev/config"
	"deps-dev/data"
	"deps-dev/depsdev"
	"deps-dev/handlers"
	"deps-dev/storage"

	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"

	_ "github.com/mattn/go-sqlite3"
	"github.com/robfig/cron/v3"
	"github.com/sirupsen/logrus"
)

func main() {
	logger := logrus.New()
	logger.SetOutput(os.Stdout)
	logger.SetFormatter(&logrus.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "2006-01-02 15:04:05",
		ForceColors:     true,
		DisableQuote:    true,
		PadLevelText:    true,
	})

	sqlitePath := os.Getenv("SQLITE_PATH")
	if sqlitePath == "" {
		sqlitePath = "./data/app.db"
	}

	db, err := sql.Open("sqlite3", sqlitePath)
	if err != nil {
		logger.Fatalf("failed to open DB: %v", err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)

	store := &storage.Storage{DB: db}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := store.InitSchema(ctx); err != nil {
		logger.Fatalf("failed to initialize schema: %v", err)
	}

	client := &depsdev.DepsDevClient{
		BaseURL:    config.BaseURL,
		HTTPClient: &http.Client{Timeout: 10 * time.Second},
	}

	dm := &data.DataManager{
		Store:         store,
		API:           client,
		Log:           logger,
		MaxConcurrent: config.DefaultMaxConcurrent,
	}

	handler := &handlers.Handler{
		Store:       store,
		DataManager: dm,
		Log:         logger,
	}

	r := chi.NewRouter()
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"http://localhost:5173"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: false,
		MaxAge:           300,
	}))
	r.Use(middleware.Logger)

	r.Get("/dependencies", handler.ListDependencies)
	r.Post("/dependencies", handler.CreateDependency)
	r.Get("/dependencies/{system}/{name}/{version}", handler.GetDependency)
	r.Put("/dependencies/{system}/{name}/{version}", handler.UpdateDependency)
	r.Delete("/dependencies/{system}/{name}/{version}", handler.DeleteDependency)
	r.Post("/dependencies/refresh", handler.RefreshHandler)

	if os.Getenv("WITH_INITIAL_DATA_REFRESH") == "true" {
		if err := dm.RefreshDependencies(ctx, config.DefaultSystem, config.DefaultPackage, config.DefaultVersion); err != nil {
			logger.Fatalf("failed to refresh dependencies: %v", err)
		}
	}

	if os.Getenv("WITH_DAILY_DATA_REFRESH") == "true" {
		c := cron.New()
		_, err := c.AddFunc("0 0 * * *", func() {
			logger.Info("Scheduled refresh triggered")
			ctx := context.Background()
			if err := dm.RefreshDependencies(ctx, config.DefaultSystem, config.DefaultPackage, config.DefaultVersion); err != nil {
				logger.Errorf("scheduled refresh failed: %v", err)
			}
		})
		if err != nil {
			logger.Fatalf("failed to schedule cron: %v", err)
		}
		c.Start()
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	logger.Infof("starting on port %s...", port)
	if err := http.ListenAndServe(":"+port, r); err != nil {
		logger.Fatal(err)
	}
}
