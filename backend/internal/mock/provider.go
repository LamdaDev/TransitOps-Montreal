package mock

import (
	"context"
	"math"
	"sync"
	"time"

	"github.com/lamda/transitops-montreal/backend/internal/models"
)

const sourceName = "mock-gtfs-rt"

type Point struct {
	Lat float64
	Lon float64
}

type vehicleSeed struct {
	VehicleID  string
	TripID     string
	Progress   float64
	Step       float64
	BaseSpeed  float64
	Phase      float64
	StaleEvery int
}

type routeFixture struct {
	Route    models.Route
	Path     []Point
	Vehicles []vehicleSeed
}

type Provider struct {
	mu       sync.Mutex
	sequence int
}

func NewProvider() *Provider {
	return &Provider{}
}

func (p *Provider) Name() string {
	return sourceName
}

func (p *Provider) FetchSnapshots(_ context.Context) ([]models.VehicleSnapshot, error) {
	p.mu.Lock()
	p.sequence++
	sequence := p.sequence
	p.mu.Unlock()

	now := time.Now().UTC()
	snapshots := make([]models.VehicleSnapshot, 0, 24)

	for _, fixture := range fixtures() {
		for _, vehicle := range fixture.Vehicles {
			progress := normalizeProgress(vehicle.Progress + float64(sequence)*vehicle.Step)
			position := interpolate(fixture.Path, progress)
			speed := math.Max(0, vehicle.BaseSpeed+math.Sin(float64(sequence)*0.7+vehicle.Phase)*2.5)
			timestamp := now

			if vehicle.StaleEvery > 0 && sequence%vehicle.StaleEvery == 0 {
				timestamp = now.Add(-4 * time.Minute)
				speed = 0
			}

			tripID := vehicle.TripID
			progressValue := progress
			speedValue := round(speed, 1)

			snapshots = append(snapshots, models.VehicleSnapshot{
				VehicleID:     vehicle.VehicleID,
				RouteID:       fixture.Route.ID,
				TripID:        &tripID,
				Latitude:      round(position.Lat, 6),
				Longitude:     round(position.Lon, 6),
				Speed:         &speedValue,
				RouteProgress: &progressValue,
				Timestamp:     timestamp,
				Source:        sourceName,
			})
		}
	}

	return snapshots, nil
}

func Routes() []models.Route {
	fixtures := fixtures()
	routes := make([]models.Route, 0, len(fixtures))
	for _, fixture := range fixtures {
		route := fixture.Route
		route.Shape = make([]models.RouteShapePoint, 0, len(fixture.Path))
		for _, point := range fixture.Path {
			route.Shape = append(route.Shape, models.RouteShapePoint{
				Latitude:  point.Lat,
				Longitude: point.Lon,
			})
		}

		routes = append(routes, route)
	}
	return routes
}

