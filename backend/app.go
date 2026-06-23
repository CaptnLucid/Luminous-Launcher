package backend

import (
	"context"
	"os"
	"path/filepath"
)

type App struct {
	ctx      context.Context
	baseDir  string
}

func NewApp() *App {
	return &App{}
}

func (a *App) StartUp(ctx context.Context) {
	a.ctx = ctx
	exePath, _ := os.Executable()
	a.baseDir = filepath.Dir(exePath)
}

func (a *App) CheckLauncherUpdates() (*UpdateStatus, error) {
	return CheckForUpdates()
}

func (a *App) LoadAvailableProfiles() map[string]string {
	return ScanNipProfiles(a.baseDir)
}

func (a *App) ExecuteGame(mode string, customPath string, nipPath string, hexMask string) string {
	config := LauncherConfig{
		LaunchMode:    mode,
		CustomExePath: customPath,
	}

	targetExe := config.ResolveExePath()

	if nipPath != "" {
		if err := InjectNipProfile(a.baseDir, nipPath); err != nil {
			return "Profile Injection Error:" + err.Error()
		}
	}

	err := SpawnGameWithAffinity(targetExe, mode == "steam", hexMask)
	if err != nil {
		return "Launch Error: " + err.Error()
	}

	return "Success"
}