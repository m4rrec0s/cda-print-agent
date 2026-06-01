package main

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type App struct {
	ctx          context.Context
	tray         *TrayManager
	allowQuit    bool
	updateCancel context.CancelFunc
	mu           sync.RWMutex
}

func NewApp() *App {
	return &App{}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	cfg, err := LoadConfigFromFile()
	if err != nil {
		log.Printf("event=config_not_found reason=first_run")
		return
	}

	log.Printf("event=startup ws_url=%s api_url=%s hot_folder_configured=%t",
		cfg.WSURL, cfg.APIURL, cfg.HotFolderPath != "")

	wsManager = NewWebSocketManager(cfg.WSURL, cfg.APIURL, cfg.AgentKey, cfg.HotFolderPath)
	a.tray = NewTrayManager(wsManager.StatusUpdates())
	a.tray.Start(a)

	if err := wsManager.Connect(); err != nil {
		log.Printf("event=websocket_initial_connection_failed error=%q", err.Error())
	}

	wsManager.StartListening(ctx)
	a.startUpdateTicker(ctx, cfg.APIURL, cfg.AgentKey)
}

// ── Config bindings ──────────────────────────────────

func (a *App) IsConfigured() bool {
	return ConfigExists()
}

func (a *App) GetAgentConfig() AgentConfig {
	cfg, err := LoadConfigFromFile()
	if err != nil {
		return AgentConfig{}
	}
	return *cfg
}

func (a *App) SaveAgentConfig(wsURL string, apiURL string, agentKey string, hotFolderPath string) error {
	cfg := AgentConfig{
		WSURL:         wsURL,
		APIURL:        apiURL,
		AgentKey:      agentKey,
		HotFolderPath: hotFolderPath,
	}

	if err := SaveConfigFile(cfg); err != nil {
		return err
	}

	wsManager = NewWebSocketManager(cfg.WSURL, cfg.APIURL, cfg.AgentKey, cfg.HotFolderPath)
	if a.tray == nil {
		a.tray = NewTrayManager(wsManager.StatusUpdates())
		a.tray.Start(a)
	}
	if err := wsManager.Connect(); err != nil {
		log.Printf("event=websocket_connect_after_config_failed error=%q", err.Error())
		return err
	}
	wsManager.StartListening(a.ctx)
	a.startUpdateTicker(a.ctx, cfg.APIURL, cfg.AgentKey)
	return nil
}

// ── Existing bindings (preservados) ──────────────────

func (a *App) GetStatus() string {
	if wsManager != nil && wsManager.IsConnected() {
		return "connected"
	}
	return "disconnected"
}

func (a *App) GetPrintersList() []string {
	printers, err := GetPrinters()
	if err != nil {
		log.Printf("event=get_printers_failed error=%q", err.Error())
		return []string{}
	}
	return printers
}

func (a *App) GetPrinterConfig() map[string]*string {
	if wsManager == nil {
		return map[string]*string{"photo": nil, "letter": nil}
	}
	cfg := wsManager.GetPrinterConfig()
	return map[string]*string{
		"photo":  cfg.Photo,
		"letter": cfg.Letter,
	}
}

func (a *App) SetSelectedPrinter(role string, printerName string) {
	if wsManager == nil {
		return
	}
	wsManager.mu.Lock()
	cfg := wsManager.printerConfig
	if role == "photo" {
		cfg.Photo = &printerName
	} else if role == "letter" {
		cfg.Letter = &printerName
	}
	wsManager.printerConfig = cfg
	wsManager.mu.Unlock()
	log.Printf("event=printer_selected role=%q printer=%q", role, printerName)

	// Send authorization to backend
	msg := WSMessage{
		Type:            "AUTHORIZE_PRINTER",
		SelectedPrinter: printerName,
		Timestamp:       time.Now().Format(time.RFC3339),
	}
	if err := wsManager.SendMessage(msg); err != nil {
		log.Printf("event=printer_auth_send_failed error=%q", err.Error())
	}
}

func (a *App) GetSelectedPrinter() string {
	return ""
}

func (a *App) Reconnect() error {
	if wsManager == nil {
		return nil
	}

	wsManager.Close()
	if err := wsManager.Connect(); err != nil {
		log.Printf("event=websocket_reconnect_failed error=%q", err.Error())
		return err
	}
	return nil
}

func (a *App) CheckUpdate() (*VersionInfo, error) {
	cfg, err := LoadConfigFromFile()
	if err != nil {
		return nil, nil
	}
	return CheckUpdate(cfg.APIURL, cfg.AgentKey, Version)
}

func (a *App) ApplyUpdateAndRestart(downloadURL string) error {
	log.Printf("event=update_started download_url=%s", downloadURL)

	if err := ApplyUpdate(downloadURL); err != nil {
		log.Printf("event=update_failed error=%q", err.Error())
		return err
	}

	if err := RestartApp(); err != nil {
		log.Printf("event=restart_failed error=%q", err.Error())
		return err
	}

	a.QuitApp()
	return nil
}

func (a *App) startUpdateTicker(ctx context.Context, apiURL string, agentKey string) {
	if ctx == nil || apiURL == "" {
		return
	}

	a.mu.Lock()
	if a.updateCancel != nil {
		a.updateCancel()
	}
	tickerCtx, cancel := context.WithCancel(ctx)
	a.updateCancel = cancel
	a.mu.Unlock()

	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()

		for {
			select {
			case <-tickerCtx.Done():
				return
			case <-ticker.C:
				info, err := CheckUpdate(apiURL, agentKey, Version)
				if err != nil {
					log.Printf("event=update_check_failed error=%q", err.Error())
					continue
				}
				if info != nil {
					runtime.EventsEmit(a.ctx, "app:update", info)
				}
			}
		}
	}()
}

func (a *App) MinimizeToTray() {
	if a.ctx == nil {
		return
	}
	runtime.WindowHide(a.ctx)
}

func (a *App) ShowWindow() {
	if a.ctx == nil {
		return
	}
	runtime.WindowShow(a.ctx)
	runtime.WindowUnminimise(a.ctx)
	runtime.WindowSetAlwaysOnTop(a.ctx, true)
	runtime.WindowSetAlwaysOnTop(a.ctx, false)
}

func (a *App) QuitApp() {
	a.mu.Lock()
	a.allowQuit = true
	a.mu.Unlock()

	if wsManager != nil {
		wsManager.Stop()
	}
	if a.updateCancel != nil {
		a.updateCancel()
	}
	if a.tray != nil {
		a.tray.Stop()
	}
	if a.ctx != nil {
		runtime.Quit(a.ctx)
	}
}

func (a *App) ShouldQuit() bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.allowQuit
}

func (a *App) shutdown(ctx context.Context) {
	log.Printf("event=shutdown")
	if wsManager != nil {
		wsManager.Stop()
	}
	if a.updateCancel != nil {
		a.updateCancel()
	}
	if a.tray != nil {
		a.tray.Stop()
	}
}
