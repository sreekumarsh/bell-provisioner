package config

// Default agent sources (private repos under the vyooham org; use a read-only GitHub PAT).
const (
	DefaultAgentRepoURL      = "git@github.com:vyooham/pi-streamer.git"
	DefaultAgentRepoBranch   = "main"
	DefaultAgentArtifactName = "doorbell-agent-linux-arm64"

	DefaultNvrAgentRepoURL    = "git@github.com:vyooham/vyooham-nvr.git"
	DefaultNvrAgentRepoBranch = "main"

	DefaultSenseAgentRepoURL    = "git@github.com:vyooham/vyooham-sense.git"
	DefaultSenseAgentRepoBranch = "main"
)
