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
	NvrAgentRepoURL    string `json:"nvr_agent_repo_url"`
	NvrAgentRepoBranch string `json:"nvr_agent_repo_branch"`

	SenseAgentRepoURL    string `json:"sense_agent_repo_url"`
	SenseAgentRepoBranch string `json:"sense_agent_repo_branch"`
	// SenseMQTTTransport is "plain" or "mtls"; empty means DefaultSenseTransport.
	// Set to "mtls" only once mqtt.vyooham.com resolves and auth-service issues
	// device certs — see DefaultSenseTransport.
	SenseMQTTTransport string `json:"sense_mqtt_transport,omitempty"`
	// SenseClaimGrantPubPath is a local PEM file holding the public half of
	// bell-auth-service's claim-grant signing key — the key its CLAIM_GRANT_KID
	// names. It is configured rather than fetched because auth-service keeps
	// that key in KMS and publishes no public half over its API, so getting it
	// to the bench is an out-of-band operator step. Required for Sense: install
	// aborts without it, since control-agent will not serve the claim flow with
	// no verify key.
	SenseClaimGrantPubPath string `json:"sense_claim_grant_pub_path,omitempty"`

	AccessToken        string `json:"access_token,omitempty"`
	RefreshToken       string `json:"refresh_token,omitempty"`
	TokenExpiresAt     int64  `json:"token_expires_at,omitempty"`
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
	if cfg.NvrAgentRepoURL == "" {
		cfg.NvrAgentRepoURL = DefaultNvrAgentRepoURL
	}
	if cfg.NvrAgentRepoBranch == "" {
		cfg.NvrAgentRepoBranch = DefaultNvrAgentRepoBranch
	}
	if cfg.SenseAgentRepoURL == "" {
		cfg.SenseAgentRepoURL = DefaultSenseAgentRepoURL
	}
	if cfg.SenseAgentRepoBranch == "" {
		cfg.SenseAgentRepoBranch = DefaultSenseAgentRepoBranch
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
