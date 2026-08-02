import { useEffect, useState } from "react";
import { EventsOn } from "../wailsjs/runtime";
import { Disconnect, GetDashboardSnapshot, OpenHotFolder, Reconnect } from "../wailsjs/go/main/App";
import {
  IconBell, IconBox, IconClock, IconFileText, IconFolder, IconHistory,
  IconHome, IconMenu2, IconPhoto, IconPrinter, IconRefresh, IconSettings,
  IconAlertTriangle, IconCheck, IconX,
} from "@tabler/icons-react";

type Screen = "dashboard" | "queue" | "printers" | "history" | "more";
type Job = { id: string; customer: string; status: string; createdAt: string; updatedAt: string; printerRole: string; files: { name: string; type: string }[]; lastError?: string };
type Snapshot = { status: string; photo: string; letter: string; today: number; printed: number; queued: number; failed: number; jobs: Job[] };

const empty: Snapshot = { status: "connecting", photo: "", letter: "", today: 0, printed: 0, queued: 0, failed: 0, jobs: [] };
const time = (value: string) => value ? new Date(value).toLocaleTimeString("pt-BR", { hour: "2-digit", minute: "2-digit" }) : "--:--";
const title = (job: Job) => `#${job.id.slice(0, 6).toUpperCase()}`;
const status = (job: Job) => job.status === "PRINTED" ? "Impresso" : job.status === "FAILED" ? "Falhou" : job.status === "PRINTING" ? "Imprimindo" : "Na fila";

export function Dashboard({ onConfigure }: { onConfigure: () => void }) {
  const [screen, setScreen] = useState<Screen>("dashboard");
  const [data, setData] = useState<Snapshot>(empty);
  const load = () => GetDashboardSnapshot().then((value: Snapshot) => setData(value)).catch(() => setData((prev) => ({ ...prev, status: "disconnected" })));
  useEffect(() => {
    load();
    const timer = window.setInterval(load, 15000);
    const offStatus = EventsOn("ws:status", load);
    const offJob = EventsOn("ws:job", load);
    return () => { window.clearInterval(timer); offStatus(); offJob(); };
  }, []);

  const connected = data.status === "connected";
  const inactive = data.status === "inactive";
  const online = connected || inactive;
  const [busy, setBusy] = useState(false);
  const toggleConnection = () => {
    setBusy(true);
    (online ? Disconnect() : Reconnect()).then(() => setTimeout(load, 300)).catch(() => {}).finally(() => setBusy(false));
  };
  const jobs = screen === "history" ? data.jobs.filter((job) => job.status === "PRINTED" || job.status === "FAILED") : data.jobs;
  return <main className="dashboard-shell">
    <aside className="dashboard-sidebar">
      <div className="dashboard-logo"><IconPrinter /><div><b>PrintFlow</b><small>Impressão Automática</small></div></div>
      <Nav active={screen} set={setScreen} icon={<IconHome />} label="Dashboard" value="dashboard" />
      <Nav active={screen} set={setScreen} icon={<IconBox />} label="Fila de impressão" value="queue" />
      <Nav active={screen} set={setScreen} icon={<IconPrinter />} label="Impressoras" value="printers" />
      <Nav active={screen} set={setScreen} icon={<IconHistory />} label="Histórico" value="history" />
      <button className="nav-item" onClick={onConfigure}><IconSettings /> Configurações</button>
      <div className={`agent-state ${online ? (inactive ? "inactive" : "online") : "offline"}`}><i /> Agente {inactive ? "inativo" : online ? "online" : "offline"}</div>
    </aside>
    <section className="dashboard-main">
      <header className="dashboard-header"><div><h1>{screen === "dashboard" ? "Olá, Administrador! 👋" : screen === "queue" ? "Fila de Impressão" : screen === "printers" ? "Impressoras" : screen === "history" ? "Histórico" : "Mais opções"}</h1><p>{screen === "dashboard" ? "Resumo da operação de impressão." : "Dados locais atualizados automaticamente."}</p></div><div className="header-tools"><div className="header-left"><span className={`online-badge ${inactive ? "inactive" : online ? "online" : ""}`}><i />{inactive ? "Inativo" : online ? "Online" : "Offline"}</span><button className="connect-btn" onClick={toggleConnection} disabled={busy} title={online ? "Desconectar agente" : "Conectar agente"}>{online ? <IconX /> : <IconCheck />}{online ? "Desconectar" : "Conectar"}</button></div><div className="header-right"><button onClick={load} title="Atualizar"><IconRefresh /></button><button onClick={onConfigure} title="Configurações"><IconSettings /></button></div></div></header>
      {screen === "dashboard" && <>
        <section className="metric-grid"><Metric icon={<IconBox />} label="Pedidos hoje" value={data.today} color="purple" /><Metric icon={<IconPrinter />} label="Impressos hoje" value={data.printed} color="green" /><Metric icon={<IconClock />} label="Na fila" value={data.queued} color="blue" /><Metric icon={<IconAlertTriangle />} label="Falhas hoje" value={data.failed} color="amber" /></section>
        <div className="dashboard-grid"><Queue jobs={data.jobs.slice(0, 6)} onAll={() => setScreen("queue")} /><Printers data={data} onAll={() => setScreen("printers")} /><Recent jobs={data.jobs.slice(0, 4)} onAll={() => setScreen("history")} /><System data={data} /></div>
      </>}
      {screen === "queue" && <Queue jobs={jobs} onAll={() => {}} expanded />}
      {screen === "printers" && <Printers data={data} onAll={() => {}} expanded />}
      {screen === "history" && <Recent jobs={jobs} onAll={() => {}} expanded />}
      {screen === "more" && <div className="empty-dashboard"><IconFolder size={34}/><b>Artes salvas</b><button onClick={() => OpenHotFolder()}>Abrir pasta local</button></div>}
    </section>
    <nav className="mobile-nav"><Nav active={screen} set={setScreen} icon={<IconHome />} label="Dashboard" value="dashboard" /><Nav active={screen} set={setScreen} icon={<IconBox />} label="Fila" value="queue" /><Nav active={screen} set={setScreen} icon={<IconPrinter />} label="Impressoras" value="printers" /><Nav active={screen} set={setScreen} icon={<IconFileText />} label="Pedidos" value="history" /><Nav active={screen} set={setScreen} icon={<IconMenu2 />} label="Mais" value="more" /></nav>
  </main>;
}

