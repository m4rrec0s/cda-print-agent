package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type PrinterConfigMap struct {
	Photo  *string `json:"photo"`
	Letter *string `json:"letter"`
}

type WSMessage struct {
	Type            string            `json:"type"`
	JobID           string            `json:"jobId,omitempty"`
	Job             *PrintJob         `json:"job,omitempty"`
	Available       bool              `json:"available,omitempty"`
	Printers        []string          `json:"printers,omitempty"`
	PrinterDetails  []PrinterInfo     `json:"printerDetails,omitempty"`
	SelectedPrinter string            `json:"selectedPrinter,omitempty"`
	Config          *PrinterConfigMap `json:"config,omitempty"`
	Version         string            `json:"version,omitempty"`
	DownloadURL     string            `json:"downloadUrl,omitempty"`
	ReleaseNotes    string            `json:"releaseNotes,omitempty"`
	FileIndex       int               `json:"fileIndex,omitempty"`
	FileStatus      string            `json:"fileStatus,omitempty"`
	Timestamp       string            `json:"timestamp,omitempty"`
	Error           string            `json:"error,omitempty"`
}

type WebSocketManager struct {
	conn            *websocket.Conn
	url             string
	apiURL          string
	agentKey        string
	hotFolderPath   string
	printerConfig   PrinterConfigMap
	connected       bool
	connecting      bool
	done            chan struct{}
	stopOnce        sync.Once
	writeMu         sync.Mutex
	mu              sync.RWMutex
	statusCh        chan string
	processedJobs   map[string]bool
	processedJobsMu sync.RWMutex
}

var wsManager *WebSocketManager

func NewWebSocketManager(url string, apiURL string, agentKey string, hotFolderPath string) *WebSocketManager {
	return &WebSocketManager{
		url:           url,
		apiURL:        apiURL,
		agentKey:      agentKey,
		hotFolderPath: hotFolderPath,
		done:          make(chan struct{}),
		statusCh:      make(chan string, 8),
		processedJobs: make(map[string]bool),
	}
}

func (wm *WebSocketManager) resolvePrinter(role string) string {
	wm.mu.RLock()
	defer wm.mu.RUnlock()

	switch role {
	case "letter":
		if wm.printerConfig.Letter != nil {
			return *wm.printerConfig.Letter
		}
	case "photo":
		if wm.printerConfig.Photo != nil {
			return *wm.printerConfig.Photo
		}
	}
	return ""
}

func (wm *WebSocketManager) GetPrinterConfig() PrinterConfigMap {
	wm.mu.RLock()
	defer wm.mu.RUnlock()
	return wm.printerConfig
}

func (wm *WebSocketManager) StatusUpdates() <-chan string {
	return wm.statusCh
}

func (wm *WebSocketManager) Connect() error {
	wm.setConnecting(true)
	headers := http.Header{}
	if wm.agentKey != "" {
		headers.Set("X-Agent-Key", wm.agentKey)
		headers.Set("X-API-Key", wm.agentKey)
	}

	conn, _, err := websocket.DefaultDialer.Dial(wm.url, headers)
	if err != nil {
		wm.setConnected(false)
		return err
	}

	wm.mu.Lock()
	wm.conn = conn
	wm.connected = true
	wm.connecting = false
	wm.mu.Unlock()
	wm.emitStatus("connected")

	conn.SetReadDeadline(time.Now().Add(90 * time.Second))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(90 * time.Second))
		return nil
	})
	conn.SetPingHandler(func(appData string) error {
		conn.SetReadDeadline(time.Now().Add(90 * time.Second))
		wm.writeMu.Lock()
		defer wm.writeMu.Unlock()
		return conn.WriteControl(websocket.PongMessage, []byte(appData), time.Now().Add(10*time.Second))
	})

	log.Printf("event=websocket_connected url=%s", wm.url)

	// Request printer config from backend on connection
	msg := WSMessage{
		Type:      "SYNC_PRINTER_CONFIG",
		Timestamp: time.Now().Format(time.RFC3339),
	}
	if err := wm.SendMessage(msg); err != nil {
		log.Printf("event=sync_printer_config_failed error=%q", err.Error())
	}

	return nil
}

