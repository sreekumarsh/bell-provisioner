package config

// Default agent sources (private repos; use a read-only GitHub PAT in the app).
const (
	DefaultAgentRepoURL      = "git@github.com:sreekumarsh/pi-streamer.git"
	DefaultAgentRepoBranch   = "main"
	DefaultAgentArtifactName = "doorbell-agent-linux-arm64"

	DefaultNvrAgentRepoURL    = "git@github.com:sreekumarsh/vyooham-nvr.git"
	DefaultNvrAgentRepoBranch = "main"
)
