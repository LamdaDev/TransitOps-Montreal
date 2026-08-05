import { RefreshCcw } from "lucide-react";
import { useCallback, useEffect, useMemo, useState } from "react";
import { fetchDashboard, triggerMockIngestion } from "../api/graphql";
import { Header } from "../components/Header";
import { InsightPanel } from "../components/InsightPanel";
import { RouteSelector } from "../components/RouteSelector";
import { SummaryCards } from "../components/SummaryCards";
import { VehicleMap } from "../components/VehicleMap";
import { VehicleTable } from "../components/VehicleTable";
import type { Theme } from "../types/theme";
import type { DashboardData } from "../types/transit";
import { formatDateTime } from "../utils/format";

const defaultRoute = "24";

interface DashboardProps {
  theme: Theme;
  onThemeChange: (theme: Theme) => void;
}

export function Dashboard({ theme, onThemeChange }: DashboardProps) {
  const [selectedRoute, setSelectedRoute] = useState(defaultRoute);
  const [selectedVehicleId, setSelectedVehicleId] = useState<string | null>(null);
  const [data, setData] = useState<DashboardData | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);
  const [refreshing, setRefreshing] = useState(false);

  const loadDashboard = useCallback(
    async (silent = false) => {
      if (!silent) {
        setLoading(true);
      }
      setError(null);

      try {
        const dashboard = await fetchDashboard(selectedRoute);
        setData(dashboard);
      } catch (caught) {
        setError(caught instanceof Error ? caught.message : "Failed to load dashboard.");
      } finally {
        setLoading(false);
      }
    },
    [selectedRoute]
  );

  useEffect(() => {
    void loadDashboard();
    const interval = window.setInterval(() => {
      void loadDashboard(true);
    }, 10000);

    return () => window.clearInterval(interval);
  }, [loadDashboard]);

  const selectedRouteDetails = useMemo(
    () => data?.routes.find((route) => route.id === selectedRoute),
    [data?.routes, selectedRoute]
  );

  async function handleManualIngest() {
    setRefreshing(true);
    setError(null);

    try {
      await triggerMockIngestion();
      await loadDashboard(true);
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : "Mock ingestion failed.");
    } finally {
      setRefreshing(false);
    }
  }

  function handleRouteSelect(routeId: string) {
    setSelectedRoute(routeId);
    setSelectedVehicleId(null);
  }

  return (
    <div className="app-shell">
      <Header
        lastUpdated={
          data?.routeMetrics.lastUpdated
            ? formatDateTime(data.routeMetrics.lastUpdated)
            : undefined
        }
        onThemeChange={onThemeChange}
        theme={theme}
      />

      <main>
        <section className="toolbar" aria-label="Dashboard controls">
          <RouteSelector
            routes={data?.routes ?? [{ id: defaultRoute, shortName: "24", longName: "Sherbrooke" }]}
            selectedRoute={selectedRoute}
            onSelectRoute={handleRouteSelect}
          />
          <div className="route-title">
            <span>Selected</span>
            <strong>
              {selectedRouteDetails
                ? `${selectedRouteDetails.shortName} ${selectedRouteDetails.longName}`
                : "24 Sherbrooke"}
            </strong>
          </div>
          <button
            className="icon-button"
            disabled={refreshing}
            onClick={handleManualIngest}
            title="Trigger mock ingestion"
            type="button"
          >
            <RefreshCcw size={18} aria-hidden="true" />
            <span>{refreshing ? "Ingesting" : "Ingest mock"}</span>
          </button>
        </section>

        {error ? <div className="error-banner">{error}</div> : null}
        {loading ? <div className="loading-panel">Loading route operations...</div> : null}

        {data && !loading ? (
          <>
            <SummaryCards metrics={data.routeMetrics} />
            <section className="dashboard-grid">
              <VehicleMap
                selectedVehicleId={selectedVehicleId}
                vehicles={data.vehicles}
              />
              <InsightPanel insights={data.routeInsights} />
            </section>
            <VehicleTable
              onSelectVehicle={setSelectedVehicleId}
              selectedVehicleId={selectedVehicleId}
              vehicles={data.vehicles}
            />
          </>
        ) : null}
      </main>
    </div>
  );
}
