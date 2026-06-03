import {
  IconArrowLeft,
  IconSettings,
  IconPalette,
  IconInfoCircle,
  IconChevronRight,
} from "@tabler/icons-react";

interface Props {
  onBack: () => void;
  onOpenBasicConfig: () => void;
  onOpenTheme: () => void;
  onOpenAbout: () => void;
}

export function SettingsPage({ onBack, onOpenBasicConfig, onOpenTheme, onOpenAbout }: Props) {
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
        <span className="settings-page-title">CONFIGURAÇÕES</span>
      </header>

      <div className="settings-page-scroll">
        <div className="settings-menu">
          <button
            type="button"
            className="settings-menu-item"
            onClick={onOpenBasicConfig}
          >
            <div className="settings-menu-icon">
              <IconSettings size={16} />
            </div>
            <div className="settings-menu-text">
              <div className="settings-menu-label">Configurações Básicas</div>
              <div className="settings-menu-desc">Servidor, API, chave e pasta de impressão</div>
            </div>
            <IconChevronRight size={16} className="settings-menu-chevron" />
          </button>

          <button
            type="button"
            className="settings-menu-item"
            onClick={onOpenTheme}
          >
            <div className="settings-menu-icon">
              <IconPalette size={16} />
            </div>
            <div className="settings-menu-text">
              <div className="settings-menu-label">Tema</div>
              <div className="settings-menu-desc">Alternar entre tema escuro, claro ou do sistema</div>
            </div>
            <IconChevronRight size={16} className="settings-menu-chevron" />
          </button>

          <button
            type="button"
            className="settings-menu-item"
            onClick={onOpenAbout}
          >
            <div className="settings-menu-icon">
              <IconInfoCircle size={16} />
            </div>
            <div className="settings-menu-text">
              <div className="settings-menu-label">Sobre</div>
              <div className="settings-menu-desc">Informações do app e do criador</div>
            </div>
            <IconChevronRight size={16} className="settings-menu-chevron" />
          </button>
        </div>
      </div>
    </div>
  );
}
