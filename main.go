// main.go
package main

import (
	"BDOLauncher/backend" // Import backend logic from the subdirectory
	"embed"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	// Instantiate the app handler from the backend package
	app := backend.NewApp()

	err := wails.Run(&options.App{
		Title:  "BDO Custom Launcher",
		Width:  900,
		Height: 580,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		OnStartup: app.StartUp, // Wire application life-cycle setup
		Bind: []interface{}{
			app, // Exposes all public backend methods to TS
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}