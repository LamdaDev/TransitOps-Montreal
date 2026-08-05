package models

import "time"

type VehicleStatus string

const (
	VehicleStatusActive       VehicleStatus = "ACTIVE"
	VehicleStatusStale        VehicleStatus = "STALE"
	VehicleStatusBunchingRisk VehicleStatus = "BUNCHING_RISK"
	VehicleStatusDelayed      VehicleStatus = "DELAYED"
	VehicleStatusUnknown      VehicleStatus = "UNKNOWN"
)

const (
	HealthHealthy  = "HEALTHY"
	HealthWatch    = "WATCH"
	HealthDegraded = "DEGRADED"
)

type Route struct {
	ID        string            `json:"id"`
	ShortName string            `json:"shortName"`
	LongName  string            `json:"longName"`
	Color     *string           `json:"color,omitempty"`
	Shape     []RouteShapePoint `json:"shape"`
}

type RouteShapePoint struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

type VehicleSnapshot struct {
	ID            int64     `json:"id"`
	VehicleID     string    `json:"vehicleId"`
	RouteID       string    `json:"routeId"`
	TripID        *string   `json:"tripId,omitempty"`
	Latitude      float64   `json:"latitude"`
	Longitude     float64   `json:"longitude"`
	Speed         *float64  `json:"speed,omitempty"`
	RouteProgress *float64  `json:"routeProgress,omitempty"`
	Timestamp     time.Time `json:"timestamp"`
	Source        string    `json:"source"`
	CreatedAt     time.Time `json:"createdAt"`
}

type Vehicle struct {
	VehicleSnapshot
	Status VehicleStatus `json:"status"`
}

type VehicleTrailPoint struct {
	Latitude  float64   `json:"latitude"`
	Longitude float64   `json:"longitude"`
	Timestamp time.Time `json:"timestamp"`
}

type VehicleTrail struct {
	VehicleID string              `json:"vehicleId"`
	Points    []VehicleTrailPoint `json:"points"`
}

type RouteMetrics struct {
	RouteID                  string    `json:"routeId"`
	ActiveVehicleCount       int       `json:"activeVehicleCount"`
	StaleVehicleCount        int       `json:"staleVehicleCount"`
	BunchingEventCount       int       `json:"bunchingEventCount"`
	LargestHeadwayGapMinutes float64   `json:"largestHeadwayGapMinutes"`
	AverageSpacingMinutes    float64   `json:"averageSpacingMinutes"`
	HealthStatus             string    `json:"healthStatus"`
	LastUpdated              time.Time `json:"lastUpdated"`
}

type RouteAnalysis struct {
	Vehicles []Vehicle    `json:"vehicles"`
	Metrics  RouteMetrics `json:"metrics"`
	Insights []string     `json:"insights"`
}

type RouteHistoryPoint struct {
	Timestamp time.Time    `json:"timestamp"`
	HasData   bool         `json:"hasData"`
	Metrics   RouteMetrics `json:"metrics"`
}

type RouteHealthEventType string

const (
	RouteHealthEventBunching       RouteHealthEventType = "BUNCHING_RISK"
	RouteHealthEventStaleTelemetry RouteHealthEventType = "STALE_TELEMETRY"
)

type RouteHealthEvent struct {
	Timestamp   time.Time            `json:"timestamp"`
	Type        RouteHealthEventType `json:"type"`
	Description string               `json:"description"`
	Count       int                  `json:"count"`
}

type RouteHistory struct {
	RouteID       string              `json:"routeId"`
	From          time.Time           `json:"from"`
	To            time.Time           `json:"to"`
	BucketSeconds int                 `json:"bucketSeconds"`
	Points        []RouteHistoryPoint `json:"points"`
	Events        []RouteHealthEvent  `json:"events"`
}

type ReplayFrame struct {
	Timestamp time.Time    `json:"timestamp"`
	HasData   bool         `json:"hasData"`
	Vehicles  []Vehicle    `json:"vehicles"`
	Metrics   RouteMetrics `json:"metrics"`
	Insights  []string     `json:"insights"`
}

type RouteReplay struct {
	RouteID     string        `json:"routeId"`
	From        time.Time     `json:"from"`
	To          time.Time     `json:"to"`
	StepSeconds int           `json:"stepSeconds"`
	Frames      []ReplayFrame `json:"frames"`
}

type IngestResult struct {
	InsertedCount int       `json:"insertedCount"`
	Timestamp     time.Time `json:"timestamp"`
	Source        string    `json:"source"`
}
