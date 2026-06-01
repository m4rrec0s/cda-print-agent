import { useEffect, useState } from "react";
import { IsConfigured } from "../wailsjs/go/main/App";
import "./App.css";
import { SetupWizard } from "./SetupWizard";
import { AgentPanel } from "./AgentPanel";

export default function App() {
  const [configured, setConfigured] = useState<boolean | null>(null);
  const [reconfiguring, setReconfiguring] = useState(false);

  useEffect(() => {
    IsConfigured().then(setConfigured);
  }, []);

  if (configured === null) {
    return (
      <main
        style={{
          display: "flex",
          alignItems: "center",
          justifyContent: "center",
          height: "100vh",
          background: "#171717",
          color: "#777",
          fontSize: "13px",
          fontFamily: "Inter, sans-serif",
        }}
      >
        Carregando...
      </main>
    );
  }

  if (!configured || reconfiguring) {
    return (
      <SetupWizard
        onComplete={() => {
          setConfigured(true);
          setReconfiguring(false);
        }}
        onCancel={configured ? () => setReconfiguring(false) : undefined}
      />
    );
  }

  return <AgentPanel onReconfigure={() => setReconfiguring(true)} />;
}