func fixtures() []routeFixture {
	blue := "#0072ce"
	green := "#00a651"
	orange := "#f58220"
	purple := "#6b4fa3"
	red := "#d71920"

	return []routeFixture{
		{
			Route: models.Route{ID: "24", ShortName: "24", LongName: "Sherbrooke", Color: &blue},
			Path: []Point{
				{Lat: 45.5112, Lon: -73.6282},
				{Lat: 45.5138, Lon: -73.6047},
				{Lat: 45.5162, Lon: -73.5798},
				{Lat: 45.5215, Lon: -73.5527},
			},
			Vehicles: []vehicleSeed{
				{VehicleID: "STM-2401", TripID: "24-E-001", Progress: 0.18, Step: 0.012, BaseSpeed: 26, Phase: 0.1},
				{VehicleID: "STM-2402", TripID: "24-E-002", Progress: 0.205, Step: 0.012, BaseSpeed: 24, Phase: 1.2},
				{VehicleID: "STM-2403", TripID: "24-W-003", Progress: 0.58, Step: 0.010, BaseSpeed: 31, Phase: 2.4},
				{VehicleID: "STM-2404", TripID: "24-W-004", Progress: 0.82, Step: 0.008, BaseSpeed: 27, Phase: 3.3},
			},
		},
		{
			Route: models.Route{ID: "55", ShortName: "55", LongName: "Saint-Laurent", Color: &green},
			Path: []Point{
				{Lat: 45.4948, Lon: -73.5561},
				{Lat: 45.5098, Lon: -73.5635},
				{Lat: 45.5268, Lon: -73.5878},
				{Lat: 45.5405, Lon: -73.6032},
			},
			Vehicles: []vehicleSeed{
				{VehicleID: "STM-5501", TripID: "55-N-001", Progress: 0.08, Step: 0.010, BaseSpeed: 19, Phase: 0.7},
				{VehicleID: "STM-5502", TripID: "55-N-002", Progress: 0.34, Step: 0.011, BaseSpeed: 23, Phase: 1.9},
				{VehicleID: "STM-5503", TripID: "55-S-003", Progress: 0.65, Step: 0.009, BaseSpeed: 4, Phase: 2.8},
				{VehicleID: "STM-5504", TripID: "55-S-004", Progress: 0.88, Step: 0.010, BaseSpeed: 21, Phase: 3.5},
			},
		},
		{
			Route: models.Route{ID: "80", ShortName: "80", LongName: "Avenue du Parc", Color: &orange},
			Path: []Point{
				{Lat: 45.5013, Lon: -73.5712},
				{Lat: 45.5134, Lon: -73.5824},
				{Lat: 45.5263, Lon: -73.5962},
				{Lat: 45.5359, Lon: -73.6071},
			},
			Vehicles: []vehicleSeed{
				{VehicleID: "STM-8001", TripID: "80-N-001", Progress: 0.12, Step: 0.011, BaseSpeed: 22, Phase: 0.2},
				{VehicleID: "STM-8002", TripID: "80-N-002", Progress: 0.39, Step: 0.010, BaseSpeed: 26, Phase: 1.6, StaleEvery: 3},
				{VehicleID: "STM-8003", TripID: "80-S-003", Progress: 0.69, Step: 0.009, BaseSpeed: 20, Phase: 2.6},
			},
		},
		{
			Route: models.Route{ID: "165", ShortName: "165", LongName: "Cote-des-Neiges", Color: &purple},
			Path: []Point{
				{Lat: 45.4949, Lon: -73.6202},
				{Lat: 45.5012, Lon: -73.6265},
				{Lat: 45.5094, Lon: -73.6326},
				{Lat: 45.5178, Lon: -73.6381},
			},
			Vehicles: []vehicleSeed{
				{VehicleID: "STM-1651", TripID: "165-N-001", Progress: 0.05, Step: 0.010, BaseSpeed: 18, Phase: 0.4},
				{VehicleID: "STM-1652", TripID: "165-N-002", Progress: 0.29, Step: 0.011, BaseSpeed: 24, Phase: 1.5},
				{VehicleID: "STM-1653", TripID: "165-S-003", Progress: 0.53, Step: 0.010, BaseSpeed: 20, Phase: 2.7},
				{VehicleID: "STM-1654", TripID: "165-S-004", Progress: 0.78, Step: 0.009, BaseSpeed: 22, Phase: 3.8},
			},
		},
		{
			Route: models.Route{ID: "470", ShortName: "470", LongName: "Express Pierrefonds", Color: &red},
			Path: []Point{
				{Lat: 45.4751, Lon: -73.8596},
				{Lat: 45.4882, Lon: -73.8071},
				{Lat: 45.4974, Lon: -73.7503},
				{Lat: 45.5145, Lon: -73.6834},
			},
			Vehicles: []vehicleSeed{
				{VehicleID: "STM-4701", TripID: "470-E-001", Progress: 0.04, Step: 0.018, BaseSpeed: 42, Phase: 0.9},
				{VehicleID: "STM-4702", TripID: "470-E-002", Progress: 0.13, Step: 0.018, BaseSpeed: 46, Phase: 1.8},
				{VehicleID: "STM-4703", TripID: "470-W-003", Progress: 0.74, Step: 0.015, BaseSpeed: 48, Phase: 2.5},
			},
		},
	}
}

func normalizeProgress(value float64) float64 {
	progress := math.Mod(value, 1)
	if progress < 0 {
		return progress + 1
	}
	return progress
}

func interpolate(path []Point, progress float64) Point {
	if len(path) == 0 {
		return Point{Lat: 45.5019, Lon: -73.5674}
	}
	if len(path) == 1 {
		return path[0]
	}

	scaled := progress * float64(len(path)-1)
	segment := int(math.Floor(scaled))
	if segment >= len(path)-1 {
		return path[len(path)-1]
	}

	local := scaled - float64(segment)
	start := path[segment]
	end := path[segment+1]

	return Point{
		Lat: start.Lat + (end.Lat-start.Lat)*local,
		Lon: start.Lon + (end.Lon-start.Lon)*local,
	}
}

func round(value float64, precision int) float64 {
	scale := math.Pow(10, float64(precision))
	return math.Round(value*scale) / scale
}
