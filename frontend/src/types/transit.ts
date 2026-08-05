export type VehicleStatus =
  | "ACTIVE"
  | "STALE"
  | "BUNCHING_RISK"
  | "DELAYED"
  | "UNKNOWN";

export interface Route {
  id: string;
  shortName: string;
  longName: string;
  color?: string | null;
}

export interface Vehicle {
  id: number;
  vehicleId: string;
  routeId: string;
  tripId?: string | null;
  latitude: number;
  longitude: number;
  speed?: number | null;
  routeProgress?: number | null;
  timestamp: string;
  source: string;
  createdAt: string;
  status: VehicleStatus;
}

export interface RouteMetrics {
  routeId: string;
  activeVehicleCount: number;
  staleVehicleCount: number;
  bunchingEventCount: number;
  largestHeadwayGapMinutes: number;
  averageSpacingMinutes: number;
  healthStatus: "HEALTHY" | "WATCH" | "DEGRADED";
  lastUpdated: string;
}

export interface DashboardData {
  routes: Route[];
  vehicles: Vehicle[];
  routeMetrics: RouteMetrics;
  routeInsights: string[];
}

export interface IngestResult {
  insertedCount: number;
  timestamp: string;
  source: string;
}
