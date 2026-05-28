package device

import (
	"bytes"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
)

// SSHConfig holds connection parameters for the Pi.
type SSHConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	KeyPath  string
}

// InstallResult reports post-install checks.
type InstallResult struct {
	AgentActive bool   `json:"agent_active"`
	Message     string `json:"message"`
}

// InstallCredentials copies device.key and identity.json to the Pi and restarts the agent.
func InstallCredentials(cfg SSHConfig, privateKeyPEM, identityJSON []byte, agentEnvContent string, deployAgentEnv bool) (*InstallResult, error) {
	client, err := dialSSH(cfg)
	if err != nil {
		return nil, err
	}
	defer client.Close()

	if err := runCmd(client, "sudo mkdir -p /etc/doorbell"); err != nil {
		return nil, fmt.Errorf("mkdir: %w", err)
	}

	if err := uploadFile(client, "/tmp/device.key", privateKeyPEM, 0600); err != nil {
		return nil, fmt.Errorf("upload device.key: %w", err)
	}
	if err := uploadFile(client, "/tmp/identity.json", identityJSON, 0644); err != nil {
		return nil, fmt.Errorf("upload identity.json: %w", err)
	}

	installScript := `
sudo install -m 600 -o root -g root /tmp/device.key /etc/doorbell/device.key &&
sudo install -m 644 -o root -g root /tmp/identity.json /etc/doorbell/identity.json &&
rm -f /tmp/device.key /tmp/identity.json
`
	if err := runCmd(client, installScript); err != nil {
		return nil, fmt.Errorf("install files: %w", err)
	}

	if deployAgentEnv && agentEnvContent != "" {
		if err := uploadFile(client, "/tmp/agent.env", []byte(agentEnvContent), 0644); err != nil {
			return nil, fmt.Errorf("upload agent.env: %w", err)
		}
		if err := runCmd(client, "sudo install -m 644 -o root -g root /tmp/agent.env /etc/doorbell/agent.env && rm -f /tmp/agent.env"); err != nil {
			return nil, fmt.Errorf("install agent.env: %w", err)
		}
	}

	if err := runCmd(client, "sudo systemctl restart doorbell-agent || sudo systemctl restart doorbell-agent.service"); err != nil {
		return nil, fmt.Errorf("restart agent: %w", err)
	}

	time.Sleep(2 * time.Second)
	active, err := isAgentActive(client)
	if err != nil {
		return &InstallResult{AgentActive: false, Message: err.Error()}, nil
	}

	msg := "Credentials installed"
	if !active {
		msg = "Credentials installed but doorbell-agent is not active — check journalctl on the Pi"
	}
	return &InstallResult{AgentActive: active, Message: msg}, nil
}

func dialSSH(cfg SSHConfig) (*ssh.Client, error) {
	port := cfg.Port
	if port == 0 {
		port = 22
	}

	var authMethods []ssh.AuthMethod
	if cfg.Password != "" {
		authMethods = append(authMethods, ssh.Password(cfg.Password))
	}
	keyPath := cfg.KeyPath
	if keyPath == "" {
		if home, err := os.UserHomeDir(); err == nil {
			for _, name := range []string{"id_ed25519", "id_rsa"} {
				p := filepath.Join(home, ".ssh", name)
				if _, err := os.Stat(p); err == nil {
					keyPath = p
					break
				}
			}
		}
	}
	if keyPath != "" {
		key, err := loadPrivateKey(keyPath)
		if err != nil {
			return nil, fmt.Errorf("load ssh key %s: %w", keyPath, err)
		}
		authMethods = append(authMethods, ssh.PublicKeys(key))
	}
	if len(authMethods) == 0 {
		return nil, fmt.Errorf("no SSH auth method: set password or add ~/.ssh/id_ed25519")
	}

	clientConfig := &ssh.ClientConfig{
		User:            cfg.User,
		Auth:            authMethods,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), //nolint:gosec // dev/lab Pi on LAN
		Timeout:         10 * time.Second,
	}

	addr := net.JoinHostPort(cfg.Host, fmt.Sprintf("%d", port))
	return ssh.Dial("tcp", addr, clientConfig)
}

func loadPrivateKey(path string) (ssh.Signer, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	signer, err := ssh.ParsePrivateKey(data)
	if err != nil {
		return nil, err
	}
	return signer, nil
}

func uploadFile(client *ssh.Client, remotePath string, content []byte, mode os.FileMode) error {
	sftp, err := client.NewSession()
	if err != nil {
		return err
	}
	defer sftp.Close()

	go func() {
		w, _ := sftp.StdinPipe()
		defer w.Close()
		_, _ = fmt.Fprintf(w, "C%04o %d %s\n", mode, len(content), remotePath)
		_, _ = w.Write(content)
		_, _ = fmt.Fprint(w, "\x00")
	}()

	if err := sftp.Run("scp -t " + remotePath); err != nil {
		return err
	}
	return nil
}

func runCmd(client *ssh.Client, script string) error {
	session, err := client.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()

	var stderr bytes.Buffer
	session.Stderr = &stderr
	if err := session.Run(script); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg != "" {
			return fmt.Errorf("%w: %s", err, msg)
		}
		return err
	}
	return nil
}

func isAgentActive(client *ssh.Client) (bool, error) {
	session, err := client.NewSession()
	if err != nil {
		return false, err
	}
	defer session.Close()

	out, err := session.Output("systemctl is-active doorbell-agent 2>/dev/null || systemctl is-active doorbell-agent.service 2>/dev/null")
	if err != nil {
		return false, nil
	}
	return strings.TrimSpace(string(out)) == "active", nil
}

func TestSSH(cfg SSHConfig) error {
	client, err := dialSSH(cfg)
	if err != nil {
		return err
	}
	defer client.Close()
	return runCmd(client, "echo ok")
}
