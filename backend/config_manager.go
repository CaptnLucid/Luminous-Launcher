// backend/config_manager.go
package backend

import (
	"io"
	"os"
	"path/filepath"
	"strings"
)

type LauncherConfig struct {
	LaunchMode      string `json:"launch_mode"`
	CustomExePath   string `json:"custom_exe_path"`
	AffinityMask    string `json:"affinity_mask"`
	SelectedProfile string `json:"selected_profile"`
}

const (
	SteamDefaultPath = `C:\Program Files (x86)\Steam\steamapps\common\Black Desert Online\BlackDesertLauncher.exe`
	PearlDefaultPath = `C:\Pearlabyss\BlackDesert\BlackDesertLauncher.exe`
)

func (c *LauncherConfig) ResolveExePath() string {
	switch c.LaunchMode {
	case "steam":
		return SteamDefaultPath
	case "custom":
		return c.CustomExePath
	case "pearl":
		return PearlDefaultPath
	default:
		return PearlDefaultPath
	}
}

// AutoPopulateEnvironment ensures directories exist, copying missing assets if needed
func AutoPopulateEnvironment(baseDir string) {
	profilesDir := filepath.Join(baseDir, "profiles")
	inspectorDir := filepath.Join(baseDir, "nvidiaProfileInspector")
	localAssetsDir := filepath.Join(baseDir, "assets")

	// Always make sure the directories themselves exist
	_ = os.MkdirAll(profilesDir, 0755)
	_ = os.MkdirAll(inspectorDir, 0755)

	// If the bundled assets folder is present, copy dependencies into place
	if _, err := os.Stat(localAssetsDir); err == nil {
		// Auto-populate inspector binary if missing
		srcInspector := filepath.Join(localAssetsDir, "nvidiaProfileInspector", "nvidiaProfileInspector.exe")
		destInspector := filepath.Join(inspectorDir, "nvidiaProfileInspector.exe")
		if _, err := os.Stat(destInspector); os.IsNotExist(err) {
			copyFile(srcInspector, destInspector)
		}

		// Auto-populate starter profiles
		srcProfilesDir := filepath.Join(localAssetsDir, "profiles")
		if files, err := os.ReadDir(srcProfilesDir); err == nil {
			for _, file := range files {
				if !file.IsDir() && strings.HasSuffix(strings.ToLower(file.Name()), ".nip") {
					destProfilePath := filepath.Join(profilesDir, file.Name())
					if _, err := os.Stat(destProfilePath); os.IsNotExist(err) {
						copyFile(filepath.Join(srcProfilesDir, file.Name()), destProfilePath)
					}
				}
			}
		}
	}
}

func ScanNipProfiles(baseDir string) map[string]string {
	// Trigger environment check prior to scanning layouts
	AutoPopulateEnvironment(baseDir)

	profiles := make(map[string]string)
	profilesDir := filepath.Join(baseDir, "profiles")

	files, err := os.ReadDir(profilesDir)
	if err != nil {
		return profiles
	}

	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(strings.ToLower(file.Name()), ".nip") {
			name := strings.TrimSuffix(file.Name(), filepath.Ext(file.Name()))
			profiles[name] = filepath.Join(profilesDir, file.Name())
		}
	}
	return profiles
}

// Helper utility to clone file contents cleanly
func copyFile(src, dst string) {
	sourceFile, err := os.Open(src)
	if err != nil {
		return
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return
	}
	defer destFile.Close()

	_, _ = io.Copy(destFile, sourceFile)
}