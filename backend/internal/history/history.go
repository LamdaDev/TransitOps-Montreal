package history

import (
	"fmt"
	"sort"
	"time"

	"github.com/lamda/transitops-montreal/backend/internal/metrics"
	"github.com/lamda/transitops-montreal/backend/internal/models"
)

const defaultBucket = time.Minute

// BuildRouteHistory reconstructs the latest known route state at each bucket.
// The caller supplies one pre-window snapshot per vehicle when available.
func BuildRouteHistory(
	routeID string,
	snapshots []models.VehicleSnapshot,
	from time.Time,
	to time.Time,
	bucket time.Duration,
) models.RouteHistory {
	if bucket <= 0 {
		bucket = defaultBucket
	}

	points := buildHistoryPoints(routeID, snapshots, from, to, bucket)
	return models.RouteHistory{
		RouteID:       routeID,
		From:          from,
		To:            to,
		BucketSeconds: int(bucket.Seconds()),
		Points:        points,
		Events:        deriveEvents(points),
	}
}

// BuildRouteReplay uses the same as-of reconstruction as the timeline, at a
// smaller interval suitable for client-side map playback.
func BuildRouteReplay(
	routeID string,
	snapshots []models.VehicleSnapshot,
	from time.Time,
	to time.Time,
	step time.Duration,
) models.RouteReplay {
	if step <= 0 {
		step = 15 * time.Second
	}

	frames := make([]models.ReplayFrame, 0)
	forEachState(snapshots, from, to, step, func(timestamp time.Time, state []models.VehicleSnapshot) {
		analysis := metrics.Analyze(routeID, state, timestamp)
		frames = append(frames, models.ReplayFrame{
			Timestamp: timestamp,
			HasData:   len(state) > 0,
			Vehicles:  analysis.Vehicles,
			Metrics:   analysis.Metrics,
			Insights:  analysis.Insights,
		})
	})

	return models.RouteReplay{
		RouteID:     routeID,
		From:        from,
		To:          to,
		StepSeconds: int(step.Seconds()),
		Frames:      frames,
	}
}

func buildHistoryPoints(
	routeID string,
	snapshots []models.VehicleSnapshot,
	from time.Time,
	to time.Time,
	bucket time.Duration,
) []models.RouteHistoryPoint {
	points := make([]models.RouteHistoryPoint, 0)
	forEachState(snapshots, from, to, bucket, func(timestamp time.Time, state []models.VehicleSnapshot) {
		analysis := metrics.Analyze(routeID, state, timestamp)
		points = append(points, models.RouteHistoryPoint{
			Timestamp: timestamp,
			HasData:   len(state) > 0,
			Metrics:   analysis.Metrics,
		})
	})
	return points
}

func forEachState(
	snapshots []models.VehicleSnapshot,
	from time.Time,
	to time.Time,
	step time.Duration,
	visit func(timestamp time.Time, state []models.VehicleSnapshot),
) {
	if from.After(to) || step <= 0 {
		return
	}

	sorted := append([]models.VehicleSnapshot(nil), snapshots...)
	sort.Slice(sorted, func(left, right int) bool {
		if sorted[left].Timestamp.Equal(sorted[right].Timestamp) {
			return sorted[left].ID < sorted[right].ID
		}
		return sorted[left].Timestamp.Before(sorted[right].Timestamp)
	})

	latestByVehicleID := make(map[string]models.VehicleSnapshot)
	snapshotIndex := 0
	for timestamp := from; !timestamp.After(to); timestamp = timestamp.Add(step) {
		for snapshotIndex < len(sorted) && !sorted[snapshotIndex].Timestamp.After(timestamp) {
			latestByVehicleID[sorted[snapshotIndex].VehicleID] = sorted[snapshotIndex]
			snapshotIndex++
		}

		visit(timestamp, stateSnapshots(latestByVehicleID))
	}
}

func stateSnapshots(latestByVehicleID map[string]models.VehicleSnapshot) []models.VehicleSnapshot {
	state := make([]models.VehicleSnapshot, 0, len(latestByVehicleID))
	for _, snapshot := range latestByVehicleID {
		state = append(state, snapshot)
	}

	sort.Slice(state, func(left, right int) bool {
		leftProgress := routeProgressValue(state[left])
		rightProgress := routeProgressValue(state[right])
		if leftProgress == rightProgress {
			return state[left].VehicleID < state[right].VehicleID
		}
		return leftProgress < rightProgress
	})

	return state
}

func routeProgressValue(snapshot models.VehicleSnapshot) float64 {
	if snapshot.RouteProgress == nil {
		return 0
	}
	return *snapshot.RouteProgress
}

func deriveEvents(points []models.RouteHistoryPoint) []models.RouteHealthEvent {
	events := make([]models.RouteHealthEvent, 0)
	wasBunching := false
	wasStale := false

	for _, point := range points {
		hasBunching := point.HasData && point.Metrics.BunchingEventCount > 0
		if hasBunching && !wasBunching {
			events = append(events, models.RouteHealthEvent{
				Timestamp:   point.Timestamp,
				Type:        models.RouteHealthEventBunching,
				Description: fmt.Sprintf("%d vehicle pair(s) entered bunching risk.", point.Metrics.BunchingEventCount),
				Count:       point.Metrics.BunchingEventCount,
			})
		}

		hasStale := point.HasData && point.Metrics.StaleVehicleCount > 0
		if hasStale && !wasStale {
			events = append(events, models.RouteHealthEvent{
				Timestamp:   point.Timestamp,
				Type:        models.RouteHealthEventStaleTelemetry,
				Description: fmt.Sprintf("%d vehicle(s) entered stale telemetry.", point.Metrics.StaleVehicleCount),
				Count:       point.Metrics.StaleVehicleCount,
			})
		}

		wasBunching = hasBunching
		wasStale = hasStale
	}

	return events
}
