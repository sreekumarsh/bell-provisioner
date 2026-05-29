package config

// Default agent source (pi-streamer private repo; use a read-only GitHub PAT in the app or a Pi deploy key).
const (
	DefaultAgentRepoURL       = "git@github.com:sreekumarsh/pi-streamer.git"
	DefaultAgentRepoBranch    = "main"
	DefaultAgentArtifactName  = "doorbell-agent-linux-arm64"
)
