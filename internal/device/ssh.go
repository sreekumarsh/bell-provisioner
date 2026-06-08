package device

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
)

// SSHConfig holds connection parameters for the Pi (password auth only).
type SSHConfig struct {
	Host     string
	Port     int
	User     string
	Password string
}

// InstallResult reports post-install checks.
type InstallResult struct {
	AgentActive    bool   `json:"agent_active"`
	AgentChecked   bool   `json:"agent_checked_out"`
	FFmpegOK       bool   `json:"ffmpeg_ok"`
	Go2rtcOK       bool   `json:"go2rtc_ok"`
	MotionOK       bool   `json:"motion_ok"`
	SetupServerOK  bool   `json:"setup_server_ok"`
	Message        string `json:"message"`
}

// InstallCredentials copies device.key and identity.json to the Pi and restarts the agent.
func InstallCredentials(cfg SSHConfig, privateKeyPEM, identityJSON []byte, agentEnvContent string, deployAgentEnv bool, agentOpts AgentInstallOptions) (*InstallResult, error) {
	client, err := dialSSH(cfg)
	if err != nil {
		return nil, err
	}
	defer client.Close()

	expectedDeviceID := parseIdentityDeviceID(identityJSON)

	if err := installAgentRuntimeDeps(client, cfg.Password); err != nil {
		return nil, err
	}

	agentDeployed := false
	if agentOpts.Enabled {
		if err := DeployAgentBundle(client, cfg.Password, agentOpts.Bundle); err != nil {
			return nil, fmt.Errorf("deploy agent: %w", err)
		}
		agentDeployed = true
	}

	if err := runSudo(client, cfg.Password, "sudo mkdir -p /etc/doorbell"); err != nil {
		return nil, fmt.Errorf("mkdir: %w", err)
	}

	if err := uploadFile(client, "/tmp/device.key", privateKeyPEM, 0600); err != nil {
		return nil, fmt.Errorf("upload device.key: %w", err)
	}
	if err := uploadFile(client, "/tmp/identity.json", identityJSON, 0644); err != nil {
		return nil, fmt.Errorf("upload identity.json: %w", err)
	}

	installScript := `
sudo mkdir -p /etc/doorbell
sudo install -m 600 -o root -g root /tmp/device.key /etc/doorbell/device.key &&
sudo install -m 644 -o root -g root /tmp/identity.json /etc/doorbell/identity.json &&
rm -f /tmp/device.key /tmp/identity.json
`
	if err := runSudo(client, cfg.Password, installScript); err != nil {
		return nil, fmt.Errorf("install files: %w", err)
	}

	if deployAgentEnv && agentEnvContent != "" {
		if err := uploadFile(client, "/tmp/agent.env", []byte(agentEnvContent), 0644); err != nil {
			return nil, fmt.Errorf("upload agent.env: %w", err)
		}
		if err := runSudo(client, cfg.Password, "sudo install -m 644 -o root -g root /tmp/agent.env /etc/doorbell/agent.env && rm -f /tmp/agent.env"); err != nil {
			return nil, fmt.Errorf("install agent.env: %w", err)
		}
	}

	if err := runSudo(client, cfg.Password, "sudo systemctl daemon-reload && (sudo systemctl restart doorbell-agent || sudo systemctl restart doorbell-agent.service)"); err != nil {
		return nil, fmt.Errorf("restart agent: %w", err)
	}

	time.Sleep(2 * time.Second)
	ffmpegOK, go2rtcOK, motionOK := verifyAgentRuntimeDeps(client)
	active, err := isAgentActive(client)
	setupOK := waitForSetupServer(cfg.Host, expectedDeviceID, 45*time.Second)
	if err != nil {
		return &InstallResult{
			AgentActive: false, FFmpegOK: ffmpegOK, Go2rtcOK: go2rtcOK, MotionOK: motionOK,
			SetupServerOK: setupOK, Message: err.Error(),
		}, nil
	}

	msg := "Credentials installed — device in setup mode (claim via QR / :4444)"
	if agentDeployed {
		msg = "Agent installed, credentials deployed — device in setup mode (claim via QR / :4444)"
	}
	if !ffmpegOK || !go2rtcOK || !motionOK {
		var missing []string
		if !ffmpegOK {
			missing = append(missing, "ffmpeg")
		}
		if !go2rtcOK {
			missing = append(missing, "go2rtc")
		}
		if !motionOK {
			missing = append(missing, "motion (venv/model/script)")
		}
		msg = fmt.Sprintf("Install finished but missing runtime deps: %s — check apt/network on the Pi", strings.Join(missing, ", "))
	} else if !active {
		msg = "Install finished but doorbell-agent is not active — check journalctl on the Pi"
		if agentDeployed {
			msg = "Agent installed from CI but service is not active — check journalctl on the Pi"
		}
	} else if !setupOK {
		msg = "Agent is running but setup server (:4444) is not responding — ensure identity.json has claimed:false and agent restarted"
	}
	return &InstallResult{
		AgentActive: active, AgentChecked: agentDeployed,
		FFmpegOK: ffmpegOK, Go2rtcOK: go2rtcOK, MotionOK: motionOK, SetupServerOK: setupOK,
		Message: msg,
	}, nil
}

