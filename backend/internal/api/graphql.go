package api

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/graphql-go/graphql"
	"github.com/graphql-go/handler"

	"github.com/lamda/transitops-montreal/backend/internal/history"
	"github.com/lamda/transitops-montreal/backend/internal/ingest"
	"github.com/lamda/transitops-montreal/backend/internal/metrics"
	"github.com/lamda/transitops-montreal/backend/internal/models"
)

const (
	defaultTrailWindowMinutes  = 10
	maxTrailWindowMinutes      = 60
	maxTrailPointsPerVehicle   = 60
	defaultHistoryRangeMinutes = 30
	maxHistoryRangeMinutes     = 60
	historyBucket              = time.Minute
	replayFrameStep            = 15 * time.Second
)

type RouteStore interface {
	Routes(ctx context.Context) ([]models.Route, error)
	LatestVehicleSnapshots(ctx context.Context, routeID string) ([]models.VehicleSnapshot, error)
	VehicleSnapshotsInRange(ctx context.Context, routeID string, from time.Time, to time.Time) ([]models.VehicleSnapshot, error)
	VehicleTrails(ctx context.Context, routeID string, since time.Time, maxPoints int) ([]models.VehicleTrail, error)
}

type GraphQLServer struct {
	store  RouteStore
	ingest *ingest.Service
}

func NewGraphQLHandler(store RouteStore, ingestService *ingest.Service) (http.Handler, error) {
	server := &GraphQLServer{store: store, ingest: ingestService}
	schema, err := server.schema()
	if err != nil {
		return nil, err
	}

	graphQLHandler := handler.New(&handler.Config{
		Schema:   &schema,
		Pretty:   true,
		GraphiQL: true,
	})

	return withCORS(graphQLHandler), nil
}

