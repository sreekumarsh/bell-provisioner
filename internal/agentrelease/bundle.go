package agentrelease

// Bundle is a Pi arm64 agent build from pi-streamer CI (Actions artifact).
type Bundle struct {
	Version     string
	Binary      []byte
	ServiceUnit []byte
}
