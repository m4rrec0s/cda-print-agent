import { useEffect, useMemo, useState } from "react";
import { useAgentLogs } from "./hooks/useAgentLogs";
import { ErrorLog } from "./components/ErrorLog";
import { useToast, ToastContainer } from "./components/Toast";
import {
  ApplyUpdateAndRestart,
  ClearHotFolder,
  CheckUpdate,
  GetStatus,
  GetPrinterConfig,
  ListSavedArts,
  MinimizeToTray,
  OpenHotFolder,
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
  IconFolder,
  IconTrash,
  IconFiles,
  IconChevronDown,
} from "@tabler/icons-react";

type ConnectionStatus = "connected" | "connecting" | "disconnected";
type FileStatus = "pending" | "downloading" | "moving" | "printed" | "failed";
type ActiveTab = "panel" | "arts";

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

interface SavedArtInfo {
  name: string;
  path: string;
  sizeBytes: number;
  modifiedAt: string;
  isDir: boolean;
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

const parseSavedArts = (value: unknown): SavedArtInfo[] => {
  if (!Array.isArray(value)) return [];
  return value
    .filter(isRecord)
    .map((item) => ({
      name: typeof item.name === "string" ? item.name : "",
      path: typeof item.path === "string" ? item.path : "",
      sizeBytes: typeof item.sizeBytes === "number" ? item.sizeBytes : 0,
      modifiedAt: typeof item.modifiedAt === "string" ? item.modifiedAt : "",
      isDir: Boolean(item.isDir),
    }))
    .filter((item) => item.name);
};

const formatBytes = (bytes: number) => {
  if (bytes <= 0) return "0 KB";
  const units = ["B", "KB", "MB", "GB"];
  const index = Math.min(
    Math.floor(Math.log(bytes) / Math.log(1024)),
    units.length - 1,
  );
  return `${(bytes / 1024 ** index).toFixed(index === 0 ? 0 : 1)} ${units[index]}`;
};

const formatDateTime = (value: string) => {
  if (!value) return "";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return "";
  return date.toLocaleString("pt-BR", {
    day: "2-digit",
    month: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
  });
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
  const [status, setStatus] = useState<ConnectionStatus>("connecting");
  const [activeTab, setActiveTab] = useState<ActiveTab>("panel");
  const [printersExpanded, setPrintersExpanded] = useState(false);
  const [currentJob, setCurrentJob] = useState<JobEvent | null>(null);
  const [history, setHistory] = useState<JobHistoryItem[]>([]);
  const [historyModalOpen, setHistoryModalOpen] = useState(false);
  const [printerConfig, setPrinterConfig] = useState<{
    photo: string | null;
    letter: string | null;
  }>({ photo: null, letter: null });
  const [printerConfigLoading, setPrinterConfigLoading] = useState(true);
  const [savedArts, setSavedArts] = useState<SavedArtInfo[]>([]);
  const [artsLoading, setArtsLoading] = useState(true);
  const [updateInfo, setUpdateInfo] = useState<UpdateInfo | null>(null);
  const [updateBannerDismissed, setUpdateBannerDismissed] = useState(false);
  const [updating, setUpdating] = useState(false);
  const { logs, addLog, clearLogs } = useAgentLogs();
  const { toasts, showToast } = useToast();

  const refreshSavedArts = () => {
    setArtsLoading(true);
    ListSavedArts()
      .then((items: unknown) => {
        setSavedArts(parseSavedArts(items));
      })
      .catch((error: unknown) => {
        addLog("error", "Falha ao carregar artes", error);
      })
      .finally(() => {
        setArtsLoading(false);
      });
  };

  useEffect(() => {
    GetStatus().then((value: string) => {
      setStatus(
        value === "connected"
          ? "connected"
          : value === "connecting"
            ? "connecting"
            : "disconnected",
      );
    });

    GetPrinterConfig().then(
      (cfg: { photo?: string | null; letter?: string | null }) => {
        setPrinterConfig({
          photo: cfg.photo ?? null,
          letter: cfg.letter ?? null,
        });
        if (cfg.photo || cfg.letter) setPrinterConfigLoading(false);
      },
    );

    CheckUpdate()
      .then((info: unknown) => {
        const parsed = parseUpdateInfo(info);
        setUpdateInfo(parsed);
        if (parsed) setUpdateBannerDismissed(false);
      })
      .catch(() => {});

    const offStatus = EventsOn("ws:status", (value: unknown) => {
      setStatus(
        value === "connected"
          ? "connected"
          : value === "connecting"
            ? "connecting"
            : "disconnected",
      );
    });

    const offJob = EventsOn("ws:job", (value: unknown) => {
      const event = parseJobEvent(value);
      if (!event) return;

      if (event.kind === "started") {
        showToast(`🖨️ Novo pedido: ${event.customerName || "Cliente"} — ${event.files.length} arquivo(s)`);
      }

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
        setPrinterConfigLoading(false);
      },
    );

    const offArtsChanged = EventsOn("arts:changed", () => {
      refreshSavedArts();
    });

    const offUpdate = EventsOn("app:update", (value: unknown) => {
      const info = parseUpdateInfo(value);
      if (info) {
        setUpdateInfo(info);
        setUpdateBannerDismissed(false);
      }
    });

    refreshSavedArts();
    const printerLoadingTimer = window.setTimeout(() => {
      setPrinterConfigLoading(false);
    }, 5000);

    return () => {
      window.clearTimeout(printerLoadingTimer);
      offStatus();
      offJob();
      offPrinterConfig();
      offArtsChanged();
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
    setStatus("connecting");
    Reconnect().catch((error: unknown) => {
      addLog("error", "Falha ao reconectar", error);
      setStatus("disconnected");
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
      addLog("error", "Falha ao atualizar", error);
      setUpdating(false);
    }
  };

  const handleOpenHotFolder = () => {
    OpenHotFolder().catch((error: unknown) => {
      addLog("error", "Falha ao abrir pasta", error);
    });
  };

  const handleClearHotFolder = () => {
    if (savedArts.length === 0) return;
    const confirmed = window.confirm("Esvaziar a pasta de artes salvas?");
    if (!confirmed) return;

    setArtsLoading(true);
    ClearHotFolder()
      .then(() => {
        setSavedArts([]);
      })
      .catch((error: unknown) => {
        addLog("error", "Falha ao esvaziar pasta", error);
      })
      .finally(() => {
        setArtsLoading(false);
      });
  };

  const statusLabel =
    status === "connected"
      ? "Conectado"
      : status === "connecting"
        ? "Conectando..."
        : "Desconectado";

  const renderPrinterName = (value: string | null) => {
    if (printerConfigLoading) return "Carregando...";
    return value ?? "Não configurada";
  };

  const printerSummary = printerConfigLoading
    ? "Carregando..."
    : printerConfig.photo && printerConfig.letter
      ? "Conectadas"
      : printerConfig.photo || printerConfig.letter
        ? "Parcial"
        : "Não configuradas";

  return (
    <main className="shell">
      <header className="topbar">
        <div className="status">
          <div className={`status-dot ${status}`} />
          <span>{statusLabel}</span>
        </div>
        <div style={{ display: "flex", alignItems: "center", gap: 8 }}>
          <span className="brand">CESTO D'AMORE</span>
          {onReconfigure && (
            <button
              type="button"
              className="icon-btn"
              onClick={onReconfigure}
              title="Configurações"
            >
              <IconSettings size={14} />
            </button>
          )}
        </div>
      </header>

      <nav className="tabs" aria-label="Seções do agente">
        <button
          type="button"
          className={`tab-btn ${activeTab === "panel" ? "active" : ""}`}
          onClick={() => setActiveTab("panel")}
        >
          PAINEL
        </button>
        <button
          type="button"
          className={`tab-btn ${activeTab === "arts" ? "active" : ""}`}
          onClick={() => setActiveTab("arts")}
        >
          ARTES
        </button>
      </nav>

      <div className="tab-content">
        {activeTab === "panel" ? (
          <>
            <section className="section">
              <button
                type="button"
                className="printer-collapse"
                onClick={() => setPrintersExpanded((value) => !value)}
              >
                <div>
                  <div className="section-label printer-collapse-label">
                    IMPRESSORAS{" "}
                    <span style={{ color: "var(--green)" }}>({printerSummary})</span>
                  </div>

                  <div
                    className={`printer-summary ${
                      printerSummary === "Conectadas"
                        ? "ok"
                        : printerSummary === "Carregando..."
                          ? "loading"
                          : "warn"
                    }`}
                  >
                    <div>
                      {printerSummary === "Conectadas" && !printersExpanded ? (
                        <span
                          className="printer-name"
                          style={{ fontWeight: 700 }}
                        >
                          {renderPrinterName(printerConfig.photo)} &amp;{" "}
                          {renderPrinterName(printerConfig.letter)}
                        </span>
                      ) : null}
                    </div>
                  </div>
                </div>
                <IconChevronDown
                  size={15}
                  className={`collapse-icon ${printersExpanded ? "expanded" : ""}`}
                  aria-hidden="true"
                />
              </button>

              {printersExpanded && (
                <div className="printer-details">
                  <div className="printer-row">
                    <div className="printer-icon photo" aria-hidden="true">
                      <IconPhoto size={14} />
                    </div>
                    <div style={{ flex: 1 }}>
                      <div className="printer-role">FOTOS &amp; QUADROS</div>
                      <div
                        className={`printer-name ${printerConfig.photo || printerConfigLoading ? "" : "unconfigured"}`}
                      >
                        {renderPrinterName(printerConfig.photo)}
                      </div>
                    </div>
                    <div
                      className={`printer-status ${printerConfigLoading ? "loading" : printerConfig.photo ? "ok" : "warn"}`}
                    />
                  </div>
                  <div className="printer-row">
                    <div className="printer-icon letter" aria-hidden="true">
                      <IconFileText size={14} />
                    </div>
                    <div style={{ flex: 1 }}>
                      <div className="printer-role">CARTINHAS</div>
                      <div
                        className={`printer-name ${printerConfig.letter || printerConfigLoading ? "" : "unconfigured"}`}
                      >
                        {renderPrinterName(printerConfig.letter)}
                      </div>
                    </div>
                    <div
                      className={`printer-status ${printerConfigLoading ? "loading" : printerConfig.letter ? "ok" : "warn"}`}
                    />
                  </div>
                </div>
              )}
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
                  <span className="history-name">
                    {history[0].customerName}
                  </span>
                  <span className="history-time">{history[0].timestamp}</span>
                </div>
              ) : (
                <div className="empty">sem registros</div>
              )}

              {historyModalOpen && (
                <div
                  className="history-overlay"
                  onClick={() => setHistoryModalOpen(false)}
                >
                  <div
                    className="history-modal"
                    onClick={(e) => e.stopPropagation()}
                  >
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
                        <span className="history-name">
                          {item.customerName}
                        </span>
                        <span className="history-time">{item.timestamp}</span>
                      </div>
                    ))}
                  </div>
                </div>
              )}
            </section>
          </>
        ) : (
          <section className="section arts-section">
            <div className="arts-toolbar">
              <div className="section-label" style={{ marginBottom: 0 }}>
                ARTES SALVAS
              </div>
              <div className="arts-actions">
                <button
                  type="button"
                  className="icon-btn"
                  onClick={refreshSavedArts}
                  title="Atualizar lista"
                >
                  <IconRefresh size={14} />
                </button>
                <button
                  type="button"
                  className="icon-btn"
                  onClick={handleOpenHotFolder}
                  title="Abrir pasta"
                >
                  <IconFolder size={14} />
                </button>
                <button
                  type="button"
                  className="icon-btn danger"
                  onClick={handleClearHotFolder}
                  disabled={savedArts.length === 0 || artsLoading}
                  title="Esvaziar pasta"
                >
                  <IconTrash size={14} />
                </button>
              </div>
            </div>

            {artsLoading ? (
              <div className="empty">
                <IconRefresh
                  size={22}
                  className="file-state-icon processing"
                  aria-hidden="true"
                />
                carregando artes
              </div>
            ) : savedArts.length > 0 ? (
              <div className="arts-list">
                {savedArts.map((art) => (
                  <div className="art-row" key={art.path}>
                    <div className="art-icon" aria-hidden="true">
                      {art.isDir ? (
                        <IconFolder size={14} />
                      ) : (
                        <IconFiles size={14} />
                      )}
                    </div>
                    <div className="art-main">
                      <span className="art-name">{art.name}</span>
                      <span className="art-meta">
                        {art.isDir ? "pasta" : formatBytes(art.sizeBytes)}
                        {formatDateTime(art.modifiedAt)
                          ? ` • ${formatDateTime(art.modifiedAt)}`
                          : ""}
                      </span>
                    </div>
                  </div>
                ))}
              </div>
            ) : (
              <div className="empty">
                <IconFiles size={22} aria-hidden="true" />
                nenhuma arte salva
              </div>
            )}
          </section>
        )}
      </div>

      {updateInfo && !updateBannerDismissed && (
        <div className="update-banner">
          <button
            type="button"
            className="update-close"
            onClick={() => setUpdateBannerDismissed(true)}
            title="Fechar aviso"
          >
            <IconX size={14} />
          </button>
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

      <ErrorLog logs={logs} onClear={clearLogs} />
      <ToastContainer toasts={toasts} />

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
