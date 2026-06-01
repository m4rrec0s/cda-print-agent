//go:build !windows

package main

import "log"

type TrayManager struct {
	statusUpdates <-chan string
	stopCh        chan struct{}
}

func NewTrayManager(statusUpdates <-chan string) *TrayManager {
	return &TrayManager{
		statusUpdates: statusUpdates,
		stopCh:        make(chan struct{}),
	}
}

func (tm *TrayManager) Start(app *App) {
	go func() {
		for {
			select {
			case <-tm.stopCh:
				return
			case status := <-tm.statusUpdates:
				log.Printf("event=tray_status_ignored platform=unsupported status=%s", status)
			}
		}
	}()
	log.Printf("event=tray_disabled platform=unsupported")
}

func (tm *TrayManager) Stop() {
	select {
	case <-tm.stopCh:
	default:
		close(tm.stopCh)
	}
}