func (wm *WebSocketManager) IsConnected() bool {
	wm.mu.RLock()
	defer wm.mu.RUnlock()
	return wm.connected && wm.conn != nil
}

func (wm *WebSocketManager) IsConnecting() bool {
	wm.mu.RLock()
	defer wm.mu.RUnlock()
	return wm.connecting
}

func (wm *WebSocketManager) Close() error {
	wm.mu.Lock()
	conn := wm.conn
	wm.conn = nil
	wm.connected = false
	wm.connecting = false
	wm.mu.Unlock()
	wm.emitStatus("disconnected")

	if conn != nil {
		return conn.Close()
	}
	return nil
}

func (wm *WebSocketManager) SendMessage(msg WSMessage) error {
	wm.mu.RLock()
	conn := wm.conn
	connected := wm.connected
	wm.mu.RUnlock()

	if !connected || conn == nil {
		return fmt.Errorf("websocket not connected")
	}

	msg.Timestamp = time.Now().Format("15:04:05")
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	wm.writeMu.Lock()
	conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
	err = conn.WriteMessage(websocket.TextMessage, data)
	wm.writeMu.Unlock()
	if err != nil {
		return err
	}

	log.Printf("event=websocket_message_sent type=%s job_id=%s", msg.Type, msg.JobID)
	return nil
}

func (wm *WebSocketManager) StartListening(ctx context.Context) {
	// Emit current status immediately
	if wm.IsConnected() {
		runtime.EventsEmit(ctx, "ws:status", "connected")
	} else {
		runtime.EventsEmit(ctx, "ws:status", "disconnected")
	}

	go wm.startPingLoop(ctx)
	go wm.readLoop(ctx)
}

func (wm *WebSocketManager) Stop() {
	wm.stopOnce.Do(func() {
		close(wm.done)
		_ = wm.Close()
	})
}

func (wm *WebSocketManager) readLoop(ctx context.Context) {
	for {
		select {
		case <-wm.done:
			return
		default:
		}

		if !wm.IsConnected() {
			time.Sleep(5 * time.Second)
			runtime.EventsEmit(ctx, "ws:status", "connecting")
			if err := wm.Connect(); err != nil {
				log.Printf("event=websocket_reconnect_failed error=%q", err.Error())
				runtime.EventsEmit(ctx, "ws:status", "disconnected")
				continue
			}
			runtime.EventsEmit(ctx, "ws:status", "connected")
		}

		wm.mu.RLock()
		conn := wm.conn
		wm.mu.RUnlock()
		if conn == nil {
			continue
		}

		var msg WSMessage
		if err := conn.ReadJSON(&msg); err != nil {
			log.Printf("event=websocket_read_failed error=%q", err.Error())
			_ = wm.Close()
			runtime.EventsEmit(ctx, "ws:status", "disconnected")
			continue
		}

		log.Printf("event=websocket_message_received type=%s job_id=%s", msg.Type, msg.JobID)
		runtime.EventsEmit(ctx, "ws:message", map[string]string{
			"direction": "in",
			"content":   fmt.Sprintf("{\"type\":\"%s\"}", msg.Type),
			"timestamp": time.Now().Format("15:04:05"),
		})

		switch msg.Type {
		case "CHECK_PRINTER":
			wm.handleCheckPrinter(ctx)
		case "AUTHORIZE_PRINTER":
			wm.handleAuthorizePrinter(ctx, msg)
		case "PRINT_JOB":
			wm.handlePrintJob(ctx, msg)
		case "PRINTER_CONFIG_UPDATE":
			wm.handlePrinterConfigUpdate(ctx, msg)
		case "UPDATE_AVAILABLE":
			wm.handleUpdateAvailable(ctx, msg)
		}
	}
}

func (wm *WebSocketManager) handleCheckPrinter(ctx context.Context) {
	printers, err := GetPrinters()
	if err != nil {
		log.Printf("event=get_printers_failed error=%q", err.Error())
	}
	printerDetails, _ := GetPrintersWithStatus()
	available := len(printers) > 0

	response := WSMessage{
		Type:           "PRINTER_STATUS",
		Available:      available,
		Printers:       printers,
		PrinterDetails: printerDetails,
	}

	if err := wm.SendMessage(response); err != nil {
		log.Printf("event=printer_status_send_failed error=%q", err.Error())
	}

	runtime.EventsEmit(ctx, "ws:message", map[string]string{
		"direction": "out",
		"content":   fmt.Sprintf("{\"type\":\"PRINTER_STATUS\",\"available\":%t}", available),
		"timestamp": time.Now().Format("15:04:05"),
	})
	runtime.EventsEmit(ctx, "ws:printers", printers)
}

