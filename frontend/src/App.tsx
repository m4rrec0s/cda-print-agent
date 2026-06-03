import { useEffect, useState } from "react";
import { IsConfigured, GetVersion } from "../wailsjs/go/main/App";
import "./App.css";
import { SetupWizard } from "./SetupWizard";
import { AgentPanel } from "./AgentPanel";
import { SettingsPage } from "./SettingsPage";
import { ThemePage } from "./ThemePage";
import { AboutPage } from "./AboutPage";
import { ThemeProvider } from "./ThemeContext";

type View =
  | "loading"
  | "wizard"
  | "panel"
  | "settings"
  | "settings-wizard"
  | "settings-theme"
  | "settings-about";

export default function App() {
  const [view, setView] = useState<View>("loading");
  const [version, setVersion] = useState("dev");

  useEffect(() => {
    Promise.all([IsConfigured(), GetVersion()]).then(([ok, ver]) => {
      setVersion(ver || "dev");
      setView(ok ? "panel" : "wizard");
    });
  }, []);

  if (view === "loading") {
    return (
      <ThemeProvider>
        <main
          style={{
            display: "flex",
            alignItems: "center",
            justifyContent: "center",
            height: "100vh",
            background: "var(--bg-base)",
            color: "var(--text-muted)",
            fontSize: "13px",
            fontFamily: "Geist, -apple-system, system-ui, sans-serif",
          }}
        >
          Carregando...
        </main>
      </ThemeProvider>
    );
  }

  if (view === "wizard") {
    return (
      <ThemeProvider>
        <SetupWizard onComplete={() => setView("panel")} />
      </ThemeProvider>
    );
  }

  if (view === "settings-wizard") {
    return (
      <ThemeProvider>
        <SetupWizard
          onComplete={() => setView("panel")}
          onCancel={() => setView("settings")}
        />
      </ThemeProvider>
    );
  }

  if (view === "settings-theme") {
    return (
      <ThemeProvider>
        <ThemePage onBack={() => setView("settings")} />
      </ThemeProvider>
    );
  }

  if (view === "settings-about") {
    return (
      <ThemeProvider>
        <AboutPage onBack={() => setView("settings")} version={version} />
      </ThemeProvider>
    );
  }

  if (view === "settings") {
    return (
      <ThemeProvider>
        <SettingsPage
          onBack={() => setView("panel")}
          onOpenBasicConfig={() => setView("settings-wizard")}
          onOpenTheme={() => setView("settings-theme")}
          onOpenAbout={() => setView("settings-about")}
        />
      </ThemeProvider>
    );
  }

  return (
    <ThemeProvider>
      <AgentPanel onReconfigure={() => setView("settings")} />
    </ThemeProvider>
  );
}
