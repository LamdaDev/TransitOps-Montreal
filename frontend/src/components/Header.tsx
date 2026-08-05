import { RadioTower } from "lucide-react";

interface HeaderProps {
  lastUpdated?: string;
}

export function Header({ lastUpdated }: HeaderProps) {
  return (
    <header className="app-header">
      <div className="brand">
        <span className="brand-mark" aria-hidden="true">
          <RadioTower size={22} />
        </span>
        <div>
          <h1>TransitOps Montréal</h1>
          <p>STM route reliability dashboard</p>
        </div>
      </div>
      <div className="header-meta">
        <span>GraphQL</span>
        <strong>{lastUpdated ?? "Waiting for snapshots"}</strong>
      </div>
    </header>
  );
}
