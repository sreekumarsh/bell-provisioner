package device

import (
	"bytes"
	"fmt"
	"net"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
)

// SSHConfig holds connection parameters for the device (password auth only).
type SSHConfig struct {
	Host     string
	Port     int
	User     string
	Password string
}

// InstallResult reports post-install checks.
type InstallResult struct {
	Profile        string `json:"profile"`
	AgentActive    bool   `json:"agent_active"`
	AgentChecked   bool   `json:"agent_checked_out"`
	FFmpegOK       bool   `json:"ffmpeg_ok"`
	Go2rtcOK       bool   `json:"go2rtc_ok"`
	MotionOK       bool   `json:"motion_ok"`
	SetupServerOK  bool   `json:"setup_server_ok"`
	Message        string `json:"message"`
}

// InstallCredentials copies device.key, identity.json, and optional mTLS certs
// to the device etc dir for the given profile and restarts the agent.
func InstallCredentials(cfg SSHConfig, profile InstallProfile, privateKeyPEM, identityJSON, deviceCertPEM, caCertPEM []byte, agentEnvContent string, deployAgentEnv bool, agentOpts AgentInstallOptions) (*InstallResult, error) {
	spec := profile.Spec()
	client, err := dialSSH(cfg)
	if err != nil {
		return nil, err
	}
	defer client.Close()

	id, err := ParseIdentity(identityJSON)
	if err != nil {
		return nil, err
	}
	expectedDeviceID := id.DeviceID

	if spec.CameraDeps {
		if err := installDoorbellRuntimeDeps(client, cfg.Password); err != nil {
			return nil, err
		}
	}

	agentDeployed := false
	if agentOpts.Enabled {
		opts := agentOpts
		if opts.BinaryName == "" {
			opts.BinaryName = spec.BinaryName
		}
		if opts.ServiceName == "" {
			opts.ServiceName = spec.ServiceName
		}
		if err := DeployAgentBundle(client, cfg.Password, opts); err != nil {
			return nil, fmt.Errorf("deploy agent: %w", err)
		}
		agentDeployed = true
	}

	etcDir := spec.EtcDir
	if err := runSudo(client, cfg.Password, "sudo mkdir -p "+shellSingleQuote(etcDir)); err != nil {
		return nil, fmt.Errorf("mkdir: %w", err)
	}

	if err := uploadFile(client, "/tmp/device.key", privateKeyPEM, 0600); err != nil {
		return nil, fmt.Errorf("upload device.key: %w", err)
	}
	if err := uploadFile(client, "/tmp/identity.json", identityJSON, 0644); err != nil {
		return nil, fmt.Errorf("upload identity.json: %w", err)
	}

	installScript := fmt.Sprintf(`
sudo mkdir -p %s
sudo install -m 600 -o root -g root /tmp/device.key %s/device.key &&
sudo install -m 644 -o root -g root /tmp/identity.json %s/identity.json &&
rm -f /tmp/device.key /tmp/identity.json
`, shellSingleQuote(etcDir), shellSingleQuote(etcDir), shellSingleQuote(etcDir))
	if len(deviceCertPEM) > 0 {
		if err := uploadFile(client, "/tmp/device.crt", deviceCertPEM, 0644); err != nil {
			return nil, fmt.Errorf("upload device.crt: %w", err)
		}
		installScript += fmt.Sprintf(`
sudo install -m 644 -o root -g root /tmp/device.crt %s/device.crt &&
rm -f /tmp/device.crt
`, shellSingleQuote(etcDir))
	}
	if len(caCertPEM) > 0 {
		if err := uploadFile(client, "/tmp/ca.crt", caCertPEM, 0644); err != nil {
			return nil, fmt.Errorf("upload ca.crt: %w", err)
		}
		installScript += fmt.Sprintf(`
sudo install -m 644 -o root -g root /tmp/ca.crt %s/ca.crt &&
rm -f /tmp/ca.crt
`, shellSingleQuote(etcDir))
	}

	if err := runSudo(client, cfg.Password, installScript); err != nil {
		return nil, fmt.Errorf("install files: %w", err)
	}

	if deployAgentEnv && agentEnvContent != "" {
		envRemote := etcDir + "/" + spec.EnvFileName
		if err := uploadFile(client, "/tmp/agent.env", []byte(agentEnvContent), 0644); err != nil {
			return nil, fmt.Errorf("upload agent.env: %w", err)
		}
		if err := runSudo(client, cfg.Password, fmt.Sprintf(
			"sudo install -m 644 -o root -g root /tmp/agent.env %s && rm -f /tmp/agent.env",
			shellSingleQuote(envRemote),
		)); err != nil {
			return nil, fmt.Errorf("install agent.env: %w", err)
		}
	}

	svc := spec.ServiceName
	restartCmd := fmt.Sprintf(
		"sudo systemctl daemon-reload && (sudo systemctl restart %s || sudo systemctl restart %s.service)",
		svc, svc,
	)
	if err := runSudo(client, cfg.Password, restartCmd); err != nil {
		return nil, fmt.Errorf("restart agent: %w", err)
	}

	time.Sleep(2 * time.Second)

	var ffmpegOK, go2rtcOK, motionOK bool
	if spec.CameraDeps {
		ffmpegOK, go2rtcOK, motionOK = verifyDoorbellRuntimeDeps(client)
	} else {
		ffmpegOK, go2rtcOK, motionOK = true, true, true
	}

	active, err := isServiceActive(client, svc)
	setupOK := waitForSetupServer(cfg.Host, expectedDeviceID, 45*time.Second)
	if err != nil {
		return &InstallResult{
			Profile: string(profile), AgentActive: false,
			FFmpegOK: ffmpegOK, Go2rtcOK: go2rtcOK, MotionOK: motionOK,
			SetupServerOK: setupOK, Message: err.Error(),
		}, nil
	}

	msg := fmt.Sprintf("Credentials installed to %s — device in setup mode (claim via QR / :4444)", etcDir)
	if agentDeployed {
		msg = fmt.Sprintf("%s installed, credentials in %s — device in setup mode (claim via QR / :4444)", spec.BinaryName, etcDir)
	}
	if spec.CameraDeps && (!ffmpegOK || !go2rtcOK || !motionOK) {
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
		msg = fmt.Sprintf("Install finished but missing runtime deps: %s — check apt/network on the device", strings.Join(missing, ", "))
	} else if !active {
		msg = fmt.Sprintf("Install finished but %s is not active — check journalctl on the device", svc)
		if agentDeployed {
			msg = fmt.Sprintf("%s installed but service is not active — check journalctl on the device", spec.BinaryName)
		}
	} else if !setupOK {
		msg = fmt.Sprintf("%s is running but setup server (:4444) is not responding — ensure identity.json has claimed:false and agent restarted", svc)
	}
	return &InstallResult{
		Profile: string(profile), AgentActive: active, AgentChecked: agentDeployed,
		FFmpegOK: ffmpegOK, Go2rtcOK: go2rtcOK, MotionOK: motionOK, SetupServerOK: setupOK,
		Message: msg,
	}, nil
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
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), //nolint:gosec // lab device on LAN
		Timeout:         15 * time.Second,
	}

	addr := net.JoinHostPort(cfg.Host, fmt.Sprintf("%d", port))
	if err := tcpReachable(addr, 5*time.Second); err != nil {
		return nil, err
	}
	client, err := ssh.Dial("tcp", addr, clientConfig)
	if err != nil {
		if strings.Contains(err.Error(), "no supported methods remain") {
			return nil, fmt.Errorf("%w — SSH accepts public keys only; enable password auth (sshd PasswordAuthentication yes, then sudo systemctl restart ssh). Bell Provisioner uses password auth, not your Mac SSH keys", err)
		}
		if strings.Contains(err.Error(), "unable to authenticate") {
			return nil, fmt.Errorf("%w — check SSH user %q and password (Environment step)", err, cfg.User)
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
		return fmt.Errorf("cannot reach %s — Mac and device must be on the same LAN (check IP, Wi‑Fi, and VPN). From Terminal: ping %s", addr, host)
	}
	if strings.Contains(err.Error(), "connection refused") {
		return fmt.Errorf("SSH port closed on %s — enable SSH on the device", addr)
	}
	if strings.Contains(err.Error(), "i/o timeout") || strings.Contains(err.Error(), "timeout") {
		return fmt.Errorf("timed out reaching %s — verify IP, same subnet, and that the device is powered on", addr)
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

func isServiceActive(client *ssh.Client, serviceName string) (bool, error) {
	session, err := client.NewSession()
	if err != nil {
		return false, err
	}
	defer session.Close()

	cmd := fmt.Sprintf("systemctl is-active %s 2>/dev/null || systemctl is-active %s.service 2>/dev/null",
		shellSingleQuote(serviceName), shellSingleQuote(serviceName))
	out, err := session.Output(cmd)
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