func parseIdentityDeviceID(identityJSON []byte) string {
	var id struct {
		DeviceID string `json:"device_id"`
	}
	if err := json.Unmarshal(identityJSON, &id); err != nil {
		return ""
	}
	return id.DeviceID
}

func dialSSH(cfg SSHConfig) (*ssh.Client, error) {
	port := cfg.Port
	if port == 0 {
		port = 22
	}

	password := strings.TrimSpace(cfg.Password)
	if password == "" {
		return nil, fmt.Errorf("SSH password is required — set it on the Environment step")
	}
	authMethods := []ssh.AuthMethod{
		ssh.Password(password),
		sshKeyboardInteractive(password),
	}

	clientConfig := &ssh.ClientConfig{
		User:            cfg.User,
		Auth:            authMethods,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), //nolint:gosec // dev/lab Pi on LAN
		Timeout:         15 * time.Second,
	}

	addr := net.JoinHostPort(cfg.Host, fmt.Sprintf("%d", port))
	if err := tcpReachable(addr, 5*time.Second); err != nil {
		return nil, err
	}
	client, err := ssh.Dial("tcp", addr, clientConfig)
	if err != nil {
		if strings.Contains(err.Error(), "unable to authenticate") {
			return nil, fmt.Errorf("%w — wrong Pi SSH user or password (user %q must be the Pi account, e.g. pi — not your Mac login)", err, cfg.User)
		}
		return nil, err
	}
	return client, nil
}

func tcpReachable(addr string, timeout time.Duration) error {
	conn, err := net.DialTimeout("tcp", addr, timeout)
	if err == nil {
		_ = conn.Close()
		return nil
	}
	if strings.Contains(err.Error(), "no route to host") {
		host, _, _ := net.SplitHostPort(addr)
		return fmt.Errorf("cannot reach %s — Mac and Pi must be on the same LAN (check Pi IP, Wi‑Fi, and VPN). From Terminal: ping %s", addr, host)
	}
	if strings.Contains(err.Error(), "connection refused") {
		return fmt.Errorf("SSH port closed on %s — enable SSH on the Pi (raspi-config or sudo systemctl start ssh)", addr)
	}
	if strings.Contains(err.Error(), "i/o timeout") || strings.Contains(err.Error(), "timeout") {
		return fmt.Errorf("timed out reaching %s — verify IP, same subnet, and that the Pi is powered on", addr)
	}
	return fmt.Errorf("network error reaching %s: %w", addr, err)
}

// sshKeyboardInteractive supports sshd configs that use PAM / KbdInteractive instead of password auth.
func sshKeyboardInteractive(password string) ssh.AuthMethod {
	return ssh.KeyboardInteractive(func(_ string, _ string, questions []string, _ []bool) ([]string, error) {
		answers := make([]string, len(questions))
		for i := range questions {
			answers[i] = password
		}
		return answers, nil
	})
}

func uploadFile(client *ssh.Client, remotePath string, content []byte, mode uint32) error {
	session, err := client.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()

	stdin, err := session.StdinPipe()
	if err != nil {
		return err
	}

	var stderr bytes.Buffer
	session.Stderr = &stderr

	cmd := fmt.Sprintf("cat > %s", shellSingleQuote(remotePath))
	if err := session.Start(cmd); err != nil {
		return err
	}

	if _, err := stdin.Write(content); err != nil {
		_ = session.Close()
		return fmt.Errorf("write upload stream: %w", err)
	}
	if err := stdin.Close(); err != nil {
		return fmt.Errorf("close upload stream: %w", err)
	}

	if err := session.Wait(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg != "" {
			return fmt.Errorf("%w: %s", err, msg)
		}
		return err
	}

	if mode != 0 {
		if err := runCmd(client, fmt.Sprintf("chmod %04o %s", mode, shellSingleQuote(remotePath))); err != nil {
			return fmt.Errorf("chmod %s: %w", remotePath, err)
		}
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
