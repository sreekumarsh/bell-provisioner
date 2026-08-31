package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
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
	// SenseMQTTTransport is "plain" or "mtls"; empty means DefaultSenseTransport (mtls).
	SenseMQTTTransport string `json:"sense_mqtt_transport,omitempty"`
	// SenseClaimGrantPubPath is a local PEM file holding the public half of
	// bell-auth-service's claim-grant signing key — the key its CLAIM_GRANT_KID
	// names. It is configured rather than fetched because auth-service keeps
	// that key in KMS and publishes no public half over its API, so getting it
	// to the bench is an out-of-band operator step. Required for Sense: install
	// aborts without it, since control-agent will not serve the claim flow with
	// no verify key.
	SenseClaimGrantPubPath string `json:"sense_claim_grant_pub_path,omitempty"`

	// UARTPort is the last-used serial device path for Sense UART provisioning
	// (e.g. /dev/cu.usbserial-310). Password is session-only and not stored here.
	UARTPort string `json:"uart_port,omitempty"`

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
	if cfg.AgentRepoBranch == "" {
		cfg.AgentRepoBranch = DefaultAgentRepoBranch
	}
	if cfg.AgentArtifactName == "" {
		cfg.AgentArtifactName = DefaultAgentArtifactName
	}
	if cfg.NvrAgentRepoBranch == "" {
		cfg.NvrAgentRepoBranch = DefaultNvrAgentRepoBranch
	}
	if cfg.SenseAgentRepoBranch == "" {
		cfg.SenseAgentRepoBranch = DefaultSenseAgentRepoBranch
	}
	NormalizeRepos(&cfg)
	return cfg
}

// NormalizeRepos fills empty agent-repo fields and rewrites known-stale
// sreekumarsh/* remotes to the vyooham org (vyooham-sense never existed under
// sreekumarsh — GitHub API 404).
func NormalizeRepos(cfg *AppConfig) {
	if cfg == nil {
		return
	}
	cfg.AgentRepoURL = migrateAgentRepoURL(cfg.AgentRepoURL, "pi-streamer", DefaultAgentRepoURL)
	cfg.NvrAgentRepoURL = migrateAgentRepoURL(cfg.NvrAgentRepoURL, "vyooham-nvr", DefaultNvrAgentRepoURL)
	cfg.SenseAgentRepoURL = migrateAgentRepoURL(cfg.SenseAgentRepoURL, "vyooham-sense", DefaultSenseAgentRepoURL)
}

// migrateAgentRepoURL fills empty URLs and rewrites known-stale sreekumarsh
// remotes to the vyooham org defaults.
func migrateAgentRepoURL(url, repoName, defaultURL string) string {
	if strings.TrimSpace(url) == "" {
		return defaultURL
	}
	owner, repo, err := GitHubRepo(url)
	if err != nil {
		return url
	}
	if repo == repoName && owner == "sreekumarsh" {
		return defaultURL
	}
	return url
}

// Save persists config to disk.
func Save(cfg AppConfig) error {
	NormalizeRepos(&cfg)
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
