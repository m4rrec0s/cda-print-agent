import {
  IconArrowLeft,
  IconSun,
  IconMoon,
  IconDeviceDesktop,
} from "@tabler/icons-react";
import { useTheme } from "./ThemeContext";

interface Props {
  onBack: () => void;
}

export function ThemePage({ onBack }: Props) {
  const { theme, setTheme } = useTheme();

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
        <span className="settings-page-title">TEMA</span>
      </header>

      <div className="settings-page-scroll">
        <div className="theme-options">
          <button
            type="button"
            className={`theme-option ${theme === "dark" ? "selected" : ""}`}
            onClick={() => setTheme("dark")}
          >
            <div className="theme-option-icon dark">
              <IconMoon size={18} />
            </div>
            <span className="theme-option-label">Escuro</span>
          </button>
          <button
            type="button"
            className={`theme-option ${theme === "light" ? "selected" : ""}`}
            onClick={() => setTheme("light")}
          >
            <div className="theme-option-icon light">
              <IconSun size={18} />
            </div>
            <span className="theme-option-label">Claro</span>
          </button>
          <button
            type="button"
            className={`theme-option ${theme === "system" ? "selected" : ""}`}
            onClick={() => setTheme("system")}
          >
            <div className="theme-option-icon system">
              <IconDeviceDesktop size={18} />
            </div>
            <span className="theme-option-label">Sistema</span>
          </button>
        </div>
      </div>
    </div>
  );
}
