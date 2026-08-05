package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/lamda/transitops-montreal/backend/internal/api"
	"github.com/lamda/transitops-montreal/backend/internal/config"
	"github.com/lamda/transitops-montreal/backend/internal/db"
	"github.com/lamda/transitops-montreal/backend/internal/ingest"
	"github.com/lamda/transitops-montreal/backend/internal/mock"
)

func main() {
	cfg := config.Load()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	store, err := db.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("database connection failed: %v", err)
	}
	defer store.Close()

	if err := store.EnsureSchema(ctx); err != nil {
		log.Fatalf("database schema setup failed: %v", err)
	}

	if err := store.UpsertRoutes(ctx, mock.Routes()); err != nil {
		log.Fatalf("route seed failed: %v", err)
	}

	if cfg.DataProvider != "mock" {
		log.Printf("DATA_PROVIDER=%s is not implemented yet; using mock provider for this MVP", cfg.DataProvider)
	}

	provider := mock.NewProvider()
	ingestService := ingest.NewService(provider, store)

	count, err := store.SnapshotCount(ctx)
	if err != nil {
		log.Fatalf("snapshot count failed: %v", err)
	}
	if count == 0 {
		result, err := ingestService.RunOnce(ctx)
		if err != nil {
			log.Fatalf("initial mock ingestion failed: %v", err)
		}
		log.Printf("seeded %d mock vehicle snapshots", result.InsertedCount)
	}
	ingestService.StartBackground(ctx, cfg.IngestInterval)

	graphQLHandler, err := api.NewGraphQLHandler(store, ingestService)
	if err != nil {
		log.Fatalf("graphql setup failed: %v", err)
	}

	mux := http.NewServeMux()
	mux.Handle("/graphql", graphQLHandler)

	server := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: mux,
	}

	go func() {
		log.Printf("TransitOps GraphQL API listening on :%s/graphql", cfg.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server failed: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("shutting down server")
}
