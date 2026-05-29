package device

import (
	"fmt"

	"bell-provisioner/internal/agentrelease"

	"golang.org/x/crypto/ssh"
)

// AgentInstallOptions configures CI artifact deploy to the Pi.
type AgentInstallOptions struct {
	Enabled bool
	Bundle  *agentrelease.Bundle
}

// DeployAgentBundle installs doorbell-agent from a CI artifact (binary + systemd unit).
func DeployAgentBundle(client *ssh.Client, sudoPassword string, bundle *agentrelease.Bundle) error {
	if bundle == nil || len(bundle.Binary) == 0 {
		return fmt.Errorf("agent bundle is empty")
	}
	if len(bundle.ServiceUnit) == 0 {
		return fmt.Errorf("agent bundle missing systemd unit")
	}

	if err := installStreamingDeps(client, sudoPassword); err != nil {
		return err
	}

	if err := uploadFile(client, "/tmp/doorbell-agent", bundle.Binary, 0755); err != nil {
		return fmt.Errorf("upload agent binary: %w", err)
	}
	if err := uploadFile(client, "/tmp/doorbell-agent.service", bundle.ServiceUnit, 0644); err != nil {
		return fmt.Errorf("upload systemd unit: %w", err)
	}

	installScript := `
sudo mkdir -p /usr/local/bin /etc/systemd/system
sudo install -m 755 /tmp/doorbell-agent /usr/local/bin/doorbell-agent
sudo install -m 644 /tmp/doorbell-agent.service /etc/systemd/system/doorbell-agent.service
rm -f /tmp/doorbell-agent /tmp/doorbell-agent.service
sudo systemctl daemon-reload
sudo systemctl enable doorbell-agent 2>/dev/null || true
echo "==> Installed doorbell-agent from CI artifact"
`
	if err := runSudo(client, sudoPassword, installScript); err != nil {
		return fmt.Errorf("install agent on Pi: %w", err)
	}
	return nil
}