func (s *GraphQLServer) schema() (graphql.Schema, error) {
	vehicleStatusType := graphql.NewEnum(graphql.EnumConfig{
		Name: "VehicleStatus",
		Values: graphql.EnumValueConfigMap{
			"ACTIVE":        &graphql.EnumValueConfig{Value: string(models.VehicleStatusActive)},
			"STALE":         &graphql.EnumValueConfig{Value: string(models.VehicleStatusStale)},
			"BUNCHING_RISK": &graphql.EnumValueConfig{Value: string(models.VehicleStatusBunchingRisk)},
			"DELAYED":       &graphql.EnumValueConfig{Value: string(models.VehicleStatusDelayed)},
			"UNKNOWN":       &graphql.EnumValueConfig{Value: string(models.VehicleStatusUnknown)},
		},
	})

	healthStatusType := graphql.NewEnum(graphql.EnumConfig{
		Name: "HealthStatus",
		Values: graphql.EnumValueConfigMap{
			"HEALTHY":  &graphql.EnumValueConfig{Value: models.HealthHealthy},
			"WATCH":    &graphql.EnumValueConfig{Value: models.HealthWatch},
			"DEGRADED": &graphql.EnumValueConfig{Value: models.HealthDegraded},
		},
	})

	routeShapePointType := graphql.NewObject(graphql.ObjectConfig{
		Name: "RouteShapePoint",
		Fields: graphql.Fields{
			"latitude":  &graphql.Field{Type: graphql.NewNonNull(graphql.Float)},
			"longitude": &graphql.Field{Type: graphql.NewNonNull(graphql.Float)},
		},
	})

	vehicleTrailPointType := graphql.NewObject(graphql.ObjectConfig{
		Name: "VehicleTrailPoint",
		Fields: graphql.Fields{
			"latitude":  &graphql.Field{Type: graphql.NewNonNull(graphql.Float)},
			"longitude": &graphql.Field{Type: graphql.NewNonNull(graphql.Float)},
			"timestamp": &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
		},
	})

	routeType := graphql.NewObject(graphql.ObjectConfig{
		Name: "Route",
		Fields: graphql.Fields{
			"id":        &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
			"shortName": &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
			"longName":  &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
			"color":     &graphql.Field{Type: graphql.String},
			"shape": &graphql.Field{Type: graphql.NewNonNull(
				graphql.NewList(graphql.NewNonNull(routeShapePointType)),
			)},
		},
	})

	vehicleType := graphql.NewObject(graphql.ObjectConfig{
		Name: "Vehicle",
		Fields: graphql.Fields{
			"id":            &graphql.Field{Type: graphql.NewNonNull(graphql.Int)},
			"vehicleId":     &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
			"routeId":       &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
			"tripId":        &graphql.Field{Type: graphql.String},
			"latitude":      &graphql.Field{Type: graphql.NewNonNull(graphql.Float)},
			"longitude":     &graphql.Field{Type: graphql.NewNonNull(graphql.Float)},
			"speed":         &graphql.Field{Type: graphql.Float},
			"routeProgress": &graphql.Field{Type: graphql.Float},
			"timestamp":     &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
			"source":        &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
			"createdAt":     &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
			"status":        &graphql.Field{Type: graphql.NewNonNull(vehicleStatusType)},
		},
	})

	vehicleTrailType := graphql.NewObject(graphql.ObjectConfig{
		Name: "VehicleTrail",
		Fields: graphql.Fields{
			"vehicleId": &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
			"points": &graphql.Field{Type: graphql.NewNonNull(
				graphql.NewList(graphql.NewNonNull(vehicleTrailPointType)),
			)},
		},
	})

	metricsType := graphql.NewObject(graphql.ObjectConfig{
		Name: "RouteMetrics",
		Fields: graphql.Fields{
			"routeId":                  &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
			"activeVehicleCount":       &graphql.Field{Type: graphql.NewNonNull(graphql.Int)},
			"staleVehicleCount":        &graphql.Field{Type: graphql.NewNonNull(graphql.Int)},
			"bunchingEventCount":       &graphql.Field{Type: graphql.NewNonNull(graphql.Int)},
			"largestHeadwayGapMinutes": &graphql.Field{Type: graphql.NewNonNull(graphql.Float)},
			"averageSpacingMinutes":    &graphql.Field{Type: graphql.NewNonNull(graphql.Float)},
			"healthStatus":             &graphql.Field{Type: graphql.NewNonNull(healthStatusType)},
			"lastUpdated":              &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
		},
	})

	routeHistoryPointType := graphql.NewObject(graphql.ObjectConfig{
		Name: "RouteHistoryPoint",
		Fields: graphql.Fields{
			"timestamp":                &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
			"hasData":                  &graphql.Field{Type: graphql.NewNonNull(graphql.Boolean)},
			"activeVehicleCount":       &graphql.Field{Type: graphql.NewNonNull(graphql.Int)},
			"staleVehicleCount":        &graphql.Field{Type: graphql.NewNonNull(graphql.Int)},
			"bunchingEventCount":       &graphql.Field{Type: graphql.NewNonNull(graphql.Int)},
			"largestHeadwayGapMinutes": &graphql.Field{Type: graphql.NewNonNull(graphql.Float)},
			"averageSpacingMinutes":    &graphql.Field{Type: graphql.NewNonNull(graphql.Float)},
			"healthStatus":             &graphql.Field{Type: graphql.NewNonNull(healthStatusType)},
		},
	})

	routeHealthEventKindType := graphql.NewEnum(graphql.EnumConfig{
		Name: "RouteHealthEventKind",
		Values: graphql.EnumValueConfigMap{
			"BUNCHING_RISK":   &graphql.EnumValueConfig{Value: string(models.RouteHealthEventBunching)},
			"STALE_TELEMETRY": &graphql.EnumValueConfig{Value: string(models.RouteHealthEventStaleTelemetry)},
		},
	})

	routeHealthEventType := graphql.NewObject(graphql.ObjectConfig{
		Name: "RouteHealthEvent",
		Fields: graphql.Fields{
			"timestamp":   &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
			"type":        &graphql.Field{Type: graphql.NewNonNull(routeHealthEventKindType)},
			"description": &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
			"count":       &graphql.Field{Type: graphql.NewNonNull(graphql.Int)},
		},
	})

	routeHistoryType := graphql.NewObject(graphql.ObjectConfig{
		Name: "RouteHistory",
		Fields: graphql.Fields{
			"routeId":       &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
			"from":          &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
			"to":            &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
			"bucketSeconds": &graphql.Field{Type: graphql.NewNonNull(graphql.Int)},
			"points": &graphql.Field{Type: graphql.NewNonNull(
				graphql.NewList(graphql.NewNonNull(routeHistoryPointType)),
			)},
			"events": &graphql.Field{Type: graphql.NewNonNull(
				graphql.NewList(graphql.NewNonNull(routeHealthEventType)),
			)},
		},
	})

	replayFrameType := graphql.NewObject(graphql.ObjectConfig{
		Name: "ReplayFrame",
		Fields: graphql.Fields{
			"timestamp": &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
			"hasData":   &graphql.Field{Type: graphql.NewNonNull(graphql.Boolean)},
			"vehicles": &graphql.Field{Type: graphql.NewNonNull(
				graphql.NewList(graphql.NewNonNull(vehicleType)),
			)},
			"metrics":  &graphql.Field{Type: graphql.NewNonNull(metricsType)},
			"insights": &graphql.Field{Type: graphql.NewNonNull(graphql.NewList(graphql.NewNonNull(graphql.String)))},
		},
	})

	routeReplayType := graphql.NewObject(graphql.ObjectConfig{
		Name: "RouteReplay",
		Fields: graphql.Fields{
			"routeId":     &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
			"from":        &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
			"to":          &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
			"stepSeconds": &graphql.Field{Type: graphql.NewNonNull(graphql.Int)},
			"frames": &graphql.Field{Type: graphql.NewNonNull(
				graphql.NewList(graphql.NewNonNull(replayFrameType)),
			)},
		},
	})

	ingestResultType := graphql.NewObject(graphql.ObjectConfig{
		Name: "IngestResult",
		Fields: graphql.Fields{
			"insertedCount": &graphql.Field{Type: graphql.NewNonNull(graphql.Int)},
			"timestamp":     &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
			"source":        &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
		},
	})

	queryType := graphql.NewObject(graphql.ObjectConfig{
		Name: "Query",
		Fields: graphql.Fields{
			"routes": &graphql.Field{
				Type: graphql.NewNonNull(graphql.NewList(graphql.NewNonNull(routeType))),
				Resolve: func(params graphql.ResolveParams) (any, error) {
					routes, err := s.store.Routes(params.Context)
					if err != nil {
						return nil, err
					}
					return formatRoutes(routes), nil
				},
			},
			"vehicles": &graphql.Field{
				Type: graphql.NewNonNull(graphql.NewList(graphql.NewNonNull(vehicleType))),
				Args: graphql.FieldConfigArgument{
					"routeId": &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.String)},
				},
				Resolve: func(params graphql.ResolveParams) (any, error) {
					routeID, _ := params.Args["routeId"].(string)
					analysis, err := s.analyzeRoute(params.Context, routeID)
					if err != nil {
						return nil, err
					}
					return formatVehicles(analysis.Vehicles), nil
				},
			},
			"routeMetrics": &graphql.Field{
				Type: graphql.NewNonNull(metricsType),
				Args: graphql.FieldConfigArgument{
					"routeId": &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.String)},
				},
				Resolve: func(params graphql.ResolveParams) (any, error) {
					routeID, _ := params.Args["routeId"].(string)
					analysis, err := s.analyzeRoute(params.Context, routeID)
					if err != nil {
						return nil, err
					}
					return formatRouteMetrics(analysis.Metrics), nil
				},
			},
			"routeInsights": &graphql.Field{
				Type: graphql.NewNonNull(graphql.NewList(graphql.NewNonNull(graphql.String))),
				Args: graphql.FieldConfigArgument{
					"routeId": &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.String)},
				},
				Resolve: func(params graphql.ResolveParams) (any, error) {
					routeID, _ := params.Args["routeId"].(string)
					analysis, err := s.analyzeRoute(params.Context, routeID)
					if err != nil {
						return nil, err
					}
					return analysis.Insights, nil
				},
			},
			"vehicleTrails": &graphql.Field{
				Type: graphql.NewNonNull(graphql.NewList(graphql.NewNonNull(vehicleTrailType))),
				Args: graphql.FieldConfigArgument{
					"routeId": &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.String)},
					"minutes": &graphql.ArgumentConfig{Type: graphql.Int},
				},
				Resolve: func(params graphql.ResolveParams) (any, error) {
					routeID, _ := params.Args["routeId"].(string)
					minutes := trailWindowMinutes(params.Args)
					trails, err := s.store.VehicleTrails(
						params.Context,
						routeID,
						time.Now().UTC().Add(-time.Duration(minutes)*time.Minute),
						maxTrailPointsPerVehicle,
					)
					if err != nil {
						return nil, fmt.Errorf("load vehicle trails for route %s: %w", routeID, err)
					}
					return formatVehicleTrails(trails), nil
				},
			},
			"routeHistory": &graphql.Field{
				Type: graphql.NewNonNull(routeHistoryType),
				Args: graphql.FieldConfigArgument{
					"routeId":      &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.String)},
					"rangeMinutes": &graphql.ArgumentConfig{Type: graphql.Int},
				},
				Resolve: func(params graphql.ResolveParams) (any, error) {
					routeID, _ := params.Args["routeId"].(string)
					rangeMinutes := historyRangeMinutes(params.Args)
					routeHistory, err := s.routeHistory(params.Context, routeID, rangeMinutes)
					if err != nil {
						return nil, err
					}
					return formatRouteHistory(routeHistory), nil
				},
			},
			"routeReplay": &graphql.Field{
				Type: graphql.NewNonNull(routeReplayType),
				Args: graphql.FieldConfigArgument{
					"routeId":      &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.String)},
					"rangeMinutes": &graphql.ArgumentConfig{Type: graphql.Int},
				},
				Resolve: func(params graphql.ResolveParams) (any, error) {
					routeID, _ := params.Args["routeId"].(string)
					rangeMinutes := historyRangeMinutes(params.Args)
					replay, err := s.routeReplay(params.Context, routeID, rangeMinutes)
					if err != nil {
						return nil, err
					}
					return formatRouteReplay(replay), nil
				},
			},
		},
	})

	mutationType := graphql.NewObject(graphql.ObjectConfig{
		Name: "Mutation",
		Fields: graphql.Fields{
			"ingestMock": &graphql.Field{
				Type: graphql.NewNonNull(ingestResultType),
				Resolve: func(params graphql.ResolveParams) (any, error) {
					result, err := s.ingest.RunOnce(params.Context)
					if err != nil {
						return nil, err
					}
					return formatIngestResult(result), nil
				},
			},
		},
	})

	return graphql.NewSchema(graphql.SchemaConfig{
		Query:    queryType,
		Mutation: mutationType,
	})
}

