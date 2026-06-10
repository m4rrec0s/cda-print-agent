import { useState } from "react";
import { LogEntry } from "../types/logs";

interface ErrorLogProps {
  logs: LogEntry[];
  onClear: () => void;
  defaultOpen?: boolean;
}

const LEVEL_STYLES: Record<string, { label: string; className: string }> = {
  error: { label: "ERR", className: "log-badge-error" },
  warn: { label: "AVS", className: "log-badge-warn" },
  info: { label: "INF", className: "log-badge-info" },
};

function formatTime(iso: string): string {
  const d = new Date(iso);
  return d.toLocaleTimeString("pt-BR", { hour: "2-digit", minute: "2-digit", second: "2-digit" });
}

export function ErrorLog({ logs, onClear, defaultOpen = false }: ErrorLogProps) {
  const [open, setOpen] = useState(defaultOpen);
  const errorCount = logs.filter((l) => l.level === "error").length;

  return (
    <section className="section error-log-section">
      <button
        type="button"
        className="error-log-toggle"
        onClick={() => setOpen((v) => !v)}
      >
        <span className="section-label" style={{ marginBottom: 0 }}>
          LOGS
          {errorCount > 0 && (
            <span className="error-log-count">{errorCount}</span>
          )}
        </span>
        <span className={`collapse-icon ${open ? "expanded" : ""}`}>▼</span>
      </button>

      {open && (
        <div className="error-log-body">
          {logs.length > 0 && (
            <button type="button" className="error-log-clear" onClick={onClear}>
              LIMPAR
            </button>
          )}
          <div className="error-log-scroll">
            {logs.length === 0 ? (
              <div className="empty">sem logs</div>
            ) : (
              logs.map((entry) => {
                const style = LEVEL_STYLES[entry.level] ?? LEVEL_STYLES.info;
                return (
                  <div className="error-log-row" key={entry.id}>
                    <span className="error-log-time">{formatTime(entry.timestamp)}</span>
                    <span className={`error-log-badge ${style.className}`}>{style.label}</span>
                    <span className="error-log-msg">{entry.message}</span>
                  </div>
                );
              })
            )}
          </div>
        </div>
      )}
    </section>
  );
}
