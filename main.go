package main

import (
	"context"
	"embed"
	"log"
	"os"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

//go:embed all:frontend/dist
var assets embed.FS

var mainWindowCtx context.Context
var Version = "dev"

func main() {
	isFirst, lockHandle := AcquireSingleInstanceLock()
	if !isFirst {
		FocusExistingInstance()
		return
	}
	defer ReleaseSingleInstanceLock(lockHandle)

	startHidden := false
	for _, arg := range os.Args[1:] {
		if arg == "--hidden" {
			startHidden = true
		}
	}

	app := NewApp()

	// Systray precisa ser iniciado numa goroutine própria ANTES do boot do Wails.
	// Veja startTray em app.go para a explicação do conflito de threads.
	app.startTray()

	err := wails.Run(&options.App{
		Title:     "CdA: Print Agent",
		Width:     460,
		Height:    640,
		MinWidth:  390,
		MinHeight: 500,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 27, G: 38, B: 54, A: 1},
		WindowStartState: func() options.WindowStartState {
			if startHidden {
				return options.Minimised
			}
			return options.Normal
		}(),
		OnStartup: func(ctx context.Context) {
			mainWindowCtx = ctx
			app.startup(ctx)
		},
		OnBeforeClose: func(ctx context.Context) (prevent bool) {
			if !app.ShouldQuit() {
				runtime.WindowHide(ctx)
				return true
			}
			return false
		},
		OnShutdown: app.shutdown,
		Bind: []interface{}{
			app,
		},
		Frameless:   false,
		AlwaysOnTop: false,
		Windows: &windows.Options{
			IsZoomControlEnabled: false,
		},
	})

	if err != nil {
		log.Printf("event=wails_run_failed error=%q", err.Error())
	}
}
