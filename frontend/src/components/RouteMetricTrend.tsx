import type { RouteHistoryPoint } from "../types/transit";

interface RouteMetricTrendProps {
  points: RouteHistoryPoint[];
}

const chartWidth = 680;
const chartHeight = 170;
const chartPadding = 16;

export function RouteMetricTrend({ points }: RouteMetricTrendProps) {
  const dataPoints = points.filter((point) => point.hasData);
  if (dataPoints.length === 0) {
    return null;
  }

  const values = dataPoints.map((point) => point.largestHeadwayGapMinutes);
  const maxValue = Math.max(1, ...values);
  const minValue = Math.min(...values);
  const latestValue = values[values.length - 1];
  const drawableWidth = chartWidth - chartPadding * 2;
  const drawableHeight = chartHeight - chartPadding * 2;
  const coordinates = dataPoints.map((point, index) => {
    const x =
      chartPadding +
      (dataPoints.length === 1 ? drawableWidth / 2 : (index / (dataPoints.length - 1)) * drawableWidth);
    const y = chartPadding + (1 - point.largestHeadwayGapMinutes / maxValue) * drawableHeight;
    return [x, y] as const;
  });
  const linePoints = coordinates.map(([x, y]) => `${x},${y}`).join(" ");
  const areaPoints = [
    `${coordinates[0][0]},${chartHeight - chartPadding}`,
    linePoints,
    `${coordinates[coordinates.length - 1][0]},${chartHeight - chartPadding}`
  ].join(" ");

  return (
    <section className="metric-trend" aria-label="Largest estimated headway gap trend">
      <div className="trend-heading">
        <div>
          <span>Largest estimated headway gap</span>
          <strong>{latestValue.toFixed(0)} min now</strong>
        </div>
        <span>Peak {maxValue.toFixed(0)} min</span>
      </div>
      <svg
        aria-label="Largest estimated headway gap over the selected time range"
        className="trend-chart"
        preserveAspectRatio="none"
        role="img"
        viewBox={`0 0 ${chartWidth} ${chartHeight}`}
      >
        <line
          stroke="var(--border)"
          strokeDasharray="4 6"
          strokeWidth="1"
          x1={chartPadding}
          x2={chartWidth - chartPadding}
          y1={chartHeight - chartPadding}
          y2={chartHeight - chartPadding}
        />
        <polygon fill="var(--accent-soft)" points={areaPoints} />
        <polyline
          fill="none"
          points={linePoints}
          stroke="var(--accent)"
          strokeLinecap="round"
          strokeLinejoin="round"
          strokeWidth="4"
        />
      </svg>
      <div className="trend-summary">
        <span>Low {minValue.toFixed(0)} min</span>
        <span>{dataPoints.length} sampled states</span>
      </div>
    </section>
  );
}
