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

CREATE TABLE IF NOT EXISTS route_shape_points (
	route_id TEXT NOT NULL REFERENCES routes(id) ON DELETE CASCADE,
	point_index INTEGER NOT NULL,
	latitude DOUBLE PRECISION NOT NULL,
	longitude DOUBLE PRECISION NOT NULL,
	PRIMARY KEY (route_id, point_index)
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

CREATE INDEX IF NOT EXISTS idx_vehicle_snapshots_route_vehicle_timestamp
	ON vehicle_snapshots (route_id, vehicle_id, timestamp DESC, id DESC);
`)
	return err
}

func (s *Store) UpsertRoutes(ctx context.Context, routes []models.Route) error {
	batch := &pgx.Batch{}
	statementCount := 0
	for _, route := range routes {
		batch.Queue(`
INSERT INTO routes (id, short_name, long_name, color)
VALUES ($1, $2, $3, $4)
ON CONFLICT (id) DO UPDATE SET
	short_name = EXCLUDED.short_name,
	long_name = EXCLUDED.long_name,
	color = EXCLUDED.color;
`, route.ID, route.ShortName, route.LongName, route.Color)
		statementCount++
	}

	for _, route := range routes {
		batch.Queue(`DELETE FROM route_shape_points WHERE route_id = $1;`, route.ID)
		statementCount++

		for pointIndex, point := range route.Shape {
			batch.Queue(`
INSERT INTO route_shape_points (route_id, point_index, latitude, longitude)
VALUES ($1, $2, $3, $4);
`, route.ID, pointIndex, point.Latitude, point.Longitude)
			statementCount++
		}
	}

	results := s.pool.SendBatch(ctx, batch)
	defer results.Close()

	for statementIndex := 0; statementIndex < statementCount; statementIndex++ {
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

	if err := rows.Err(); err != nil {
		return nil, err
	}

	shapeRows, err := s.pool.Query(ctx, `
SELECT route_id, latitude, longitude
FROM route_shape_points
ORDER BY route_id, point_index;
`)
	if err != nil {
		return nil, err
	}
	defer shapeRows.Close()

	shapesByRouteID := make(map[string][]models.RouteShapePoint)
	for shapeRows.Next() {
		var routeID string
		var point models.RouteShapePoint
		if err := shapeRows.Scan(&routeID, &point.Latitude, &point.Longitude); err != nil {
			return nil, err
		}
		shapesByRouteID[routeID] = append(shapesByRouteID[routeID], point)
	}
	if err := shapeRows.Err(); err != nil {
		return nil, err
	}

	for index := range routes {
		routes[index].Shape = shapesByRouteID[routes[index].ID]
	}

	return routes, nil
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

// VehicleSnapshotsInRange includes each vehicle's last observation before the
// requested window so callers can reconstruct an as-of state at its start.
func (s *Store) VehicleSnapshotsInRange(
	ctx context.Context,
	routeID string,
	from time.Time,
	to time.Time,
) ([]models.VehicleSnapshot, error) {
	rows, err := s.pool.Query(ctx, `
WITH baseline AS (
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
	WHERE route_id = $1 AND timestamp < $2
	ORDER BY vehicle_id, timestamp DESC, id DESC
),
windowed AS (
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
	FROM vehicle_snapshots
	WHERE route_id = $1 AND timestamp >= $2 AND timestamp <= $3
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
FROM (
	SELECT * FROM baseline
	UNION ALL
	SELECT * FROM windowed
) AS snapshots
ORDER BY timestamp, id;
`, routeID, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	snapshots := make([]models.VehicleSnapshot, 0)
	for rows.Next() {
		snapshot, err := scanVehicleSnapshot(rows)
		if err != nil {
			return nil, err
		}
		snapshots = append(snapshots, snapshot)
	}

	return snapshots, rows.Err()
}

func (s *Store) VehicleTrails(ctx context.Context, routeID string, since time.Time, maxPoints int) ([]models.VehicleTrail, error) {
	if maxPoints <= 0 {
		return []models.VehicleTrail{}, nil
	}

	rows, err := s.pool.Query(ctx, `
WITH recent AS (
	SELECT
		vehicle_id,
		latitude,
		longitude,
		timestamp,
		ROW_NUMBER() OVER (
			PARTITION BY vehicle_id
			ORDER BY timestamp DESC, id DESC
		) AS snapshot_rank
	FROM vehicle_snapshots
	WHERE route_id = $1 AND timestamp >= $2
)
SELECT vehicle_id, latitude, longitude, timestamp
FROM recent
WHERE snapshot_rank <= $3
ORDER BY vehicle_id, timestamp, snapshot_rank DESC;
`, routeID, since, maxPoints)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	trails := make([]models.VehicleTrail, 0)
	currentVehicleID := ""
	for rows.Next() {
		var vehicleID string
		var point models.VehicleTrailPoint
		if err := rows.Scan(&vehicleID, &point.Latitude, &point.Longitude, &point.Timestamp); err != nil {
			return nil, err
		}

		if vehicleID != currentVehicleID {
			trails = append(trails, models.VehicleTrail{VehicleID: vehicleID})
			currentVehicleID = vehicleID
		}

		trailIndex := len(trails) - 1
		trails[trailIndex].Points = append(trails[trailIndex].Points, point)
	}

	return trails, rows.Err()
}

func (s *Store) SnapshotCount(ctx context.Context) (int64, error) {
	var count int64
	if err := s.pool.QueryRow(ctx, `SELECT COUNT(*) FROM vehicle_snapshots;`).Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

func (s *Store) HasSnapshotInRange(ctx context.Context, from time.Time, to time.Time) (bool, error) {
	var exists bool
	if err := s.pool.QueryRow(ctx, `
SELECT EXISTS(
	SELECT 1
	FROM vehicle_snapshots
WHERE timestamp >= $1 AND timestamp <= $2
);
`, from, to).Scan(&exists); err != nil {
		return false, err
	}
	return exists, nil
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
