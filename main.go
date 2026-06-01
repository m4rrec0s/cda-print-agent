package main

import (
	"context"
	"embed"
	"log"

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
	app := NewApp()

	err := wails.Run(&options.App{
		Title:  "CdA: Print Agent",
		Width:  400,
		Height: 500,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 27, G: 38, B: 54, A: 1},
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
