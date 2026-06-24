package backend

import (
	"archive/zip"
	"bytes"
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

	tempNewExe := currentExe + ".tmp"

	// 1. Download the compressed ZIP file asset package data stream
	resp, err := http.Get(downloadURL)
	if err != nil {
		return "Download failed: " + err.Error()
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Sprintf("Server returned bad status code: %d", resp.StatusCode)
	}

	// Read downloaded payload directly into memory buffers to avoid wasting local disk operations
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "Failed to buffer update package payload: " + err.Error()
	}

	// 2. Open up the compressed in-memory archive engine 
	zipReader, err := zip.NewReader(bytes.NewReader(bodyBytes), int64(len(bodyBytes)))
	if err != nil {
		return "Downloaded package asset is not a valid zip archive: " + err.Error()
	}

	// 3. Locate and extract the target executable from inside the archive bounds
	var targetFileInZip *zip.File
	for _, file := range zipReader.File {
		if filepath.Ext(file.Name) == ".exe" {
			targetFileInZip = file
			break
		}
	}

	if targetFileInZip == nil {
		return "Invalid patch layout: compiled executable asset could not be found within zip payload"
	}

	zipFileStream, err := targetFileInZip.Open()
	if err != nil {
		return "Failed opening internal file entry handle: " + err.Error()
	}
	defer zipFileStream.Close()

	// 4. Stream extraction payload directly out onto the disk layout envelope
	out, err := os.Create(tempNewExe)
	if err != nil {
		return "Failed to initialize temporary storage path: " + err.Error()
	}

	_, err = io.Copy(out, zipFileStream)
	out.Close() 
	if err != nil {
		return "Failed streaming unpacked binary payload context: " + err.Error()
	}

	// 5. Spawn a detached cmd background worker loop script 
	cmdScript := fmt.Sprintf(
		"taskkill /F /PID %d && timeout /T 1 /NOBREAK && move /Y \"%s\" \"%s\" && start \"\" \"%s\"",
		os.Getpid(), tempNewExe, currentExe, currentExe,
	)

	worker := exec.Command("cmd", "/C", cmdScript)
	worker.SysProcAttr = &syscall.SysProcAttr{HideWindow: true} 

	if err := worker.Start(); err != nil {
		return "Process swapping failure: " + err.Error()
	}

	os.Exit(0)
	return "Success"
}

func (a *App) ExecuteGame(mode string, customPath string, nipPath string, hexMask string) string {
	config := LauncherConfig{
		LaunchMode:    mode,
		CustomExePath: customPath,
	}

	targetExe := customPath
	if targetExe == "" {
		targetExe = config.ResolveExePath()
	}

	if nipPath != "" {
		if err := InjectNipProfile(a.baseDir, nipPath); err != nil {
			return "Profile Injection Error: " + err.Error()
		}
	}

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

// DetectCPUMask exposes the hardware topology parser to the Wails frontend.
// The Win32 sizing syscall must NOT run on the Wails/JS bridge goroutine — doing
// so blocks the entire IPC layer and freezes the UI. We hand the work off to a
// fresh goroutine, wait for it on a channel, and return once it finishes.
func (a *App) DetectCPUMask() map[string]interface{} {
	type result struct {
		mask string
		logs []string
	}

	ch := make(chan result, 1)

	go func() {
		mask, logs := DetectOptimalAffinityMask()
		ch <- result{mask, logs}
	}()

	r := <-ch
	return map[string]interface{}{
		"mask": r.mask,
		"logs": r.logs,
	}
}