func (wm *WebSocketManager) handleAuthorizePrinter(ctx context.Context, msg WSMessage) {
	wm.mu.Lock()
	if msg.SelectedPrinter == "pdf_fallback" {
		wm.printerConfig = PrinterConfigMap{Photo: nil, Letter: nil}
	} else {
		s := msg.SelectedPrinter
		wm.printerConfig.Photo = &s
	}
	wm.mu.Unlock()

	log.Printf("event=printer_authorized printer=%q", msg.SelectedPrinter)
	runtime.EventsEmit(ctx, "ws:message", map[string]string{
		"direction": "in",
		"content":   fmt.Sprintf("{\"type\":\"AUTHORIZE_PRINTER\",\"selectedPrinter\":\"%s\"}", msg.SelectedPrinter),
		"timestamp": time.Now().Format("15:04:05"),
	})
	runtime.EventsEmit(ctx, "ws:printerConfig", wm.GetPrinterConfig())
}

func (wm *WebSocketManager) handlePrinterConfigUpdate(ctx context.Context, msg WSMessage) {
	if msg.Config == nil {
		log.Printf("event=printer_config_update_invalid reason=missing_config")
		return
	}

	wm.mu.Lock()
	wm.printerConfig = *msg.Config
	wm.mu.Unlock()

	log.Printf("event=printer_config_updated photo=%v letter=%v", msg.Config.Photo, msg.Config.Letter)
	runtime.EventsEmit(ctx, "ws:printerConfig", msg.Config)
}

func (wm *WebSocketManager) handleUpdateAvailable(ctx context.Context, msg WSMessage) {
	info := &VersionInfo{
		Version:      msg.Version,
		DownloadURL:  msg.DownloadURL,
		ReleaseNotes: msg.ReleaseNotes,
	}
	if info.Version == "" || info.DownloadURL == "" {
		log.Printf("event=update_available_invalid reason=missing_version_or_url")
		return
	}

	log.Printf("event=update_available version=%s", info.Version)
	runtime.EventsEmit(ctx, "app:update", info)
}

func (wm *WebSocketManager) sendFileEvent(ctx context.Context, jobID string, fileIndex int, status string, errMsg string) {
	msg := WSMessage{
		Type:       status,
		JobID:      jobID,
		FileIndex:  fileIndex,
		FileStatus: status,
	}
	if errMsg != "" {
		msg.Error = errMsg
	}
	_ = wm.SendMessage(msg)
}

