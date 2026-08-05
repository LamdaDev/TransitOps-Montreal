import { Activity, AlertTriangle, Clock3 } from "lucide-react";
import type { RouteHistory } from "../types/transit";
import { formatShortTime } from "../utils/format";
import { RouteHealthTimeline } from "./RouteHealthTimeline";
import { RouteMetricTrend } from "./RouteMetricTrend";

interface RouteHistoryPanelProps {
  error: string | null;
  history: RouteHistory | null;
  loading: boolean;
  rangeMinutes: number;
  onRangeChange: (rangeMinutes: number) => void;
  onSelectPoint: (timestamp: string) => void;
}

const historyRanges = [30, 60];

export function RouteHistoryPanel({
  error,
  history,
  loading,
  rangeMinutes,
  onRangeChange,
  onSelectPoint
}: RouteHistoryPanelProps) {
  const hasHistory = history?.points.some((point) => point.hasData) ?? false;
  const recentEvents = history ? [...history.events].slice(-4).reverse() : [];

  return (
    <section className="history-panel" aria-label="Historical route health">
      <div className="history-heading">
        <div>
          <span className="section-kicker">Historical operations</span>
          <h2>Route health timeline</h2>
        </div>
        <div className="history-range" aria-label="History range" role="group">
          {historyRanges.map((range) => (
            <button
              aria-pressed={rangeMinutes === range}
              className="history-range-option"
              key={range}
              onClick={() => onRangeChange(range)}
              type="button"
            >
              Last {range === 60 ? "hour" : `${range} min`}
            </button>
          ))}
        </div>
      </div>

      {loading ? <div className="history-state">Loading stored snapshots...</div> : null}
      {error ? <div className="history-state history-state-error">{error}</div> : null}
      {!loading && !error && !hasHistory ? (
        <div className="history-state">
          No historical snapshots are available for this range yet.
        </div>
      ) : null}

      {!loading && !error && hasHistory && history ? (
        <div className="history-content">
          <RouteHealthTimeline history={history} onSelectPoint={onSelectPoint} />
          <div className="history-detail-grid">
            <RouteMetricTrend points={history.points} />
            <aside className="history-events" aria-label="Historical operational events">
              <div className="history-events-heading">
                <div>
                  <span>Detected events</span>
                  <strong>{history.events.length}</strong>
                </div>
                <AlertTriangle size={18} aria-hidden="true" />
              </div>
              {recentEvents.length > 0 ? (
                <ul>
                  {recentEvents.map((event) => (
                    <li key={`${event.timestamp}-${event.type}`}>
                      <span className={`history-event-icon history-event-${event.type.toLowerCase()}`}>
                        {event.type === "BUNCHING_RISK" ? <Activity size={14} /> : <Clock3 size={14} />}
                      </span>
                      <span>
                        <strong>{formatShortTime(event.timestamp)}</strong>
                        {event.description}
                      </span>
                    </li>
                  ))}
                </ul>
              ) : (
                <p>No bunching or stale-telemetry starts in this range.</p>
              )}
            </aside>
          </div>
          <p className="history-hint">Select a timeline segment to jump to that point in replay.</p>
        </div>
      ) : null}
    </section>
  );
}
