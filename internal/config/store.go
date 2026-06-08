package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// AppConfig is persisted user preferences. github_token is stored locally (chmod 0600).
type AppConfig struct {
	GatewayURL        string `json:"gateway_url"`
	BackendProfile    string `json:"backend_profile"`
	SSHUser           string `json:"ssh_user"`
	SSHHost           string `json:"ssh_host"`
	SSHPort           int    `json:"ssh_port"`
	MacIP             string `json:"mac_ip"`
	Phone             string `json:"phone"`
	GitHubToken       string `json:"github_token,omitempty"`
	SSHPassword       string `json:"ssh_password,omitempty"`
	AgentRepoURL      string `json:"agent_repo_url"`
	AgentRepoBranch   string `json:"agent_repo_branch"`
	AgentRepoPath     string `json:"agent_repo_path"`
	AgentArtifactName string `json:"agent_artifact_name"`
	AccessToken       string `json:"access_token,omitempty"`
	RefreshToken      string `json:"refresh_token,omitempty"`
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
	if cfg.AgentRepoURL == "" {
		cfg.AgentRepoURL = DefaultAgentRepoURL
	}
	if cfg.AgentRepoBranch == "" {
		cfg.AgentRepoBranch = DefaultAgentRepoBranch
	}
	if cfg.AgentArtifactName == "" {
		cfg.AgentArtifactName = DefaultAgentArtifactName
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
