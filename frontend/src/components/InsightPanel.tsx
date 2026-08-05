import { Sparkles } from "lucide-react";
import { formatDateTime } from "../utils/format";

interface InsightPanelProps {
  insights: string[];
  replayTimestamp?: string;
}

export function InsightPanel({ insights, replayTimestamp }: InsightPanelProps) {
  return (
    <section className="insight-panel" aria-label="Operational insights">
      <div className="panel-heading">
        <div>
          <h2>{replayTimestamp ? "Replay insights" : "Operational insights"}</h2>
          {replayTimestamp ? <span>{formatDateTime(replayTimestamp)}</span> : null}
        </div>
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
