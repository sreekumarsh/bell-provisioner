package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// AppConfig is persisted user preferences (not secrets).
type AppConfig struct {
	GatewayURL     string `json:"gateway_url"`
	BackendProfile string `json:"backend_profile"`
	SSHUser        string `json:"ssh_user"`
	SSHHost        string `json:"ssh_host"`
	SSHPort        int    `json:"ssh_port"`
	MacIP          string `json:"mac_ip"`
	Phone          string `json:"phone"`
}

// Default returns sensible defaults for first launch.
func Default() AppConfig {
	return AppConfig{
		GatewayURL:     "https://api.vyooham.com",
		BackendProfile: string(ProfileVPS),
		SSHUser:        "pi",
		SSHHost:        "raspberrypi.local",
		SSHPort:        22,
		MacIP:          "",
	}
}

func configPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "bell-provisioner", "config.json"), nil
}

// Load reads saved config or returns defaults.
func Load() AppConfig {
	path, err := configPath()
	if err != nil {
		return Default()
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return Default()
	}
	var cfg AppConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Default()
	}
	if cfg.GatewayURL == "" {
		cfg.GatewayURL = Default().GatewayURL
	}
	if cfg.SSHUser == "" {
		cfg.SSHUser = "pi"
	}
	if cfg.SSHHost == "" {
		cfg.SSHHost = "raspberrypi.local"
	}
	if cfg.SSHPort == 0 {
		cfg.SSHPort = 22
	}
	if cfg.BackendProfile == "" {
		cfg.BackendProfile = string(ProfileVPS)
	}
	return cfg
}

// Save persists config to disk.
func Save(cfg AppConfig) error {
	path, err := configPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}
