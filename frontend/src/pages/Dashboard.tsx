import { RefreshCcw } from "lucide-react";
import { useCallback, useEffect, useMemo, useState } from "react";
import {
  fetchDashboard,
  fetchRouteHistory,
  fetchRouteReplay,
  triggerMockIngestion
} from "../api/graphql";
import { Header } from "../components/Header";
import { InsightPanel } from "../components/InsightPanel";
import { ReplayControls } from "../components/ReplayControls";
import { RouteHistoryPanel } from "../components/RouteHistoryPanel";
import { RouteSelector } from "../components/RouteSelector";
import { SummaryCards } from "../components/SummaryCards";
import { VehicleMap } from "../components/VehicleMap";
import { VehicleTable } from "../components/VehicleTable";
import type { Theme } from "../types/theme";
import type { DashboardData, ReplayFrame, RouteHistory, RouteReplay } from "../types/transit";
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
  const [historyRangeMinutes, setHistoryRangeMinutes] = useState(30);
  const [history, setHistory] = useState<RouteHistory | null>(null);
  const [historyError, setHistoryError] = useState<string | null>(null);
  const [historyLoading, setHistoryLoading] = useState(true);
  const [historyRefreshKey, setHistoryRefreshKey] = useState(0);
  const [isReplayMode, setIsReplayMode] = useState(false);
  const [replay, setReplay] = useState<RouteReplay | null>(null);
  const [replayError, setReplayError] = useState<string | null>(null);
  const [replayLoading, setReplayLoading] = useState(false);
  const [replayFrameIndex, setReplayFrameIndex] = useState(0);
  const [isReplayPlaying, setIsReplayPlaying] = useState(false);
  const [replaySpeed, setReplaySpeed] = useState(1);
  const [replayTargetTimestamp, setReplayTargetTimestamp] = useState<string | null>(null);

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

  useEffect(() => {
    let cancelled = false;

    async function loadHistory() {
      setHistoryLoading(true);
      setHistoryError(null);

      try {
        const nextHistory = await fetchRouteHistory(selectedRoute, historyRangeMinutes);
        if (!cancelled) {
          setHistory(nextHistory);
        }
      } catch (caught) {
        if (!cancelled) {
          setHistoryError(
            caught instanceof Error ? caught.message : "Failed to load route history."
          );
        }
      } finally {
        if (!cancelled) {
          setHistoryLoading(false);
        }
      }
    }

    void loadHistory();
    return () => {
      cancelled = true;
    };
  }, [historyRangeMinutes, historyRefreshKey, selectedRoute]);

  useEffect(() => {
    if (!isReplayMode) {
      return;
    }

    let cancelled = false;

    async function loadReplay() {
      setReplayLoading(true);
      setReplayError(null);

      try {
        const nextReplay = await fetchRouteReplay(selectedRoute, historyRangeMinutes);
        if (!cancelled) {
          setReplay(nextReplay);
          setReplayFrameIndex(0);
        }
      } catch (caught) {
        if (!cancelled) {
          setReplayError(caught instanceof Error ? caught.message : "Failed to load replay frames.");
        }
      } finally {
        if (!cancelled) {
          setReplayLoading(false);
        }
      }
    }

    void loadReplay();
    return () => {
      cancelled = true;
    };
  }, [historyRangeMinutes, isReplayMode, selectedRoute]);

  const selectedRouteDetails = useMemo(
    () => data?.routes.find((route) => route.id === selectedRoute),
    [data?.routes, selectedRoute]
  );

  const routeReplay = replay?.routeId === selectedRoute ? replay : null;
  const replayFrame =
    isReplayMode && routeReplay ? routeReplay.frames[replayFrameIndex] : undefined;
  const isShowingReplay = Boolean(replayFrame?.hasData);
  const displayedVehicles = isShowingReplay ? replayFrame!.vehicles : data?.vehicles ?? [];
  const displayedMetrics = isShowingReplay ? replayFrame!.metrics : data?.routeMetrics;
  const displayedInsights = isShowingReplay ? replayFrame!.insights : data?.routeInsights ?? [];

  useEffect(() => {
    if (!routeReplay || !replayTargetTimestamp) {
      return;
    }

    const targetTime = new Date(replayTargetTimestamp).getTime();
    const closestFrameIndex = routeReplay.frames.reduce((closestIndex, frame, index) => {
      const currentDistance = Math.abs(new Date(frame.timestamp).getTime() - targetTime);
      const closestDistance = Math.abs(
        new Date(routeReplay.frames[closestIndex].timestamp).getTime() - targetTime
      );
      return currentDistance < closestDistance ? index : closestIndex;
    }, 0);

    setReplayFrameIndex(closestFrameIndex);
    setReplayTargetTimestamp(null);
  }, [replayTargetTimestamp, routeReplay]);

  useEffect(() => {
    if (!isReplayMode || !isReplayPlaying || !routeReplay || routeReplay.frames.length < 2) {
      return;
    }

    const finalFrameIndex = routeReplay.frames.length - 1;
    const interval = window.setInterval(() => {
      setReplayFrameIndex((currentIndex) => Math.min(currentIndex + 1, finalFrameIndex));
    }, Math.round(900 / replaySpeed));

    return () => window.clearInterval(interval);
  }, [isReplayMode, isReplayPlaying, replaySpeed, routeReplay]);

  useEffect(() => {
    if (
      isReplayPlaying &&
      routeReplay &&
      replayFrameIndex >= routeReplay.frames.length - 1
    ) {
      setIsReplayPlaying(false);
    }
  }, [isReplayPlaying, replayFrameIndex, routeReplay]);

  async function handleManualIngest() {
    setRefreshing(true);
    setError(null);

    try {
      await triggerMockIngestion();
      await loadDashboard(true);
      setHistoryRefreshKey((current) => current + 1);
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : "Mock ingestion failed.");
    } finally {
      setRefreshing(false);
    }
  }

  function handleRouteSelect(routeId: string) {
    setSelectedRoute(routeId);
    setSelectedVehicleId(null);
    setIsReplayMode(false);
    setIsReplayPlaying(false);
    setReplay(null);
    setReplayFrameIndex(0);
    setReplayTargetTimestamp(null);
  }

  function handleHistoryRangeChange(rangeMinutes: number) {
    setHistoryRangeMinutes(rangeMinutes);
    setIsReplayPlaying(false);
    setReplay(null);
    setReplayFrameIndex(0);
    setReplayTargetTimestamp(null);
  }

  function handleHistoryPointSelect(timestamp: string) {
    setReplayTargetTimestamp(timestamp);
    setIsReplayPlaying(false);
    setIsReplayMode(true);
  }

  function handleEnterReplay() {
    setReplayFrameIndex(0);
    setReplayTargetTimestamp(null);
    setIsReplayPlaying(false);
    setIsReplayMode(true);
  }

  function handleExitReplay() {
    setIsReplayMode(false);
    setIsReplayPlaying(false);
    setReplayTargetTimestamp(null);
  }

  function handleReplayFrameIndexChange(frameIndex: number) {
    setReplayFrameIndex(frameIndex);
    setIsReplayPlaying(false);
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
            routes={
              data?.routes ?? [
                { id: defaultRoute, shortName: "24", longName: "Sherbrooke", shape: [] }
              ]
            }
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
            {displayedMetrics ? <SummaryCards isReplay={isShowingReplay} metrics={displayedMetrics} /> : null}
            <section className="dashboard-grid">
              <VehicleMap
                isReplayMode={isShowingReplay}
                replayFrameIndex={replayFrameIndex}
                replayFrames={isShowingReplay ? routeReplay?.frames ?? [] : []}
                replayTimestamp={replayFrame?.timestamp}
                routeColor={selectedRouteDetails?.color}
                routeShape={selectedRouteDetails?.shape ?? []}
                selectedVehicleId={selectedVehicleId}
                vehicleTrails={data.vehicleTrails}
                vehicles={displayedVehicles}
              />
              <InsightPanel
                insights={displayedInsights}
                replayTimestamp={isShowingReplay ? replayFrame?.timestamp : undefined}
              />
            </section>
            <RouteHistoryPanel
              error={historyError}
              history={history}
              loading={historyLoading}
              onRangeChange={handleHistoryRangeChange}
              onSelectPoint={handleHistoryPointSelect}
              rangeMinutes={historyRangeMinutes}
            />
            <ReplayControls
              error={replayError}
              frameIndex={replayFrameIndex}
              isLoading={replayLoading}
              isPlaying={isReplayPlaying}
              isReplayMode={isReplayMode}
              onEnterReplay={handleEnterReplay}
              onExitReplay={handleExitReplay}
              onFrameIndexChange={handleReplayFrameIndexChange}
              onSpeedChange={setReplaySpeed}
              onTogglePlayback={() => setIsReplayPlaying((playing) => !playing)}
              replay={routeReplay}
              speed={replaySpeed}
            />
            <VehicleTable
              historical={isShowingReplay}
              onSelectVehicle={setSelectedVehicleId}
              selectedVehicleId={selectedVehicleId}
              vehicles={displayedVehicles}
            />
          </>
        ) : null}
      </main>
    </div>
  );
}
