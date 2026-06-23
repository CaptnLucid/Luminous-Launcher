package backend

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type AppConfig struct {
	ExecutablePath string `json:"executablePath"`
	IsSteam 	   bool   `json:"isSteam"`
	AffinityMask   string `json:"affinityMask"`
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
			ExecutablePath: "",
			IsSteam: false,
			AffinityMask: "FFFF",
		}
	}

	_ = json.Unmarshal(file, &config)
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