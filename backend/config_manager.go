// backend/config_manager.go
package backend

import (
	"os"
	"path/filepath"
	"strings"
)

type LauncherConfig struct {
	LaunchMode			string `json:"launch_mode"`
	CustomExePath		string `json:"custom_exe_path"`
	AffinityMask		string `json:"affinity_mask"`
	SelectedProfile		string `json:"selected_profile"`
}

const (
	SteamDefaultPath = `C:\Program Files (x86)\Steam\steamapps\common\Black Desert Online\BlackDesertLauncher.exe`
	PearlDefaultPath = `C:\Program Files (x86)\Black Desert Online\BlackDesertLauncher.exe`
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

func ScanNipProfiles(baseDir string) map[string]string {
	profiles := make (map[string]string)
	profilesDir := filepath.Join(baseDir, "profiles")

	_ = os.MkdirAll(profilesDir, 0755)

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