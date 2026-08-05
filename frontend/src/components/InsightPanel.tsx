import { Sparkles } from "lucide-react";

interface InsightPanelProps {
  insights: string[];
}

export function InsightPanel({ insights }: InsightPanelProps) {
  return (
    <section className="insight-panel" aria-label="Operational insights">
      <div className="panel-heading">
        <h2>Operational insights</h2>
        <Sparkles size={18} aria-hidden="true" />
      </div>
      <ul>
        {insights.map((insight) => (
          <li key={insight}>{insight}</li>
        ))}
      </ul>
    </section>
  );
}