func (wm *WebSocketManager) handlePrintJob(ctx context.Context, msg WSMessage) {
	if msg.Job == nil {
		log.Printf("event=print_job_invalid reason=missing_job")
		return
	}

	job := *msg.Job
	if job.JobID == "" {
		job.JobID = msg.JobID
	}
	if job.JobID == "" {
		job.JobID = job.OrderID
	}

	// Verificar se esse job já foi processado nessa sessão
	wm.processedJobsMu.RLock()
	alreadyProcessed := wm.processedJobs[job.JobID]
	wm.processedJobsMu.RUnlock()

	if alreadyProcessed {
		log.Printf("event=print_job_skipped reason=already_processed job_id=%s", job.JobID)
		_ = wm.SendMessage(WSMessage{Type: "PRINTED", JobID: job.JobID})
		return
	}

	wm.processedJobsMu.Lock()
	wm.processedJobs[job.JobID] = true
	wm.processedJobsMu.Unlock()

	if job.OrderID == "" || job.CustomerName == "" || job.DriveFolderID == "" || len(job.Files) == 0 {
		errMsg := fmt.Sprintf("payload incompleto: orderId=%q customerName=%q driveFolderId=%q files=%d", job.OrderID, job.CustomerName, job.DriveFolderID, len(job.Files))
		log.Printf("event=print_job_invalid job_id=%s reason=%q", job.JobID, errMsg)
		_ = wm.SendMessage(WSMessage{Type: "FAILED", JobID: job.JobID, Error: errMsg})
		return
	}

	for index, file := range job.Files {
		if file.Name == "" || file.DriveFileID == "" || file.PrinterRole == "" || file.Type == "" {
			errMsg := fmt.Sprintf("arquivo %d incompleto: name=%q driveFileId=%q printerRole=%q type=%q", index, file.Name, file.DriveFileID, file.PrinterRole, file.Type)
			log.Printf("event=print_job_invalid job_id=%s reason=%q", job.JobID, errMsg)
			_ = wm.SendMessage(WSMessage{Type: "FAILED", JobID: job.JobID, Error: errMsg})
			return
		}

		log.Printf(
			"event=print_job_file_received job_id=%s index=%d name=%q drive_file_id=%s subfolder=%q type=%s printer_role=%s size=%dx%d label=%q",
			job.JobID,
			index,
			file.Name,
			file.DriveFileID,
			file.SubfolderName,
			file.Type,
			file.PrinterRole,
			file.SizeConfig.WidthMm,
			file.SizeConfig.HeightMm,
			file.SizeConfig.Label,
		)
	}

	if err := wm.SendMessage(WSMessage{Type: "ACK", JobID: job.JobID}); err != nil {
		log.Printf("event=print_job_ack_failed job_id=%s error=%q", job.JobID, err.Error())
		return
	}

	runtime.EventsEmit(ctx, "ws:message", map[string]string{
		"direction": "out",
		"content":   fmt.Sprintf("{\"type\":\"ACK\",\"jobId\":\"%s\"}", job.JobID),
		"timestamp": time.Now().Format("15:04:05"),
	})

	go func() {
		resolvePrinter := func(role string) string {
			return wm.resolvePrinter(role)
		}

		emitBackend := func(stepType string, fileIndex int, errMsg string) {
			wm.sendFileEvent(ctx, job.JobID, fileIndex, stepType, errMsg)
		}

		err := ProcessPrintJob(ctx, wm.apiURL, wm.agentKey, wm.hotFolderPath, resolvePrinter, job, func(event JobUIEvent) {
			runtime.EventsEmit(ctx, "ws:job", event)
		}, emitBackend)

		if err != nil {
			log.Printf("event=print_job_failed job_id=%s error=%q", job.JobID, err.Error())
			_ = wm.SendMessage(WSMessage{Type: "FAILED", JobID: job.JobID, Error: err.Error()})
			return
		}

		log.Printf("event=print_job_completed job_id=%s", job.JobID)
		_ = wm.SendMessage(WSMessage{Type: "PRINTED", JobID: job.JobID})
		_ = wm.SendMessage(WSMessage{Type: "COMPLETED", JobID: job.JobID})
	}()
}

func (wm *WebSocketManager) startPingLoop(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-wm.done:
			return
		case <-ticker.C:
			wm.mu.RLock()
			conn := wm.conn
			connected := wm.connected
			wm.mu.RUnlock()

			if !connected || conn == nil {
				continue
			}

			wm.writeMu.Lock()
			conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			err := conn.WriteMessage(websocket.PingMessage, []byte{})
			wm.writeMu.Unlock()

			if err != nil {
				log.Printf("event=websocket_ping_failed error=%q", err.Error())
				_ = wm.Close()
				runtime.EventsEmit(ctx, "ws:status", "disconnected")
			}
		}
	}
}

func (wm *WebSocketManager) setConnected(connected bool) {
	wm.mu.Lock()
	wm.connected = connected
	wm.connecting = false
	wm.mu.Unlock()
	if connected {
		wm.emitStatus("connected")
	} else {
		wm.emitStatus("disconnected")
	}
}

func (wm *WebSocketManager) setConnecting(connecting bool) {
	wm.mu.Lock()
	wm.connecting = connecting
	wm.mu.Unlock()
	if connecting {
		wm.emitStatus("connecting")
	}
}

func (wm *WebSocketManager) emitStatus(status string) {
	select {
	case wm.statusCh <- status:
	default:
	}
}
