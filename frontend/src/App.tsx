import { useEffect, useState } from "react";
import { Dashboard } from "./pages/Dashboard";
import type { Theme } from "./types/theme";
import "./App.css";

const themeStorageKey = "transitops-theme";

function getInitialTheme(): Theme {
  return window.localStorage.getItem(themeStorageKey) === "navy" ? "navy" : "light";
}

export default function App() {
  const [theme, setTheme] = useState<Theme>(getInitialTheme);

  useEffect(() => {
    document.documentElement.dataset.theme = theme;
    window.localStorage.setItem(themeStorageKey, theme);
  }, [theme]);

  return <Dashboard theme={theme} onThemeChange={setTheme} />;
}
