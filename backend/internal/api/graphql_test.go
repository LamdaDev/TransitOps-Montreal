package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/lamda/transitops-montreal/backend/internal/models"
)

type graphQLTestStore struct {
	snapshots []models.VehicleSnapshot
}

func (s graphQLTestStore) Routes(context.Context) ([]models.Route, error) {
	return []models.Route{}, nil
}

func (s graphQLTestStore) LatestVehicleSnapshots(context.Context, string) ([]models.VehicleSnapshot, error) {
	return s.snapshots, nil
}

func (s graphQLTestStore) VehicleSnapshotsInRange(
	context.Context,
	string,
	time.Time,
	time.Time,
) ([]models.VehicleSnapshot, error) {
	return s.snapshots, nil
}

func (s graphQLTestStore) VehicleTrails(
	context.Context,
	string,
	time.Time,
	int,
) ([]models.VehicleTrail, error) {
	return []models.VehicleTrail{}, nil
}

func TestHistoryAndReplayQueries(t *testing.T) {
	observedAt := time.Now().UTC().Add(-time.Minute)
	progressOne := 0.2
	progressTwo := 0.205
	speed := 25.0
	store := graphQLTestStore{snapshots: []models.VehicleSnapshot{
		{
			ID: 1, VehicleID: "STM-1", RouteID: "24", Latitude: 45.5, Longitude: -73.5,
			RouteProgress: &progressOne, Speed: &speed, Timestamp: observedAt, CreatedAt: observedAt, Source: "test",
		},
		{
			ID: 2, VehicleID: "STM-2", RouteID: "24", Latitude: 45.5001, Longitude: -73.5001,
			RouteProgress: &progressTwo, Speed: &speed, Timestamp: observedAt, CreatedAt: observedAt, Source: "test",
		},
	}}

	graphQLHandler, err := NewGraphQLHandler(store, nil)
	if err != nil {
		t.Fatalf("build graphql schema: %v", err)
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/graphql", strings.NewReader(`{
		"query": "query { routeHistory(routeId: \"24\", rangeMinutes: 30) { routeId bucketSeconds points { timestamp hasData healthStatus } events { type } } routeReplay(routeId: \"24\", rangeMinutes: 30) { routeId stepSeconds frames { timestamp hasData vehicles { vehicleId status } metrics { healthStatus } insights } } }"
	}`))
	request.Header.Set("Content-Type", "application/json")
	graphQLHandler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, recorder.Code, recorder.Body.String())
	}

	var response struct {
		Data   json.RawMessage `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode graphql response: %v", err)
	}
	if len(response.Errors) > 0 {
		t.Fatalf("expected graphql data, got errors: %#v", response.Errors)
	}
	if len(response.Data) == 0 {
		t.Fatal("expected graphql data")
	}
}
