package device

import (
	"fmt"

	"bell-provisioner/internal/agentrelease"

	"golang.org/x/crypto/ssh"
)

// AgentInstallOptions configures agent binary deploy.
type AgentInstallOptions struct {
	Enabled bool
	Bundle  *agentrelease.Bundle
	// BinaryName and ServiceName default from profile when empty.
	BinaryName  string
	ServiceName string
}

// DeployAgentBundle installs an agent binary + systemd unit from a bundle.
func DeployAgentBundle(client *ssh.Client, sudoPassword string, opts AgentInstallOptions) error {
	bundle := opts.Bundle
	if bundle == nil || len(bundle.Binary) == 0 {
		return fmt.Errorf("agent bundle is empty")
	}
	if len(bundle.ServiceUnit) == 0 {
		return fmt.Errorf("agent bundle missing systemd unit")
	}
	binaryName := opts.BinaryName
	if binaryName == "" {
		binaryName = "doorbell-agent"
	}
	serviceName := opts.ServiceName
	if serviceName == "" {
		serviceName = binaryName
	}

	tmpBin := "/tmp/" + binaryName
	tmpUnit := "/tmp/" + serviceName + ".service"
	if err := uploadFile(client, tmpBin, bundle.Binary, 0755); err != nil {
		return fmt.Errorf("upload agent binary: %w", err)
	}
	if err := uploadFile(client, tmpUnit, bundle.ServiceUnit, 0644); err != nil {
		return fmt.Errorf("upload systemd unit: %w", err)
	}

	installScript := fmt.Sprintf(`
sudo mkdir -p /usr/local/bin /etc/systemd/system
sudo install -m 755 %s /usr/local/bin/%s
sudo install -m 644 %s /etc/systemd/system/%s.service
rm -f %s %s
sudo systemctl daemon-reload
sudo systemctl enable %s 2>/dev/null || true
echo "==> Installed %s"
`,
		shellSingleQuote(tmpBin), binaryName,
		shellSingleQuote(tmpUnit), serviceName,
		shellSingleQuote(tmpBin), shellSingleQuote(tmpUnit),
		serviceName, binaryName,
	)
	if err := runSudo(client, sudoPassword, installScript); err != nil {
		return fmt.Errorf("install agent on device: %w", err)
	}
	return nil
}
