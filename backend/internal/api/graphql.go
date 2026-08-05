package api

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/graphql-go/graphql"
	"github.com/graphql-go/handler"

	"github.com/lamda/transitops-montreal/backend/internal/ingest"
	"github.com/lamda/transitops-montreal/backend/internal/metrics"
	"github.com/lamda/transitops-montreal/backend/internal/models"
)

type RouteStore interface {
	Routes(ctx context.Context) ([]models.Route, error)
	LatestVehicleSnapshots(ctx context.Context, routeID string) ([]models.VehicleSnapshot, error)
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
			"ACTIVE":         &graphql.EnumValueConfig{Value: string(models.VehicleStatusActive)},
			"STALE":          &graphql.EnumValueConfig{Value: string(models.VehicleStatusStale)},
			"BUNCHING_RISK":  &graphql.EnumValueConfig{Value: string(models.VehicleStatusBunchingRisk)},
			"DELAYED":        &graphql.EnumValueConfig{Value: string(models.VehicleStatusDelayed)},
			"UNKNOWN":        &graphql.EnumValueConfig{Value: string(models.VehicleStatusUnknown)},
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

	routeType := graphql.NewObject(graphql.ObjectConfig{
		Name: "Route",
		Fields: graphql.Fields{
			"id":        &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
			"shortName": &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
			"longName":  &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
			"color":     &graphql.Field{Type: graphql.String},
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

	metricsType := graphql.NewObject(graphql.ObjectConfig{
		Name: "RouteMetrics",
		Fields: graphql.Fields{
			"routeId":                   &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
			"activeVehicleCount":       &graphql.Field{Type: graphql.NewNonNull(graphql.Int)},
			"staleVehicleCount":        &graphql.Field{Type: graphql.NewNonNull(graphql.Int)},
			"bunchingEventCount":       &graphql.Field{Type: graphql.NewNonNull(graphql.Int)},
			"largestHeadwayGapMinutes": &graphql.Field{Type: graphql.NewNonNull(graphql.Float)},
			"averageSpacingMinutes":    &graphql.Field{Type: graphql.NewNonNull(graphql.Float)},
			"healthStatus":             &graphql.Field{Type: graphql.NewNonNull(healthStatusType)},
			"lastUpdated":              &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
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

func formatRoutes(routes []models.Route) []map[string]any {
	formatted := make([]map[string]any, 0, len(routes))
	for _, route := range routes {
		formatted = append(formatted, map[string]any{
			"id":        route.ID,
			"shortName": route.ShortName,
			"longName":  route.LongName,
			"color":     stringPtrValue(route.Color),
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

func formatRouteMetrics(routeMetrics models.RouteMetrics) map[string]any {
	return map[string]any{
		"routeId":                   routeMetrics.RouteID,
		"activeVehicleCount":       routeMetrics.ActiveVehicleCount,
		"staleVehicleCount":        routeMetrics.StaleVehicleCount,
		"bunchingEventCount":       routeMetrics.BunchingEventCount,
		"largestHeadwayGapMinutes": routeMetrics.LargestHeadwayGapMinutes,
		"averageSpacingMinutes":    routeMetrics.AverageSpacingMinutes,
		"healthStatus":             routeMetrics.HealthStatus,
		"lastUpdated":              formatTime(routeMetrics.LastUpdated),
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
