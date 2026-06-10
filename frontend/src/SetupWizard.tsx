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
  const [deviceName, setDeviceName] = useState("");
  const [error, setError] = useState("");
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    GetAgentConfig().then((cfg) => {
      if (cfg.wsUrl) setWsUrl(cfg.wsUrl);
      if (cfg.apiUrl) setApiUrl(cfg.apiUrl);
      if (cfg.agentKey) setAgentKey(cfg.agentKey);
      if (cfg.hotFolderPath) setHotFolderPath(cfg.hotFolderPath);
      if (cfg.deviceName) setDeviceName(cfg.deviceName);
    });
  }, []);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError("");
    setSaving(true);
    try {
      await SaveAgentConfig(wsUrl, apiUrl, agentKey, hotFolderPath, deviceName);
      onComplete();
    } catch (err) {
      setError(String(err));
    } finally {
      setSaving(false);
    }
  };

  return (
    <main className="setup-shell">
      <header className="setup-topbar">
        <span className="brand">CONFIGURAÇÃO</span>
      </header>

      <form onSubmit={handleSubmit} className="setup-form">
        <section className="settings-panel">
          <div className="settings-fields">
            <div className="settings-field">
              <label>NOME DO DISPOSITIVO</label>
              <input
                value={deviceName}
                onChange={(e) => setDeviceName(e.target.value)}
                placeholder="Ex: PC da Loja"
              />
            </div>

            <div className="settings-field">
              <label>SERVIDOR WEBSOCKET</label>
              <input
                value={wsUrl}
                onChange={(e) => setWsUrl(e.target.value)}
                placeholder="wss://..."
              />
            </div>

            <div className="settings-field">
              <label>URL DA API</label>
              <input
                value={apiUrl}
                onChange={(e) => setApiUrl(e.target.value)}
                placeholder="https://..."
              />
            </div>

            <div className="settings-field">
              <label>CHAVE DO AGENTE</label>
              <input
                type="password"
                value={agentKey}
                onChange={(e) => setAgentKey(e.target.value)}
                placeholder="Opcional"
              />
            </div>

            <div className="settings-field">
              <label>PASTA DE IMPRESSÃO</label>
              <input
                value={hotFolderPath}
                onChange={(e) => setHotFolderPath(e.target.value)}
                placeholder="Caminho"
              />
            </div>
          </div>

          {error && <div className="settings-error">{error}</div>}
        </section>

        <footer className="actions">
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
