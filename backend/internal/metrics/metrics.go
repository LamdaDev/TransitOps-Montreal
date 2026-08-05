package metrics

import (
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/lamda/transitops-montreal/backend/internal/models"
)

const (
	staleAfter            = 2 * time.Minute
	bunchingDistanceM    = 400.0
	routeCycleMinutes    = 60.0
	longGapWarningMinutes = 18.0
	delayedSpeedKPH       = 6.0
)

func Analyze(routeID string, snapshots []models.VehicleSnapshot, now time.Time) models.RouteAnalysis {
	vehicles := make([]models.Vehicle, 0, len(snapshots))
	bunchingVehicles, bunchingEvents := detectBunching(snapshots, now)
	lastUpdated := time.Time{}
	staleCount := 0
	activeVehicleCount := 0

	for _, snapshot := range snapshots {
		if snapshot.Timestamp.After(lastUpdated) {
			lastUpdated = snapshot.Timestamp
		}

		status := statusFor(snapshot, now, bunchingVehicles[snapshot.VehicleID])
		if status == models.VehicleStatusStale {
			staleCount++
		} else {
			activeVehicleCount++
		}

		vehicles = append(vehicles, models.Vehicle{
			VehicleSnapshot: snapshot,
			Status:          status,
		})
	}

	largestGap := largestHeadwayGapMinutes(snapshots)
	averageSpacing := 0.0
	if activeVehicleCount > 0 {
		averageSpacing = routeCycleMinutes / float64(activeVehicleCount)
	}

	issueCount := 0
	if staleCount > 0 {
		issueCount++
	}
	if bunchingEvents > 0 {
		issueCount++
	}
	if largestGap >= longGapWarningMinutes {
		issueCount++
	}

	healthStatus := models.HealthHealthy
	if issueCount == 1 {
		healthStatus = models.HealthWatch
	}
	if issueCount >= 2 {
		healthStatus = models.HealthDegraded
	}

	routeMetrics := models.RouteMetrics{
		RouteID:                   routeID,
		ActiveVehicleCount:       activeVehicleCount,
		StaleVehicleCount:        staleCount,
		BunchingEventCount:       bunchingEvents,
		LargestHeadwayGapMinutes: round(largestGap, 1),
		AverageSpacingMinutes:    round(averageSpacing, 1),
		HealthStatus:             healthStatus,
		LastUpdated:              lastUpdated,
	}

	return models.RouteAnalysis{
		Vehicles: vehicles,
		Metrics:  routeMetrics,
		Insights: buildInsights(routeMetrics),
	}
}

func statusFor(snapshot models.VehicleSnapshot, now time.Time, hasBunchingRisk bool) models.VehicleStatus {
	if snapshot.Timestamp.IsZero() {
		return models.VehicleStatusUnknown
	}
	if now.Sub(snapshot.Timestamp) > staleAfter {
		return models.VehicleStatusStale
	}
	if hasBunchingRisk {
		return models.VehicleStatusBunchingRisk
	}
	if snapshot.Speed != nil && *snapshot.Speed < delayedSpeedKPH {
		return models.VehicleStatusDelayed
	}
	return models.VehicleStatusActive
}

func detectBunching(snapshots []models.VehicleSnapshot, now time.Time) (map[string]bool, int) {
	flagged := map[string]bool{}
	events := 0

	for i := 0; i < len(snapshots); i++ {
		for j := i + 1; j < len(snapshots); j++ {
			left := snapshots[i]
			right := snapshots[j]
			if left.RouteID != right.RouteID ||
				now.Sub(left.Timestamp) > staleAfter ||
				now.Sub(right.Timestamp) > staleAfter {
				continue
			}

			if haversineMeters(left.Latitude, left.Longitude, right.Latitude, right.Longitude) <= bunchingDistanceM {
				flagged[left.VehicleID] = true
				flagged[right.VehicleID] = true
				events++
			}
		}
	}

	return flagged, events
}

func largestHeadwayGapMinutes(snapshots []models.VehicleSnapshot) float64 {
	progressValues := make([]float64, 0, len(snapshots))
	for _, snapshot := range snapshots {
		if snapshot.RouteProgress != nil {
			progressValues = append(progressValues, *snapshot.RouteProgress)
		}
	}

	if len(progressValues) < 2 {
		return 0
	}

	sort.Float64s(progressValues)
	largestGap := 0.0
	for i := 1; i < len(progressValues); i++ {
		largestGap = math.Max(largestGap, progressValues[i]-progressValues[i-1])
	}

	wrapGap := (1 - progressValues[len(progressValues)-1]) + progressValues[0]
	largestGap = math.Max(largestGap, wrapGap)

	return largestGap * routeCycleMinutes
}

func buildInsights(routeMetrics models.RouteMetrics) []string {
	insights := []string{}

	if routeMetrics.BunchingEventCount > 0 {
		insights = append(insights, fmt.Sprintf("%d vehicle pair(s) appear close together, possible bunching risk.", routeMetrics.BunchingEventCount))
	}

	if routeMetrics.LargestHeadwayGapMinutes >= longGapWarningMinutes {
		insights = append(insights, fmt.Sprintf("Largest estimated route gap is %.0f minutes.", routeMetrics.LargestHeadwayGapMinutes))
	}

	if routeMetrics.StaleVehicleCount > 0 {
		insights = append(insights, fmt.Sprintf("%d vehicle(s) have not updated recently.", routeMetrics.StaleVehicleCount))
	}

	if len(insights) == 0 {
		insights = append(insights, "Route currently looks healthy.")
	}

	return insights
}

func haversineMeters(lat1, lon1, lat2, lon2 float64) float64 {
	const earthRadiusMeters = 6371000.0

	lat1Rad := degreesToRadians(lat1)
	lat2Rad := degreesToRadians(lat2)
	deltaLat := degreesToRadians(lat2 - lat1)
	deltaLon := degreesToRadians(lon2 - lon1)

	a := math.Sin(deltaLat/2)*math.Sin(deltaLat/2) +
		math.Cos(lat1Rad)*math.Cos(lat2Rad)*math.Sin(deltaLon/2)*math.Sin(deltaLon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return earthRadiusMeters * c
}

func degreesToRadians(value float64) float64 {
	return value * math.Pi / 180
}

func round(value float64, precision int) float64 {
	scale := math.Pow(10, float64(precision))
	return math.Round(value*scale) / scale
}
