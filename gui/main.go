// Command awg-gui is a native desktop app (Wails) that installs and manages a
// self-hosted AmneziaWG VPN on a remote Linux server over SSH — a point-and-click
// front end over the same logic the CLI (cmd/awg-deploy) drives.
//
// It lives in its own Go module (gui/) so the WebKit/GTK-bound Wails dependency
// never breaks the root module's headless CI.
package main

import (
	"embed"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
var assets embed.FS

// appVersion is the build's version, injected at release time via
//
//	-ldflags "-X main.appVersion=v1.2.3"
//
// It stays "dev" for local builds (the update check then never nags).
var appVersion = "dev"

func main() {
	app := NewApp()

	err := wails.Run(&options.App{
		Title:     "AmneziaWG Manager",
		Width:     1080,
		Height:    740,
		MinWidth:  900,
		MinHeight: 600,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 14, G: 17, B: 22, A: 1},
		OnStartup:        app.startup,
		OnShutdown:       app.shutdown,
		Bind: []interface{}{
			app,
		},
	})
	if err != nil {
		println("error:", err.Error())
	}
}
