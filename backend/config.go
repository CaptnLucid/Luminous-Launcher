package backend

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type AppConfig struct {
	LaunchMode      string `json:"launchMode"`
	ExecutablePath  string `json:"executablePath"`
	IsSteam         bool   `json:"isSteam"`
	AffinityMask    string `json:"affinityMask"`
	SelectedProfile string `json:"selectedProfile"`
}

func GetConfigPath() string {
	configDir, _ := os.UserConfigDir()
	appDir := filepath.Join(configDir, "BDOLauncher")
	_ = os.MkdirAll(appDir, os.ModePerm)
	return filepath.Join(appDir, "config.json")
}

func LoadConfig() AppConfig {
	path := GetConfigPath()
	var config AppConfig

	file, err := os.ReadFile(path)
	if err != nil {
		return AppConfig{
			LaunchMode:      "pearl",
			ExecutablePath:  "",
			IsSteam:         false,
			AffinityMask:    "FFFF",
			SelectedProfile: "",
		}
	}

	_ = json.Unmarshal(file, &config)

	// Older config files saved before LaunchMode existed will decode it as "".
	// Fall back to the previous heuristic so existing installs don't silently
	// reset to a mode the user didn't pick.
	if config.LaunchMode == "" {
		if config.ExecutablePath != "" {
			config.LaunchMode = "custom"
		} else if config.IsSteam {
			config.LaunchMode = "steam"
		} else {
			config.LaunchMode = "pearl"
		}
	}

	return config
}

func SaveConfig(config AppConfig) error {
	path := GetConfigPath()
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}