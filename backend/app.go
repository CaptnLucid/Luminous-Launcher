// backend/app.go
package backend

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
)

type App struct {
	ctx     context.Context
	baseDir string
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

// ApplyApplicationUpdate streams down the raw asset attachment and handles process swaps
func (a *App) ApplyApplicationUpdate(downloadURL string) string {
	currentExe, err := os.Executable()
	if err != nil {
		return "Error finding current executable path: " + err.Error()
	}

	// Create path for the temporary installer payload stream
	tempNewExe := currentExe + ".tmp"

	// 1. Download the executable payload binary
	resp, err := http.Get(downloadURL)
	if err != nil {
		return "Download failed: " + err.Error()
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Sprintf("Server returned bad status code: %d", resp.StatusCode)
	}

	out, err := os.Create(tempNewExe)
	if err != nil {
		return "Failed to initialize temporary storage path: " + err.Error()
	}
	
	_, err = io.Copy(out, resp.Body)
	out.Close() // Explicitly close file buffer locks before shifting
	if err != nil {
		return "Failed streaming attachment data: " + err.Error()
	}

	// 2. Spawn a detached cmd background worker loop script 
	// This kills the engine, waits a fraction of a second, replaces the file, and runs the new app.
	cmdScript := fmt.Sprintf(
		"taskkill /F /PID %d && timeout /T 1 /NOBREAK && move /Y \"%s\" \"%s\" && start \"\" \"%s\"",
		os.Getpid(), tempNewExe, currentExe, currentExe,
	)

	worker := exec.Command("cmd", "/C", cmdScript)
	worker.SysProcAttr = &syscall.SysProcAttr{HideWindow: true} // Silently run completely hidden

	if err := worker.Start(); err != nil {
		return "Process swapping failure: " + err.Error()
	}

	// Exit the current running main app thread gracefully to unlock the system handle
	os.Exit(0)
	return "Success"
}

func (a *App) ExecuteGame(mode string, customPath string, nipPath string, hexMask string) string {
    config := LauncherConfig{
        LaunchMode:    mode,
        CustomExePath: customPath,
    }

    // 💡 Fix: If a custom path override is provided, use it directly instead of resolving a default
    targetExe := customPath
    if targetExe == "" {
        targetExe = config.ResolveExePath()
    }

    if nipPath != "" {
        if err := InjectNipProfile(a.baseDir, nipPath); err != nil {
            return "Profile Injection Error: " + err.Error()
        }
    }

    // This correctly passes down your exact D:\... path, and mode == "steam" still catches your check box!
    err := SpawnGameWithAffinity(targetExe, mode == "steam", hexMask)
    if err != nil {
        return "Launch Error: " + err.Error()
    }

    return "Success"
}

func (a *App) GetCurrentVersion() string {
	return CurrentVersion
}

// GetSettings handles transmitting the saved or default config state to the UI
func (a *App) GetSettings() AppConfig {
	return LoadConfig()
}

// SaveSettingUpdate updates specific properties on the fly from UI elements
func (a *App) SaveSettingUpdate(path string, isSteam bool, affinity string) bool {
	config := AppConfig{
		ExecutablePath: path,
		IsSteam:        isSteam,
		AffinityMask:   affinity,
	}
	err := SaveConfig(config)
	return err == nil
}