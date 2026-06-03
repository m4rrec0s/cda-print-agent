import {
  IconArrowLeft,
  IconPrinter,
  IconBrandGithub,
  IconWorld,
  IconExternalLink,
} from "@tabler/icons-react";

interface Props {
  onBack: () => void;
  version: string;
}

export function AboutPage({ onBack, version }: Props) {
  return (
    <div className="settings-page">
      <header className="settings-page-header">
        <button
          type="button"
          className="back-btn"
          onClick={onBack}
          title="Voltar"
        >
          <IconArrowLeft size={16} />
        </button>
        <span className="settings-page-title">SOBRE</span>
      </header>

      <div className="settings-page-scroll">
        <div className="about-card">
          <div className="about-row">
            <div className="about-icon">
              <IconPrinter size={16} />
            </div>
            <div className="about-info">
              <div className="about-label">Cesto d'Amore</div>
              <div className="about-value">Agente de Impressão v{version}</div>
            </div>
          </div>

          <div className="about-row">
            <div className="about-icon">
              <IconBrandGithub size={16} />
            </div>
            <div className="about-info">
              <div className="about-label">Criador</div>
              <div className="about-value">m4rrec0s</div>
            </div>
            <a
              href="https://github.com/m4rrec0s"
              target="_blank"
              rel="noopener noreferrer"
              className="about-link"
              style={{ display: "inline-flex", alignItems: "center", gap: 4 }}
            >
              GitHub <IconExternalLink size={12} />
            </a>
          </div>

          <div className="about-row">
            <div className="about-icon">
              <IconWorld size={16} />
            </div>
            <div className="about-info">
              <div className="about-label">Painel de Moderação</div>
              <div className="about-value">manager.cestodamore.com.br</div>
            </div>
            <a
              href="https://manager.cestodamore.com.br/"
              target="_blank"
              rel="noopener noreferrer"
              className="about-link"
              style={{ display: "inline-flex", alignItems: "center", gap: 4 }}
            >
              Abrir <IconExternalLink size={12} />
            </a>
          </div>
        </div>
      </div>
    </div>
  );
}