function Nav({ active, set, icon, label, value }: { active: Screen; set: (value: Screen) => void; icon: JSX.Element; label: string; value: Screen }) { return <button className={`nav-item ${active === value ? "active" : ""}`} onClick={() => set(value)}>{icon}<span>{label}</span></button>; }
function Metric({ icon, label, value, color }: { icon: JSX.Element; label: string; value: number; color: string }) { return <article className="metric-card"><span className={`metric-icon ${color}`}>{icon}</span><div><small>{label}</small><b>{value}</b><em>{label === "Na fila" ? "Aguardando impressão" : label === "Falhas hoje" ? "Verifique os logs" : "Dados locais"}</em></div></article>; }
function Queue({ jobs, onAll, expanded = false }: { jobs: Job[]; onAll: () => void; expanded?: boolean }) { return <article className={`dashboard-card queue-card ${expanded ? "expanded" : ""}`}><header><h2>Fila de Impressão <span>{jobs.filter((job) => job.status !== "PRINTED" && job.status !== "FAILED").length}</span></h2>{!expanded && <button onClick={onAll}>Ver fila completa</button>}</header>{jobs.length ? <div className="queue-table">{jobs.map((job) => <div className="queue-row" key={job.id}><div><b>{title(job)}</b><small>{time(job.createdAt)}</small></div><div className="job-files">{job.files.slice(0, 3).map((file, index) => <span key={`${file.name}-${index}`} className={file.type === "carta" ? "text-file" : "image-file"}>{file.type === "carta" ? <IconFileText /> : <IconPhoto />}</span>)}{job.files.length > 3 && <small>+{job.files.length - 3}</small>}</div><div className="printer-cell"><IconPrinter /><span><b>{job.printerRole === "letter" ? "Texto" : "Fotos"}</b><small>{job.printerRole === "letter" ? "Cartinhas e resumo" : "Artes"}</small></span></div><div className={`job-state ${job.status.toLowerCase()}`}><i />{status(job)}</div></div>)}</div> : <p className="queue-empty">Nenhum job registrado ainda.</p>}<footer><i />Atualização automática ativa</footer></article>; }
function Printers({ data, onAll, expanded = false }: { data: Snapshot; onAll: () => void; expanded?: boolean }) { const printers = [{ name: data.photo || "Fotos não configurada", role: "Fotos", icon: <IconPhoto />, configured: !!data.photo }, { name: data.letter || "Texto não configurada", role: "Texto", icon: <IconPrinter />, configured: !!data.letter }]; return <article className={`dashboard-card printers-card ${expanded ? "expanded" : ""}`}><header><h2>Impressoras</h2>{!expanded && <button onClick={onAll}>Ver todas</button>}</header>{printers.map((printer) => <div className="printer-dashboard" key={printer.role}><span className={printer.role === "Fotos" ? "photo" : "text"}>{printer.icon}</span><div><b>{printer.name}</b><small>Impressora de {printer.role}</small></div><em className={printer.configured ? "ready" : "missing"}>{printer.configured ? "Pronta" : "Configurar"}</em></div>)}</article>; }
function Recent({ jobs, onAll, expanded = false }: { jobs: Job[]; onAll: () => void; expanded?: boolean }) { return <article className={`dashboard-card recent-card ${expanded ? "expanded" : ""}`}><header><h2>Atividade recente</h2>{!expanded && <button onClick={onAll}>Ver histórico</button>}</header>{jobs.length ? jobs.map((job) => <div className="recent-row" key={job.id}><span className={job.status === "FAILED" ? "fail" : "ok"}>{job.status === "FAILED" ? <IconX /> : <IconCheck />}</span><div><b>{job.status === "FAILED" ? "Falha ao imprimir" : "Pedido processado"} {title(job)}</b><small>{job.customer || "Cliente"}{job.lastError ? ` • ${job.lastError}` : ""}</small></div><time>{time(job.updatedAt)}</time></div>) : <p className="queue-empty">Atividades surgirão após primeiro job.</p>}</article>; }
function System({ data }: { data: Snapshot }) { const connected = data.status === "connected"; const inactive = data.status === "inactive"; const label = inactive ? "Inativo" : connected ? "Online" : "Offline"; return <article className="dashboard-card system-card"><header><h2>Status do Sistema</h2></header><div><span>Agente</span><b className={inactive ? "inactive" : connected ? "ready" : "missing"}>{label}</b></div><div><span>Fila local persistida</span><b className="ready">Ativa</b></div><div><span>Impressora fotos</span><b className={data.photo ? "ready" : "missing"}>{data.photo ? "Configurada" : "Pendente"}</b></div><div><span>Impressora texto</span><b className={data.letter ? "ready" : "missing"}>{data.letter ? "Configurada" : "Pendente"}</b></div></article>; }