func (s *GraphQLServer) analyzeRoute(ctx context.Context, routeID string) (models.RouteAnalysis, error) {
	snapshots, err := s.store.LatestVehicleSnapshots(ctx, routeID)
	if err != nil {
		return models.RouteAnalysis{}, fmt.Errorf("load latest vehicles for route %s: %w", routeID, err)
	}
	return metrics.Analyze(routeID, snapshots, time.Now().UTC()), nil
}

func (s *GraphQLServer) routeHistory(
	ctx context.Context,
	routeID string,
	rangeMinutes int,
) (models.RouteHistory, error) {
	to := time.Now().UTC().Truncate(historyBucket)
	from := to.Add(-time.Duration(rangeMinutes) * time.Minute)
	snapshots, err := s.store.VehicleSnapshotsInRange(ctx, routeID, from, to)
	if err != nil {
		return models.RouteHistory{}, fmt.Errorf("load historical snapshots for route %s: %w", routeID, err)
	}

	return history.BuildRouteHistory(routeID, snapshots, from, to, historyBucket), nil
}

func (s *GraphQLServer) routeReplay(
	ctx context.Context,
	routeID string,
	rangeMinutes int,
) (models.RouteReplay, error) {
	to := time.Now().UTC().Truncate(replayFrameStep)
	from := to.Add(-time.Duration(rangeMinutes) * time.Minute)
	snapshots, err := s.store.VehicleSnapshotsInRange(ctx, routeID, from, to)
	if err != nil {
		return models.RouteReplay{}, fmt.Errorf("load replay snapshots for route %s: %w", routeID, err)
	}

	return history.BuildRouteReplay(routeID, snapshots, from, to, replayFrameStep), nil
}

