import { AlertTriangle, BusFront, Clock3, Gauge, Radio, ShieldCheck } from "lucide-react";
import type { RouteMetrics } from "../types/transit";
import { formatDateTime } from "../utils/format";

interface SummaryCardsProps {
  metrics: RouteMetrics;
}

export function SummaryCards({ metrics }: SummaryCardsProps) {
  const cards = [
    {
      label: "Active vehicles",
      value: metrics.activeVehicleCount,
      icon: BusFront
    },
    {
      label: "Last updated",
      value: formatDateTime(metrics.lastUpdated),
      icon: Radio
    },
    {
      label: "Bunching events",
      value: metrics.bunchingEventCount,
      icon: AlertTriangle
    },
    {
      label: "Largest gap",
      value: `${metrics.largestHeadwayGapMinutes.toFixed(0)} min`,
      icon: Clock3
    },
    {
      label: "Stale vehicles",
      value: metrics.staleVehicleCount,
      icon: Gauge
    },
    {
      label: "Route health",
      value: metrics.healthStatus,
      icon: ShieldCheck
    }
  ];

  return (
    <section className="summary-grid" aria-label="Fleet summary">
      {cards.map((card) => {
        const Icon = card.icon;
        return (
          <article className="summary-card" key={card.label}>
            <span className="summary-icon" aria-hidden="true">
              <Icon size={18} />
            </span>
            <span className="summary-label">{card.label}</span>
            <strong>{card.value}</strong>
          </article>
        );
      })}
    </section>
  );
}
