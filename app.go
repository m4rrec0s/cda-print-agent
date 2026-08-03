package main

import (
	"context"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	stdruntime "runtime"
	"sort"
	"sync"
	"time"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

type App struct {
	ctx          context.Context
	tray         *TrayManager
	statusCh     chan string
	allowQuit    bool
	updateCancel context.CancelFunc
	mu           sync.RWMutex
}

type SavedArtInfo struct {
	Name       string `json:"name"`
	Path       string `json:"path"`
	SizeBytes  int64  `json:"sizeBytes"`
	ModifiedAt string `json:"modifiedAt"`
	IsDir      bool   `json:"isDir"`
}

type DashboardJobFile struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

type DashboardJob struct {
	ID          string             `json:"id"`
	Customer    string             `json:"customer"`
	Status      string             `json:"status"`
	CreatedAt   string             `json:"createdAt"`
	UpdatedAt   string             `json:"updatedAt"`
	PrinterRole string             `json:"printerRole"`
	Files       []DashboardJobFile `json:"files"`
	LastError   string             `json:"lastError,omitempty"`
}

type DashboardSnapshot struct {
	Status  string         `json:"status"`
	Photo   string         `json:"photo"`
	Letter  string         `json:"letter"`
	Today   int            `json:"today"`
	Printed int            `json:"printed"`
	Queued  int            `json:"queued"`
	Failed  int            `json:"failed"`
	Jobs    []DashboardJob `json:"jobs"`
}

func NewApp() *App {
	return &App{}
}

// startTray inicializa o systray numa goroutine própria, ANTES do wails.Run.
// O Wails v2 e o fyne.io/systray disputam a thread principal (o pacote systray
// chama runtime.LockOSThread no init). Iniciar de dentro do OnStartup fazia a
// janela/message loop do tray ficarem numa thread que não recebia WM_RBUTTONUP,
// e o menu de clique direito nunca abria.
func (a *App) startTray() {
	if a.statusCh == nil {
		a.statusCh = make(chan string, 8)
	}
	a.tray = NewTrayManager(a.statusCh)
	a.tray.Start(a)
}

// bridgeStatusUpdates repassa os status do WebSocket para o canal do tray.
// O wsManager só existe após o startup/config, então o bridge é instalado aqui.
func (a *App) bridgeStatusUpdates() {
	if wsManager == nil {
		return
	}
	go func() {
		for s := range wsManager.StatusUpdates() {
			select {
			case a.statusCh <- s:
			default:
			}
		}
	}()
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	if err := setAutoStart(true); err != nil {
		log.Printf("event=autostart_set_failed error=%q", err.Error())
	}

	cfg, err := LoadConfigFromFile()
	if err != nil {
		log.Printf("event=config_not_found reason=first_run")
		return
	}

	if err := ValidateConfig(cfg); err != nil {
		log.Printf("event=config_invalid error=%q", err.Error())
		wailsruntime.EventsEmit(a.ctx, "config:invalid", err.Error())
		return
	}

	log.Printf("event=startup ws_url=%s api_url=%s hot_folder_configured=%t",
		cfg.WSURL, cfg.APIURL, cfg.HotFolderPath != "")

	wsManager = NewWebSocketManager(cfg.WSURL, cfg.APIURL, cfg.AgentKey, cfg.HotFolderPath, cfg.DeviceID, cfg.DeviceName, cfg.ToPrinterConfig())
	a.bridgeStatusUpdates()

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

func (a *App) GetAutoStartEnabled() bool {
	return getAutoStartEnabled()
}

func (a *App) GetVersion() string {
	return Version
}

func (a *App) SetAutoStartEnabled(enable bool) error {
	return setAutoStart(enable)
}

func (a *App) GetAgentConfig() AgentConfig {
	cfg, err := LoadConfigFromFile()
	if err != nil {
		return AgentConfig{}
	}
	return *cfg
}

func (a *App) SaveAgentConfig(wsURL string, apiURL string, agentKey string, hotFolderPath string, deviceName string) error {
	// Load existing config to preserve DeviceID
	existing, _ := LoadConfigFromFile()
	cfg := AgentConfig{
		WSURL:         wsURL,
		APIURL:        apiURL,
		AgentKey:      agentKey,
		HotFolderPath: hotFolderPath,
		DeviceName:    deviceName,
	}
	if existing != nil {
		cfg.DeviceID = existing.DeviceID
		cfg.PrinterPhoto = existing.PrinterPhoto
		cfg.PrinterLetter = existing.PrinterLetter
	}
	if cfg.DeviceName == "" && existing != nil {
		cfg.DeviceName = existing.DeviceName
	}

	if err := ValidateConfig(&cfg); err != nil {
		return err
	}

	if err := SaveConfigFile(cfg); err != nil {
		return err
	}

	wsManager = NewWebSocketManager(cfg.WSURL, cfg.APIURL, cfg.AgentKey, cfg.HotFolderPath, cfg.DeviceID, cfg.DeviceName, cfg.ToPrinterConfig())
	a.bridgeStatusUpdates()
	wailsruntime.EventsEmit(a.ctx, "ws:status", "connecting")
	if err := wsManager.Connect(); err != nil {
		log.Printf("event=websocket_connect_after_config_failed error=%q", err.Error())
		wailsruntime.EventsEmit(a.ctx, "ws:status", "disconnected")
		return err
	}
	wailsruntime.EventsEmit(a.ctx, "ws:status", wsManager.ConnectionStatus())
	wsManager.StartListening(a.ctx)
	a.startUpdateTicker(a.ctx, cfg.APIURL, cfg.AgentKey)
	return nil
}

// ── Existing bindings (preservados) ──────────────────

func (a *App) GetStatus() string {
	if wsManager != nil && wsManager.IsConnecting() {
		return "connecting"
	}
	if wsManager != nil {
		return wsManager.ConnectionStatus()
	}
	return "disconnected"
}

func (a *App) GetDashboardSnapshot() DashboardSnapshot {
	snapshot := DashboardSnapshot{Status: a.GetStatus(), Jobs: []DashboardJob{}}
	if wsManager == nil {
		return snapshot
	}

	cfg := wsManager.GetPrinterConfig()
	if cfg.Photo != nil {
		snapshot.Photo = *cfg.Photo
	}
	if cfg.Letter != nil {
		snapshot.Letter = *cfg.Letter
	}
	if wsManager.jobStore == nil {
		return snapshot
	}

	today := time.Now().Format("2006-01-02")
	for _, entry := range wsManager.jobStore.dashboardJobs() {
		if entry.CreatedAt.Format("2006-01-02") == today {
			snapshot.Today++
		}
		switch entry.Status {
		case jobStatusPrinted:
			if entry.UpdatedAt.Format("2006-01-02") == today {
				snapshot.Printed++
			}
		case jobStatusReceived, jobStatusPrinting:
			snapshot.Queued++
		case jobStatusFailed:
			if entry.UpdatedAt.Format("2006-01-02") == today {
				snapshot.Failed++
			}
		}

		files := make([]DashboardJobFile, 0, len(entry.Job.Files))
		role := ""
		for _, file := range entry.Job.Files {
			files = append(files, DashboardJobFile{Name: file.Name, Type: file.Type})
			if role == "" {
				role = file.PrinterRole
			}
		}
		snapshot.Jobs = append(snapshot.Jobs, DashboardJob{
			ID: entry.Job.JobID, Customer: entry.Job.CustomerName, Status: entry.Status,
			CreatedAt: entry.CreatedAt.Format(time.RFC3339), UpdatedAt: entry.UpdatedAt.Format(time.RFC3339),
			PrinterRole: role, Files: files, LastError: entry.LastError,
		})
	}
	return snapshot
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

func (a *App) GetPrintSettings() map[string]interface{} {
	if wsManager == nil {
		return map[string]interface{}{}
	}
	cfg := wsManager.GetPrinterConfig()
	result := map[string]interface{}{}
	if cfg.PhotoSettings != nil {
		result["photoSettings"] = cfg.PhotoSettings
	}
	if cfg.LetterSettings != nil {
		result["letterSettings"] = cfg.LetterSettings
	}
	return result
}

func (a *App) ListSavedArts() ([]SavedArtInfo, error) {
	cfg, err := LoadConfigFromFile()
	if err != nil || cfg.HotFolderPath == "" {
		return []SavedArtInfo{}, nil
	}

	if err := os.MkdirAll(cfg.HotFolderPath, 0755); err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(cfg.HotFolderPath)
	if err != nil {
		return nil, err
	}

	arts := make([]SavedArtInfo, 0, len(entries))
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			log.Printf("event=saved_art_stat_failed name=%q error=%q", entry.Name(), err.Error())
			continue
		}

		arts = append(arts, SavedArtInfo{
			Name:       entry.Name(),
			Path:       filepath.Join(cfg.HotFolderPath, entry.Name()),
			SizeBytes:  info.Size(),
			ModifiedAt: info.ModTime().Format(time.RFC3339),
			IsDir:      entry.IsDir(),
		})
	}

	sort.Slice(arts, func(i, j int) bool {
		return arts[i].ModifiedAt > arts[j].ModifiedAt
	})

	return arts, nil
}

func (a *App) OpenHotFolder() error {
	cfg, err := LoadConfigFromFile()
	if err != nil || cfg.HotFolderPath == "" {
		return nil
	}

	if err := os.MkdirAll(cfg.HotFolderPath, 0755); err != nil {
		return err
	}

	var cmd *exec.Cmd
	switch stdruntime.GOOS {
	case "windows":
		cmd = exec.Command("explorer", cfg.HotFolderPath)
	case "darwin":
		cmd = exec.Command("open", cfg.HotFolderPath)
	default:
		cmd = exec.Command("xdg-open", cfg.HotFolderPath)
	}

	return cmd.Start()
}

func (a *App) ClearHotFolder() error {
	cfg, err := LoadConfigFromFile()
	if err != nil || cfg.HotFolderPath == "" {
		return nil
	}

	if err := os.MkdirAll(cfg.HotFolderPath, 0755); err != nil {
		return err
	}

	entries, err := os.ReadDir(cfg.HotFolderPath)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		path := filepath.Join(cfg.HotFolderPath, entry.Name())
		if err := os.RemoveAll(path); err != nil {
			return err
		}
	}

	wailsruntime.EventsEmit(a.ctx, "arts:changed")
	return nil
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
	_ = SavePrinterConfigToFile(cfg)
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
	wailsruntime.EventsEmit(a.ctx, "ws:status", "connecting")
	if err := wsManager.Connect(); err != nil {
		log.Printf("event=websocket_reconnect_failed error=%q", err.Error())
		wailsruntime.EventsEmit(a.ctx, "ws:status", "disconnected")
		return err
	}
	wailsruntime.EventsEmit(a.ctx, "ws:status", wsManager.ConnectionStatus())
	return nil
}

func (a *App) Disconnect() error {
	if wsManager == nil {
		return nil
	}
	wsManager.Stop()
	wailsruntime.EventsEmit(a.ctx, "ws:status", "disconnected")
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
					wailsruntime.EventsEmit(a.ctx, "app:update", info)
				}
			}
		}
	}()
}

func (a *App) MinimizeToTray() {
	if a.ctx == nil {
		return
	}
	wailsruntime.WindowHide(a.ctx)
}

func (a *App) ShowWindow() {
	if a.ctx == nil {
		return
	}
	wailsruntime.WindowShow(a.ctx)
	wailsruntime.WindowUnminimise(a.ctx)
	wailsruntime.WindowSetAlwaysOnTop(a.ctx, true)
	wailsruntime.WindowSetAlwaysOnTop(a.ctx, false)
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
		wailsruntime.Quit(a.ctx)
	}
}

func (a *App) ShouldQuit() bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.allowQuit
}

// GetPrinterPaperSizes returns supported paper sizes for a printer
func (a *App) GetPrinterPaperSizes(printerName string) ([]PaperSizeInfo, error) {
	return GetPrinterPaperSizes(printerName)
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
