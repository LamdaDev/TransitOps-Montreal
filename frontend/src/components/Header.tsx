import { Moon, RadioTower, Sun } from "lucide-react";
import type { Theme } from "../types/theme";

interface HeaderProps {
  lastUpdated?: string;
  theme: Theme;
  onThemeChange: (theme: Theme) => void;
}

export function Header({ lastUpdated, theme, onThemeChange }: HeaderProps) {
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
      <div className="header-actions">
        <div className="theme-switch" aria-label="Color theme" role="group">
          <button
            aria-pressed={theme === "light"}
            className="theme-option"
            onClick={() => onThemeChange("light")}
            type="button"
          >
            <Sun size={15} aria-hidden="true" />
            <span>Light</span>
          </button>
          <button
            aria-pressed={theme === "navy"}
            className="theme-option"
            onClick={() => onThemeChange("navy")}
            type="button"
          >
            <Moon size={15} aria-hidden="true" />
            <span>Navy</span>
          </button>
        </div>
        <div className="header-meta">
          <span>GraphQL</span>
          <strong>{lastUpdated ?? "Waiting for snapshots"}</strong>
        </div>
      </div>
    </header>
  );
}
