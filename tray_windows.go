//go:build windows

package main

import (
	_ "embed"
	"log"

	"fyne.io/systray"
)

//go:embed assets/icon.ico
var trayIcon []byte

type TrayManager struct {
	statusUpdates <-chan string
	app           *App
	statusItem    *systray.MenuItem
	stopCh        chan struct{}
}

func NewTrayManager(statusUpdates <-chan string) *TrayManager {
	return &TrayManager{
		statusUpdates: statusUpdates,
		stopCh:        make(chan struct{}),
	}
}

func (tm *TrayManager) Start(app *App) {
	tm.app = app
	go systray.Run(tm.onReady, tm.onExit)
}

func (tm *TrayManager) Stop() {
	select {
	case <-tm.stopCh:
	default:
		close(tm.stopCh)
		systray.Quit()
	}
}

func (tm *TrayManager) onReady() {
	systray.SetIcon(trayIcon)
	systray.SetTitle("CdA")
	systray.SetTooltip("Cesto d'Amore - Agente ativo")

	openItem := systray.AddMenuItem("Abrir painel", "Mostra a janela do agente")
	tm.statusItem = systray.AddMenuItem("Status: 🔴 Desconectado", "Status atual do WebSocket")
	tm.statusItem.Disable()
	systray.AddSeparator()
	quitItem := systray.AddMenuItem("Sair", "Encerra o agente")

	go tm.consumeStatusUpdates()
	go func() {
		for {
			select {
			case <-tm.stopCh:
				return
			case <-openItem.ClickedCh:
				if tm.app != nil {
					tm.app.ShowWindow()
				}
			case <-quitItem.ClickedCh:
				if tm.app != nil {
					tm.app.QuitApp()
				}
				return
			}
		}
	}()

	log.Printf("event=tray_started")
}

func (tm *TrayManager) onExit() {
	log.Printf("event=tray_stopped")
}

func (tm *TrayManager) consumeStatusUpdates() {
	for {
		select {
		case <-tm.stopCh:
			return
		case status := <-tm.statusUpdates:
			tm.setStatus(status)
		}
	}
}

func (tm *TrayManager) setStatus(status string) {
	if tm.statusItem == nil {
		return
	}

	if status == "connected" {
		tm.statusItem.SetTitle("Status: 🟢 Conectado")
		return
	}
	if status == "inactive" {
		tm.statusItem.SetTitle("Status: 🟡 Inativo")
		return
	}

	tm.statusItem.SetTitle("Status: 🔴 Desconectado")
}
