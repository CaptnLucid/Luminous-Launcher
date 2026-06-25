// main.go
package main

import (
	"BDOLauncher/backend" // Import backend logic from the subdirectory
	"context"
	"embed"
	"os"

	"github.com/energye/systray"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

//go:embed all:frontend/dist
var assets embed.FS

// 💡 Force bind the icon directly from your build directory using an absolute compile hook
//go:embed build/windows/icon.ico
var localIconBytes []byte

func main() {
	// Instantiate the app handler from the backend package
	app := backend.NewApp()

	// 1. Kick off the system tray event handler loop
	go func() {
		systray.Run(func() {
			// 💡 Fix: Pass the hard-bound localIconBytes array explicitly inside the initialization loop
			if len(localIconBytes) > 0 {
				systray.SetIcon(localIconBytes)
			}
			
			systray.SetTooltip("Luminous BDO Launcher")

			// Generate the menu options manually
			showBtn := systray.AddMenuItem("Show Launcher", "Bring launcher to front")
			systray.AddSeparator()
			quitBtn := systray.AddMenuItem("Quit", "Exit completely")

			showBtn.Click(func() {
				ctx := app.GetContext()
				if ctx != nil {
					runtime.WindowShow(ctx)
				}
			})

			quitBtn.Click(func() {
				ctx := app.GetContext()
				if ctx != nil {
					runtime.Quit(ctx)
				} else {
					os.Exit(0)
				}
			})
			
		}, func() {
			// Optional lifecycle exit cleanup goes here
		})
	}()

	// 2. Launch the normal Wails application window frame
	err := wails.Run(&options.App{
		Title:  "BDO Custom Launcher",
		Width:  900,
		Height: 580,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		OnStartup: func(ctx context.Context) {
			app.StartUp(ctx)
		},
		Bind: []interface{}{
			app,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}