func formatRoutes(routes []models.Route) []map[string]any {
	formatted := make([]map[string]any, 0, len(routes))
	for _, route := range routes {
		formatted = append(formatted, map[string]any{
			"id":        route.ID,
			"shortName": route.ShortName,
			"longName":  route.LongName,
			"color":     stringPtrValue(route.Color),
			"shape":     formatRouteShape(route.Shape),
		})
	}
	return formatted
}

func formatRouteShape(points []models.RouteShapePoint) []map[string]any {
	formatted := make([]map[string]any, 0, len(points))
	for _, point := range points {
		formatted = append(formatted, map[string]any{
			"latitude":  point.Latitude,
			"longitude": point.Longitude,
		})
	}
	return formatted
}

func formatVehicles(vehicles []models.Vehicle) []map[string]any {
	formatted := make([]map[string]any, 0, len(vehicles))
	for _, vehicle := range vehicles {
		formatted = append(formatted, map[string]any{
			"id":            int(vehicle.ID),
			"vehicleId":     vehicle.VehicleID,
			"routeId":       vehicle.RouteID,
			"tripId":        stringPtrValue(vehicle.TripID),
			"latitude":      vehicle.Latitude,
			"longitude":     vehicle.Longitude,
			"speed":         floatPtrValue(vehicle.Speed),
			"routeProgress": floatPtrValue(vehicle.RouteProgress),
			"timestamp":     formatTime(vehicle.Timestamp),
			"source":        vehicle.Source,
			"createdAt":     formatTime(vehicle.CreatedAt),
			"status":        string(vehicle.Status),
		})
	}
	return formatted
}

