import { useEffect, useMemo, useState } from "react";
import {
  ApplyUpdateAndRestart,
  CheckUpdate,
  GetStatus,
  GetPrinterConfig,
  MinimizeToTray,
  Reconnect,
} from "../wailsjs/go/main/App";
import { EventsOn } from "../wailsjs/runtime";
import {
  IconPhoto,
  IconFileText,
  IconPrinter,
  IconCheck,
  IconX,
  IconDownload,
  IconRefresh,
  IconClock,
  IconSettings,
  IconChevronUp,
} from "@tabler/icons-react";

type ConnectionStatus = "connected" | "disconnected";
type FileStatus = "pending" | "downloading" | "moving" | "printed" | "failed";

interface JobFileStatus {
  name: string;
  type: string;
  status: FileStatus;
  error?: string;
}

interface JobEvent {
  kind: string;
  jobId: string;
  customerName: string;
  status: string;
  message: string;
  files: JobFileStatus[];
  timestamp: string;
}

interface JobHistoryItem {
  jobId: string;
  customerName: string;
  timestamp: string;
  status: string;
}

interface UpdateInfo {
  version: string;
  downloadUrl: string;
  releaseNotes: string;
}

const isRecord = (value: unknown): value is Record<string, unknown> => {
  return typeof value === "object" && value !== null;
};

const parseUpdateInfo = (value: unknown): UpdateInfo | null => {
  if (!isRecord(value)) return null;
  const version = typeof value.version === "string" ? value.version : "";
  const downloadUrl =
    typeof value.downloadUrl === "string" ? value.downloadUrl : "";
  if (!version || !downloadUrl) return null;
  return {
    version,
    downloadUrl,
    releaseNotes:
      typeof value.releaseNotes === "string" ? value.releaseNotes : "",
  };
};

const parseJobEvent = (value: unknown): JobEvent | null => {
  if (!isRecord(value)) return null;
  const filesValue = value.files;
  if (!Array.isArray(filesValue)) return null;

  const files: JobFileStatus[] = filesValue
    .filter(isRecord)
    .map((file) => ({
      name: typeof file.name === "string" ? file.name : "",
      type: typeof file.type === "string" ? file.type : "polaroid",
      status: parseFileStatus(file.status),
      error: typeof file.error === "string" ? file.error : undefined,
    }))
    .filter((file) => file.name);

  return {
    kind: typeof value.kind === "string" ? value.kind : "",
    jobId: typeof value.jobId === "string" ? value.jobId : "",
    customerName:
      typeof value.customerName === "string" ? value.customerName : "",
    status: typeof value.status === "string" ? value.status : "",
    message: typeof value.message === "string" ? value.message : "",
    files,
    timestamp: typeof value.timestamp === "string" ? value.timestamp : "",
  };
};

const parseFileStatus = (value: unknown): FileStatus => {
  if (
    value === "pending" ||
    value === "downloading" ||
    value === "moving" ||
    value === "printed" ||
    value === "failed"
  ) {
    return value;
  }
  return "pending";
};

function FileStatusIcon({ status }: { status: string }) {
  switch (status) {
    case "printed":
      return <IconCheck size={13} className="file-state-icon ok" aria-hidden />;
    case "failed":
      return <IconX size={13} className="file-state-icon err" aria-hidden />;
    case "downloading":
      return (
        <IconDownload size={13} className="file-state-icon dl" aria-hidden />
      );
    case "moving":
    case "generating_pdf":
    case "sending_to_printer":
      return (
        <IconRefresh
          size={13}
          className="file-state-icon processing"
          aria-hidden
        />
      );
    default:
      return (
        <IconClock size={13} className="file-state-icon dim" aria-hidden />
      );
  }
}

function FileTypeBadge({ type }: { type: string }) {
  const label = type === "carta" ? "CARTA" : type === "foto" ? "FOTO" : "OUTRO";
  return <span className={`file-type-badge ${type}`}>{label}</span>;
}

function JobStatusBadge({ status }: { status: string }) {
  const map: Record<string, { label: string; cls: string }> = {
    printed: { label: "IMPRESSO", cls: "printed" },
    failed: { label: "FALHOU", cls: "failed" },
    started: { label: "IMPRIMINDO", cls: "printing" },
    received: { label: "RECEBIDO", cls: "printing" },
  };
  const { label, cls } = map[status] ?? { label: "AGUARDANDO", cls: "waiting" };
  return <span className={`job-status-badge ${cls}`}>{label}</span>;
}

