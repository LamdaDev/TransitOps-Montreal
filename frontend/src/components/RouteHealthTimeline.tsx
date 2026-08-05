import type { HealthStatus, RouteHealthEvent, RouteHistory } from "../types/transit";
import { formatShortTime } from "../utils/format";

interface RouteHealthTimelineProps {
  history: RouteHistory;
  onSelectPoint: (timestamp: string) => void;
}

function statusClass(status: HealthStatus, hasData: boolean): string {
  if (!hasData) {
    return "timeline-status-empty";
  }
  return `timeline-status-${status.toLowerCase()}`;
}

function statusLabel(status: HealthStatus, hasData: boolean): string {
  return hasData ? status.toLowerCase() : "no telemetry";
}

function eventLabel(events: RouteHealthEvent[]): string {
  if (events.length === 0) {
    return "";
  }
  return ` ${events.map((event) => event.description).join(" ")}`;
}

export function RouteHealthTimeline({ history, onSelectPoint }: RouteHealthTimelineProps) {
  const eventsByTimestamp = new Map<string, RouteHealthEvent[]>();
  for (const event of history.events) {
    const matchingEvents = eventsByTimestamp.get(event.timestamp) ?? [];
    matchingEvents.push(event);
    eventsByTimestamp.set(event.timestamp, matchingEvents);
  }

  return (
    <div className="route-health-timeline">
      <div className="timeline-legend" aria-label="Route health legend">
        <span className="timeline-legend-item timeline-status-healthy">Healthy</span>
        <span className="timeline-legend-item timeline-status-watch">Watch</span>
        <span className="timeline-legend-item timeline-status-degraded">Degraded</span>
        <span className="timeline-legend-item timeline-status-empty">No data</span>
      </div>
      <div className="health-timeline" aria-label="Route health by minute" role="list">
        {history.points.map((point) => {
          const events = eventsByTimestamp.get(point.timestamp) ?? [];
          const label = `${formatShortTime(point.timestamp)}: ${statusLabel(
            point.healthStatus,
            point.hasData
          )}.${eventLabel(events)} Select to jump to this moment in replay.`;

          return (
            <button
              aria-label={label}
              className={`timeline-segment ${statusClass(point.healthStatus, point.hasData)}`}
              key={point.timestamp}
              onClick={() => onSelectPoint(point.timestamp)}
              title={label}
              type="button"
            >
              {events.length > 0 ? (
                <span
                  aria-hidden="true"
                  className={`timeline-event-marker timeline-event-${events[0].type.toLowerCase()}`}
                />
              ) : null}
            </button>
          );
        })}
      </div>
      <div className="timeline-axis" aria-hidden="true">
        <span>{formatShortTime(history.from)}</span>
        <span>{formatShortTime(history.to)}</span>
      </div>
    </div>
  );
}
