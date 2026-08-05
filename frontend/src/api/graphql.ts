import type { DashboardData, IngestResult } from "../types/transit";

const GRAPHQL_URL =
  import.meta.env.VITE_GRAPHQL_URL ?? "http://localhost:8080/graphql";

interface GraphQLResponse<T> {
  data?: T;
  errors?: Array<{ message: string }>;
}

async function graphQLRequest<T>(
  query: string,
  variables?: Record<string, unknown>
): Promise<T> {
  const response = await fetch(GRAPHQL_URL, {
    method: "POST",
    headers: {
      "Content-Type": "application/json"
    },
    body: JSON.stringify({ query, variables })
  });

  if (!response.ok) {
    throw new Error(`GraphQL request failed with HTTP ${response.status}`);
  }

  const payload = (await response.json()) as GraphQLResponse<T>;
  if (payload.errors?.length) {
    throw new Error(payload.errors.map((error) => error.message).join("; "));
  }
  if (!payload.data) {
    throw new Error("GraphQL response did not include data.");
  }

  return payload.data;
}

export async function fetchDashboard(routeId: string): Promise<DashboardData> {
  return graphQLRequest<DashboardData>(
    `
      query Dashboard($routeId: String!) {
        routes {
          id
          shortName
          longName
          color
          shape {
            latitude
            longitude
          }
        }
        vehicles(routeId: $routeId) {
          id
          vehicleId
          routeId
          tripId
          latitude
          longitude
          speed
          routeProgress
          timestamp
          source
          createdAt
          status
        }
        routeMetrics(routeId: $routeId) {
          routeId
          activeVehicleCount
          staleVehicleCount
          bunchingEventCount
          largestHeadwayGapMinutes
          averageSpacingMinutes
          healthStatus
          lastUpdated
        }
        routeInsights(routeId: $routeId)
        vehicleTrails(routeId: $routeId, minutes: 10) {
          vehicleId
          points {
            latitude
            longitude
            timestamp
          }
        }
      }
    `,
    { routeId }
  );
}

export async function triggerMockIngestion(): Promise<IngestResult> {
  const data = await graphQLRequest<{ ingestMock: IngestResult }>(
    `
      mutation IngestMock {
        ingestMock {
          insertedCount
          timestamp
          source
        }
      }
    `
  );

  return data.ingestMock;
}