interface AgentPanelProps {
  onReconfigure?: () => void;
}

export function AgentPanel({ onReconfigure }: AgentPanelProps) {
  const [status, setStatus] = useState<ConnectionStatus>("disconnected");
  const [currentJob, setCurrentJob] = useState<JobEvent | null>(null);
  const [history, setHistory] = useState<JobHistoryItem[]>([]);
  const [historyModalOpen, setHistoryModalOpen] = useState(false);
  const [printerConfig, setPrinterConfig] = useState<{
    photo: string | null;
    letter: string | null;
  }>({ photo: null, letter: null });
  const [updateInfo, setUpdateInfo] = useState<UpdateInfo | null>(null);
  const [updating, setUpdating] = useState(false);

  useEffect(() => {
    GetStatus().then((value: string) => {
      setStatus(value === "connected" ? "connected" : "disconnected");
    });

    GetPrinterConfig().then(
      (cfg: { photo?: string | null; letter?: string | null }) => {
        setPrinterConfig({
          photo: cfg.photo ?? null,
          letter: cfg.letter ?? null,
        });
      },
    );

    CheckUpdate()
      .then((info: unknown) => {
        setUpdateInfo(parseUpdateInfo(info));
      })
      .catch(() => {});

    const offStatus = EventsOn("ws:status", (value: unknown) => {
      setStatus(value === "connected" ? "connected" : "disconnected");
    });

    const offJob = EventsOn("ws:job", (value: unknown) => {
      const event = parseJobEvent(value);
      if (!event) return;

      setCurrentJob(event);
      if (event.status === "printed" || event.status === "failed") {
        setHistory((prev) => {
          const next = [
            {
              jobId: event.jobId,
              customerName: event.customerName,
              timestamp: event.timestamp,
              status: event.status,
            },
            ...prev.filter((item) => item.jobId !== event.jobId),
          ];
          return next.slice(0, 5);
        });
      }
    });

    const offPrinterConfig = EventsOn(
      "ws:printerConfig",
      (cfg: { photo?: string | null; letter?: string | null }) => {
        setPrinterConfig({
          photo: cfg.photo ?? null,
          letter: cfg.letter ?? null,
        });
      },
    );

    const offUpdate = EventsOn("app:update", (value: unknown) => {
      const info = parseUpdateInfo(value);
      if (info) setUpdateInfo(info);
    });

    return () => {
      offStatus();
      offJob();
      offPrinterConfig();
      offUpdate();
    };
  }, []);

  const filesCount = currentJob?.files.length ?? 0;

  const historyRows = useMemo(() => {
    return history.map((item) => {
      const printed = item.status === "printed";
      return {
        ...item,
        label: `${item.customerName || "Cliente"} — ${item.timestamp} — ${
          printed ? "✅ Impresso" : "❌ Falhou"
        }`,
      };
    });
  }, [history]);

  const handleReconnect = () => {
    Reconnect().catch((error: unknown) => {
      console.error(error);
    });
  };

  const handleClose = () => {
    MinimizeToTray();
  };

  const handleUpdate = async () => {
    if (!updateInfo) return;
    setUpdating(true);
    try {
      await ApplyUpdateAndRestart(updateInfo.downloadUrl);
    } catch (error) {
      console.error(error);
      setUpdating(false);
    }
  };

  return (
    <main className="shell">
      <header className="topbar">
        <div className="status">
          <div
            className={`status-dot ${status === "connected" ? "connected" : "disconnected"}`}
          />
          <span>{status === "connected" ? "conectado" : "desconectado"}</span>
        </div>
        <div style={{ display: "flex", alignItems: "center", gap: 8 }}>
          <span className="brand">CESTO D'AMORE</span>
          {onReconfigure && (
            <button
              type="button"
              onClick={onReconfigure}
              style={{
                background: "none",
                border: 0,
                cursor: "pointer",
                color: "#3a3a3a",
                padding: 0,
                display: "flex",
                alignItems: "center",
                height: 20,
                width: 20,
              }}
              title="Reconfigurar"
            >
              <IconSettings size={14} />
            </button>
          )}
        </div>
      </header>

      <section className="section">
        <div className="section-label">IMPRESSORAS</div>
        <div className="printer-row">
          <div className="printer-icon photo" aria-hidden="true">
            <IconPhoto size={14} />
          </div>
          <div style={{ flex: 1 }}>
            <div className="printer-role">FOTOS &amp; QUADROS</div>
            <div
              className={`printer-name ${printerConfig.photo ? "" : "unconfigured"}`}
            >
              {printerConfig.photo ?? "não configurada"}
            </div>
          </div>
          <div
            className={`printer-status ${printerConfig.photo ? "ok" : "warn"}`}
          />
        </div>
        <div className="printer-row">
          <div className="printer-icon letter" aria-hidden="true">
            <IconFileText size={14} />
          </div>
          <div style={{ flex: 1 }}>
            <div className="printer-role">CARTINHAS</div>
            <div
              className={`printer-name ${printerConfig.letter ? "" : "unconfigured"}`}
            >
              {printerConfig.letter ?? "não configurada"}
            </div>
          </div>
          <div
            className={`printer-status ${printerConfig.letter ? "ok" : "warn"}`}
          />
        </div>
      </section>

      <section className="section job-current">
        <div className="job-header">
          <div className="section-label" style={{ marginBottom: 0 }}>
            JOB ATUAL
          </div>
          <JobStatusBadge status={currentJob?.status ?? ""} />
        </div>
        {currentJob ? (
          <>
            <div className="customer-name">
              {currentJob.customerName || "Sem nome"}
            </div>
            <div className="file-count">{filesCount} arquivos</div>
            {currentJob.message && currentJob.status === "failed" ? (
              <div className="job-error">{currentJob.message}</div>
            ) : null}
            <div className="file-list">
              {currentJob.files.map((file, index) => (
                <div
                  className={`file-row${file.status === "failed" ? " file-failed" : ""}`}
                  key={`${file.type}-${file.name}-${index}`}
                >
                  <FileTypeBadge type={file.type} />
                  <span className="file-name">{file.name}</span>
                  <FileStatusIcon status={file.status} />
                  {file.error ? (
                    <span className="file-error">{file.error}</span>
                  ) : null}
                </div>
              ))}
            </div>
          </>
        ) : (
          <div className="empty">
            <IconPrinter size={22} aria-hidden="true" />
            nenhum job em fila
          </div>
        )}
      </section>

      <section className="section history">
        <div className="history-header">
          <div className="section-label" style={{ marginBottom: 0 }}>
            HISTÓRICO
          </div>
          {history.length > 1 && (
            <button
              type="button"
              className="history-btn"
              onClick={() => setHistoryModalOpen(true)}
              title="Ver histórico completo"
            >
              <IconChevronUp size={14} />
            </button>
          )}
        </div>
        {history.length > 0 ? (
          <div className="history-row" key={history[0].jobId}>
            <div
              className={`history-dot ${history[0].status === "failed" ? "fail" : "ok"}`}
            />
            <span className="history-name">{history[0].customerName}</span>
            <span className="history-time">{history[0].timestamp}</span>
          </div>
        ) : (
          <div className="empty">sem registros</div>
        )}

        {historyModalOpen && (
          <div className="history-overlay" onClick={() => setHistoryModalOpen(false)}>
            <div className="history-modal" onClick={(e) => e.stopPropagation()}>
              <div className="history-modal-header">
                <span className="section-label">HISTÓRICO COMPLETO</span>
                <button
                  type="button"
                  className="history-modal-close"
                  onClick={() => setHistoryModalOpen(false)}
                >
                  <IconX size={16} />
                </button>
              </div>
              {historyRows.map((item) => (
                <div className="history-row" key={item.jobId}>
                  <div
                    className={`history-dot ${item.status === "failed" ? "fail" : "ok"}`}
                  />
                  <span className="history-name">{item.customerName}</span>
                  <span className="history-time">{item.timestamp}</span>
                </div>
              ))}
            </div>
          </div>
        )}
      </section>

      {updateInfo && (
        <div className="update-banner">
          <div className="update-info">
            <span className="update-version">
              {updateInfo.version} disponível
            </span>
            {updateInfo.releaseNotes && (
              <span className="update-notes">{updateInfo.releaseNotes}</span>
            )}
          </div>
          <button
            type="button"
            className="btn-update"
            onClick={handleUpdate}
            disabled={updating}
          >
            {updating ? "ATUALIZANDO..." : "ATUALIZAR"}
          </button>
        </div>
      )}

      <footer className="actions">
        <button type="button" className="btn-ghost" onClick={handleReconnect}>
          RECONECTAR
        </button>
        <button type="button" className="btn-primary" onClick={handleClose}>
          FECHAR
        </button>
      </footer>
    </main>
  );
}