func formatVehicleTrails(trails []models.VehicleTrail) []map[string]any {
	formatted := make([]map[string]any, 0, len(trails))
	for _, trail := range trails {
		points := make([]map[string]any, 0, len(trail.Points))
		for _, point := range trail.Points {
			points = append(points, map[string]any{
				"latitude":  point.Latitude,
				"longitude": point.Longitude,
				"timestamp": formatTime(point.Timestamp),
			})
		}

		formatted = append(formatted, map[string]any{
			"vehicleId": trail.VehicleID,
			"points":    points,
		})
	}
	return formatted
}

func formatRouteMetrics(routeMetrics models.RouteMetrics) map[string]any {
	return map[string]any{
		"routeId":                  routeMetrics.RouteID,
		"activeVehicleCount":       routeMetrics.ActiveVehicleCount,
		"staleVehicleCount":        routeMetrics.StaleVehicleCount,
		"bunchingEventCount":       routeMetrics.BunchingEventCount,
		"largestHeadwayGapMinutes": routeMetrics.LargestHeadwayGapMinutes,
		"averageSpacingMinutes":    routeMetrics.AverageSpacingMinutes,
		"healthStatus":             routeMetrics.HealthStatus,
		"lastUpdated":              formatTime(routeMetrics.LastUpdated),
	}
}

func formatRouteHistory(routeHistory models.RouteHistory) map[string]any {
	points := make([]map[string]any, 0, len(routeHistory.Points))
	for _, point := range routeHistory.Points {
		points = append(points, map[string]any{
			"timestamp":                formatTime(point.Timestamp),
			"hasData":                  point.HasData,
			"activeVehicleCount":       point.Metrics.ActiveVehicleCount,
			"staleVehicleCount":        point.Metrics.StaleVehicleCount,
			"bunchingEventCount":       point.Metrics.BunchingEventCount,
			"largestHeadwayGapMinutes": point.Metrics.LargestHeadwayGapMinutes,
			"averageSpacingMinutes":    point.Metrics.AverageSpacingMinutes,
			"healthStatus":             point.Metrics.HealthStatus,
		})
	}

	events := make([]map[string]any, 0, len(routeHistory.Events))
	for _, event := range routeHistory.Events {
		events = append(events, map[string]any{
			"timestamp":   formatTime(event.Timestamp),
			"type":        string(event.Type),
			"description": event.Description,
			"count":       event.Count,
		})
	}

	return map[string]any{
		"routeId":       routeHistory.RouteID,
		"from":          formatTime(routeHistory.From),
		"to":            formatTime(routeHistory.To),
		"bucketSeconds": routeHistory.BucketSeconds,
		"points":        points,
		"events":        events,
	}
}

func formatRouteReplay(replay models.RouteReplay) map[string]any {
	frames := make([]map[string]any, 0, len(replay.Frames))
	for _, frame := range replay.Frames {
		frames = append(frames, map[string]any{
			"timestamp": formatTime(frame.Timestamp),
			"hasData":   frame.HasData,
			"vehicles":  formatVehicles(frame.Vehicles),
			"metrics":   formatRouteMetrics(frame.Metrics),
			"insights":  frame.Insights,
		})
	}

	return map[string]any{
		"routeId":     replay.RouteID,
		"from":        formatTime(replay.From),
		"to":          formatTime(replay.To),
		"stepSeconds": replay.StepSeconds,
		"frames":      frames,
	}
}

func formatIngestResult(result models.IngestResult) map[string]any {
	return map[string]any{
		"insertedCount": result.InsertedCount,
		"timestamp":     formatTime(result.Timestamp),
		"source":        result.Source,
	}
}

func formatTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format(time.RFC3339)
}

func stringPtrValue(value *string) any {
	if value == nil {
		return nil
	}
	return *value
}

func floatPtrValue(value *float64) any {
	if value == nil {
		return nil
	}
	return *value
}

func trailWindowMinutes(args map[string]any) int {
	minutes, ok := args["minutes"].(int)
	if !ok || minutes <= 0 {
		return defaultTrailWindowMinutes
	}
	if minutes > maxTrailWindowMinutes {
		return maxTrailWindowMinutes
	}
	return minutes
}

func historyRangeMinutes(args map[string]any) int {
	minutes, ok := args["rangeMinutes"].(int)
	if !ok || minutes <= 0 {
		return defaultHistoryRangeMinutes
	}
	if minutes > maxHistoryRangeMinutes {
		return maxHistoryRangeMinutes
	}
	return minutes
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}
