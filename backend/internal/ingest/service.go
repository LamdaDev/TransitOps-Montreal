package ingest

import (
	"context"
	"log"
	"time"

	"github.com/lamda/transitops-montreal/backend/internal/models"
)

type SnapshotProvider interface {
	Name() string
	FetchSnapshots(ctx context.Context) ([]models.VehicleSnapshot, error)
}

type SnapshotStore interface {
	InsertVehicleSnapshots(ctx context.Context, snapshots []models.VehicleSnapshot) error
}

type Service struct {
	provider SnapshotProvider
	store    SnapshotStore
}

func NewService(provider SnapshotProvider, store SnapshotStore) *Service {
	return &Service{
		provider: provider,
		store:    store,
	}
}

func (s *Service) RunOnce(ctx context.Context) (models.IngestResult, error) {
	snapshots, err := s.provider.FetchSnapshots(ctx)
	if err != nil {
		return models.IngestResult{}, err
	}

	if err := s.store.InsertVehicleSnapshots(ctx, snapshots); err != nil {
		return models.IngestResult{}, err
	}

	return models.IngestResult{
		InsertedCount: len(snapshots),
		Timestamp:     time.Now().UTC(),
		Source:        s.provider.Name(),
	}, nil
}

func (s *Service) StartBackground(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)

	go func() {
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				result, err := s.RunOnce(ctx)
				if err != nil {
					log.Printf("mock ingestion failed: %v", err)
					continue
				}
				log.Printf("ingested %d %s snapshots", result.InsertedCount, result.Source)
			}
		}
	}()
}
