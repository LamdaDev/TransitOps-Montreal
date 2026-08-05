package history

import (
	"testing"
	"time"

	"github.com/lamda/transitops-montreal/backend/internal/models"
)

func TestBuildRouteHistoryEmitsBunchingStartEvent(t *testing.T) {
	from := time.Date(2026, time.August, 5, 9, 0, 0, 0, time.UTC)
	to := from.Add(2 * time.Minute)
	firstProgress := 0.2
	secondProgress := 0.205
	separatedProgress := 0.7
	speed := 25.0

	snapshots := []models.VehicleSnapshot{
		{
			ID: 1, VehicleID: "STM-1", RouteID: "24", Latitude: 45.5, Longitude: -73.5,
			RouteProgress: &firstProgress, Speed: &speed, Timestamp: from,
		},
		{
			ID: 2, VehicleID: "STM-2", RouteID: "24", Latitude: 45.5001, Longitude: -73.5001,
			RouteProgress: &secondProgress, Speed: &speed, Timestamp: from,
		},
		{
			ID: 3, VehicleID: "STM-2", RouteID: "24", Latitude: 45.53, Longitude: -73.55,
			RouteProgress: &separatedProgress, Speed: &speed, Timestamp: from.Add(time.Minute),
		},
	}

	history := BuildRouteHistory("24", snapshots, from, to, time.Minute)

	if len(history.Points) != 3 {
		t.Fatalf("expected 3 history points, got %d", len(history.Points))
	}
	if !history.Points[0].HasData {
		t.Fatal("expected first point to contain reconstructed vehicle state")
	}
	if history.Points[0].Metrics.BunchingEventCount != 1 {
		t.Fatalf("expected initial bunching event, got %d", history.Points[0].Metrics.BunchingEventCount)
	}
	if history.Points[1].Metrics.BunchingEventCount != 0 {
		t.Fatalf("expected bunching to clear after the updated vehicle position, got %d", history.Points[1].Metrics.BunchingEventCount)
	}
	if len(history.Events) != 1 || history.Events[0].Type != models.RouteHealthEventBunching {
		t.Fatalf("expected one bunching-start event, got %#v", history.Events)
	}
}
