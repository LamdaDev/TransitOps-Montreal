package db

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/lamda/transitops-montreal/backend/internal/models"
)

type Store struct {
	pool *pgxpool.Pool
}

func Connect(ctx context.Context, databaseURL string) (*Store, error) {
	var lastErr error

	for attempt := 1; attempt <= 20; attempt++ {
		pool, err := pgxpool.New(ctx, databaseURL)
		if err == nil {
			if pingErr := pool.Ping(ctx); pingErr == nil {
				return &Store{pool: pool}, nil
			} else {
				lastErr = pingErr
			}
			pool.Close()
		} else {
			lastErr = err
		}

		time.Sleep(1 * time.Second)
	}

	return nil, fmt.Errorf("connect to postgres: %w", lastErr)
}

func (s *Store) Close() {
	s.pool.Close()
}

func (s *Store) EnsureSchema(ctx context.Context) error {
	_, err := s.pool.Exec(ctx, `
CREATE TABLE IF NOT EXISTS routes (
	id TEXT PRIMARY KEY,
	short_name TEXT NOT NULL,
	long_name TEXT NOT NULL,
	color TEXT
);

CREATE TABLE IF NOT EXISTS vehicle_snapshots (
	id BIGSERIAL PRIMARY KEY,
	vehicle_id TEXT NOT NULL,
	route_id TEXT NOT NULL REFERENCES routes(id),
	trip_id TEXT,
	latitude DOUBLE PRECISION NOT NULL,
	longitude DOUBLE PRECISION NOT NULL,
	speed DOUBLE PRECISION,
	route_progress DOUBLE PRECISION,
	timestamp TIMESTAMPTZ NOT NULL,
	source TEXT NOT NULL,
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_vehicle_snapshots_route_timestamp
	ON vehicle_snapshots (route_id, timestamp DESC);

CREATE INDEX IF NOT EXISTS idx_vehicle_snapshots_vehicle_timestamp
	ON vehicle_snapshots (vehicle_id, timestamp DESC);
`)
	return err
}

func (s *Store) UpsertRoutes(ctx context.Context, routes []models.Route) error {
	batch := &pgx.Batch{}
	for _, route := range routes {
		batch.Queue(`
INSERT INTO routes (id, short_name, long_name, color)
VALUES ($1, $2, $3, $4)
ON CONFLICT (id) DO UPDATE SET
	short_name = EXCLUDED.short_name,
	long_name = EXCLUDED.long_name,
	color = EXCLUDED.color;
`, route.ID, route.ShortName, route.LongName, route.Color)
	}

	results := s.pool.SendBatch(ctx, batch)
	defer results.Close()

	for range routes {
		if _, err := results.Exec(); err != nil {
			return err
		}
	}

	return nil
}

func (s *Store) Routes(ctx context.Context) ([]models.Route, error) {
	rows, err := s.pool.Query(ctx, `
SELECT id, short_name, long_name, color
FROM routes
ORDER BY CASE id
	WHEN '24' THEN 1
	WHEN '55' THEN 2
	WHEN '80' THEN 3
	WHEN '165' THEN 4
	WHEN '470' THEN 5
	ELSE 99
END, id;
`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var routes []models.Route
	for rows.Next() {
		var route models.Route
		var color pgtype.Text

		if err := rows.Scan(&route.ID, &route.ShortName, &route.LongName, &color); err != nil {
			return nil, err
		}
		if color.Valid {
			route.Color = &color.String
		}

		routes = append(routes, route)
	}

	return routes, rows.Err()
}

func (s *Store) InsertVehicleSnapshots(ctx context.Context, snapshots []models.VehicleSnapshot) error {
	if len(snapshots) == 0 {
		return nil
	}

	batch := &pgx.Batch{}
	for _, snapshot := range snapshots {
		batch.Queue(`
INSERT INTO vehicle_snapshots (
	vehicle_id,
	route_id,
	trip_id,
	latitude,
	longitude,
	speed,
	route_progress,
	timestamp,
	source
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9);
`, snapshot.VehicleID, snapshot.RouteID, snapshot.TripID, snapshot.Latitude, snapshot.Longitude, snapshot.Speed, snapshot.RouteProgress, snapshot.Timestamp, snapshot.Source)
	}

	results := s.pool.SendBatch(ctx, batch)
	defer results.Close()

	for range snapshots {
		if _, err := results.Exec(); err != nil {
			return err
		}
	}

	return nil
}

func (s *Store) LatestVehicleSnapshots(ctx context.Context, routeID string) ([]models.VehicleSnapshot, error) {
	rows, err := s.pool.Query(ctx, `
WITH latest AS (
	SELECT DISTINCT ON (vehicle_id)
		id,
		vehicle_id,
		route_id,
		trip_id,
		latitude,
		longitude,
		speed,
		route_progress,
		timestamp,
		source,
		created_at
	FROM vehicle_snapshots
	WHERE route_id = $1
	ORDER BY vehicle_id, timestamp DESC
)
SELECT
	id,
	vehicle_id,
	route_id,
	trip_id,
	latitude,
	longitude,
	speed,
	route_progress,
	timestamp,
	source,
	created_at
FROM latest
ORDER BY COALESCE(route_progress, 0), vehicle_id;
`, routeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var snapshots []models.VehicleSnapshot
	for rows.Next() {
		snapshot, err := scanVehicleSnapshot(rows)
		if err != nil {
			return nil, err
		}
		snapshots = append(snapshots, snapshot)
	}

	return snapshots, rows.Err()
}

func (s *Store) SnapshotCount(ctx context.Context) (int64, error) {
	var count int64
	if err := s.pool.QueryRow(ctx, `SELECT COUNT(*) FROM vehicle_snapshots;`).Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanVehicleSnapshot(row rowScanner) (models.VehicleSnapshot, error) {
	var snapshot models.VehicleSnapshot
	var tripID pgtype.Text
	var speed pgtype.Float8
	var routeProgress pgtype.Float8

	err := row.Scan(
		&snapshot.ID,
		&snapshot.VehicleID,
		&snapshot.RouteID,
		&tripID,
		&snapshot.Latitude,
		&snapshot.Longitude,
		&speed,
		&routeProgress,
		&snapshot.Timestamp,
		&snapshot.Source,
		&snapshot.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return snapshot, err
		}
		return snapshot, fmt.Errorf("scan vehicle snapshot: %w", err)
	}

	if tripID.Valid {
		snapshot.TripID = &tripID.String
	}
	if speed.Valid {
		snapshot.Speed = &speed.Float64
	}
	if routeProgress.Valid {
		snapshot.RouteProgress = &routeProgress.Float64
	}

	return snapshot, nil
}
