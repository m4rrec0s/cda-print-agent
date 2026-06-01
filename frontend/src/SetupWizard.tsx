import { useEffect, useState } from "react";
import { GetAgentConfig, SaveAgentConfig } from "../wailsjs/go/main/App";

interface Props {
  onComplete: () => void;
  onCancel?: () => void;
}

export function SetupWizard({ onComplete, onCancel }: Props) {
  const [wsUrl, setWsUrl] = useState("");
  const [apiUrl, setApiUrl] = useState("");
  const [agentKey, setAgentKey] = useState("");
  const [hotFolderPath, setHotFolderPath] = useState("");
  const [error, setError] = useState("");
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    GetAgentConfig().then((cfg) => {
      if (cfg.wsUrl) setWsUrl(cfg.wsUrl);
      if (cfg.apiUrl) setApiUrl(cfg.apiUrl);
      if (cfg.agentKey) setAgentKey(cfg.agentKey);
      if (cfg.hotFolderPath) setHotFolderPath(cfg.hotFolderPath);
    });
  }, []);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError("");
    setSaving(true);
    try {
      await SaveAgentConfig(wsUrl, apiUrl, agentKey, hotFolderPath);
      onComplete();
    } catch (err) {
      setError(String(err));
    } finally {
      setSaving(false);
    }
  };

  const inputStyle: React.CSSProperties = {
    background: "#0d0d0d",
    border: "1px solid #1a1a1a",
    borderRadius: 6,
    padding: "8px 10px",
    color: "#d4d4d4",
    fontFamily: '"JetBrains Mono", monospace',
    fontSize: 12,
    outline: "none",
    width: "100%",
  };

  return (
    <main
      style={{
        display: "flex",
        flexDirection: "column",
        height: "100vh",
        color: "#d4d4d4",
        fontFamily:
          '"Geist", -apple-system, "Segoe UI Variable", system-ui, sans-serif',
        background: "#0d0d0d",
      }}
    >
      <header
        style={{
          display: "flex",
          alignItems: "center",
          justifyContent: "center",
          padding: "12px 14px",
          borderBottom: "1px solid #1a1a1a",
          minHeight: 42,
        }}
      >
        <span
          style={{
            fontSize: 9,
            letterSpacing: "0.16em",
            fontWeight: 600,
            color: "#777",
          }}
        >
          CONFIGURAÇÃO
        </span>
      </header>

      <form
        onSubmit={handleSubmit}
        style={{
          flex: 1,
          display: "flex",
          flexDirection: "column",
          overflow: "hidden",
        }}
      >
        <div style={{ textAlign: "center", padding: "20px 20px 12px" }}>
          <h1
            style={{
              margin: 0,
              fontSize: 15,
              fontWeight: 700,
              color: "#e8e8e8",
              letterSpacing: "0.02em",
            }}
          >
            Cesto d'Amore
          </h1>
          <p
            style={{
              margin: "4px 0 0",
              color: "#555",
              fontSize: 11,
              letterSpacing: "0.06em",
            }}
          >
            CONFIGURAÇÃO DO AGENTE
          </p>
        </div>

        <div style={{ flex: 1, overflowY: "auto", padding: "0 20px 12px" }}>
          <div
            className="printer-row"
            style={{
              flexDirection: "column",
              alignItems: "stretch",
              gap: 6,
              padding: "12px 14px",
            }}
          >
            <label
              className="printer-role"
              style={{ fontSize: 10, marginBottom: 0 }}
            >
              SERVIDOR WEBSOCKET
            </label>
            <input
              style={inputStyle}
              value={wsUrl}
              onChange={(e) => setWsUrl(e.target.value)}
              placeholder="wss://api.cestodamore.com.br/ws/print-agent"
            />
          </div>

          <div
            className="printer-row"
            style={{
              flexDirection: "column",
              alignItems: "stretch",
              gap: 6,
              padding: "12px 14px",
            }}
          >
            <label
              className="printer-role"
              style={{ fontSize: 10, marginBottom: 0 }}
            >
              URL DA API
            </label>
            <input
              style={inputStyle}
              value={apiUrl}
              onChange={(e) => setApiUrl(e.target.value)}
              placeholder="https://api.cestodamore.com.br"
            />
          </div>

          <div
            className="printer-row"
            style={{
              flexDirection: "column",
              alignItems: "stretch",
              gap: 6,
              padding: "12px 14px",
            }}
          >
            <label
              className="printer-role"
              style={{ fontSize: 10, marginBottom: 0 }}
            >
              CHAVE DO AGENTE
            </label>
            <input
              style={inputStyle}
              type="password"
              value={agentKey}
              onChange={(e) => setAgentKey(e.target.value)}
              placeholder="Opcional"
            />
          </div>

          <div
            className="printer-row"
            style={{
              flexDirection: "column",
              alignItems: "stretch",
              gap: 6,
              padding: "12px 14px",
            }}
          >
            <label
              className="printer-role"
              style={{ fontSize: 10, marginBottom: 0 }}
            >
              PASTA DE IMPRESSÃO
            </label>
            <input
              style={inputStyle}
              value={hotFolderPath}
              onChange={(e) => setHotFolderPath(e.target.value)}
              placeholder="C:\PrintHotFolder"
            />
          </div>

          {error && (
            <div
              className="printer-row"
              style={{
                background: "#1a0d0d",
                border: "1px solid #2a1515",
                color: "#ef4444",
                fontSize: 11,
                justifyContent: "center",
                marginTop: 8,
              }}
            >
              {error}
            </div>
          )}
        </div>

        <footer
          className="actions"
          style={{
            borderTop: "1px solid #1a1a1a",
          }}
        >
          {onCancel && (
            <button type="button" className="btn-ghost" onClick={onCancel}>
              CANCELAR
            </button>
          )}
          <button
            type="submit"
            className="btn-primary"
            disabled={saving}
            style={{
              gridColumn: onCancel ? undefined : "1 / -1",
              opacity: saving ? 0.6 : 1,
              cursor: saving ? "not-allowed" : "pointer",
            }}
          >
            {saving ? "SALVANDO..." : "SALVAR E CONECTAR"}
          </button>
        </footer>
      </form>
    </main>
  );
}
