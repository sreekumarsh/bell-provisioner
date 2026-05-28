package main

import (
	"context"
	"fmt"
	"sync"
	"time"

	appcfg "bell-provisioner/internal/config"
	"bell-provisioner/internal/device"
	"bell-provisioner/internal/discover"
	"bell-provisioner/internal/gateway"
	"bell-provisioner/internal/provision"
	"bell-provisioner/internal/verify"
)

// App is the Wails-bound backend.
type App struct {
	ctx context.Context

	mu sync.Mutex

	cfg          appcfg.AppConfig
	accessToken  string
	refreshToken string
	userName     string
	userRole     string

	pendingPrivateKey []byte
	pendingIdentity   []byte
	pendingDeviceID   string
	pendingMQTTUser   string
	pendingMQTTPass   string
}

// NewApp creates a new App application struct.
func NewApp() *App {
	return &App{cfg: appcfg.Load()}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

// GetConfig returns persisted settings.
func (a *App) GetConfig() appcfg.AppConfig {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.cfg
}

// SaveConfig persists environment settings.
func (a *App) SaveConfig(cfg appcfg.AppConfig) error {
	a.mu.Lock()
	a.cfg = cfg
	a.mu.Unlock()
	return appcfg.Save(cfg)
}

// LoginResult is returned after admin login.
type LoginResult struct {
	UserID   string `json:"user_id"`
	Name     string `json:"name"`
	Phone    string `json:"phone"`
	Role     string `json:"role"`
	IsAdmin  bool   `json:"is_admin"`
}

// Login authenticates against the API gateway.
func (a *App) Login(phone, password string) (*LoginResult, error) {
	a.mu.Lock()
	gatewayURL := a.cfg.GatewayURL
	a.mu.Unlock()

	client := gateway.NewClient(gatewayURL)
	resp, err := client.Login(phone, password)
	if err != nil {
		return nil, err
	}

	isAdmin := resp.User.Role == "admin"
	if !isAdmin {
		return nil, fmt.Errorf("account lacks admin role — set role=admin in auth.users or add user to gateway ADMIN_USER_IDS")
	}

	a.mu.Lock()
	a.accessToken = resp.AccessToken
	a.refreshToken = resp.RefreshToken
	a.userName = resp.User.Name
	a.userRole = resp.User.Role
	a.cfg.Phone = phone
	_ = appcfg.Save(a.cfg)
	a.mu.Unlock()

	return &LoginResult{
		UserID:  resp.User.UserID,
		Name:    resp.User.Name,
		Phone:   resp.User.Phone,
		Role:    resp.User.Role,
		IsAdmin: true,
	}, nil
}

// DiscoverDevices scans the LAN for Pi hosts.
func (a *App) DiscoverDevices(manualHost string) []discover.Candidate {
	a.mu.Lock()
	hosts := []string{}
	if manualHost != "" {
		hosts = append(hosts, manualHost)
	}
	if a.cfg.SSHHost != "" {
		hosts = append(hosts, a.cfg.SSHHost)
	}
	a.mu.Unlock()

	ctx, cancel := context.WithTimeout(a.ctx, 45*time.Second)
	defer cancel()

	return discover.Scan(ctx, discover.ScanOptions{
		ManualHosts: hosts,
		Timeout:     2 * time.Second,
	})
}

// ProvisionRequest holds serial/hw for cloud registration.
type ProvisionRequest struct {
	Serial     string `json:"serial"`
	HWVersion  string `json:"hw_version"`
}

// ProvisionResult holds cloud registration outcome (no private key exposed).
type ProvisionResult struct {
	DeviceID     string `json:"device_id"`
	SerialNumber string `json:"serial_number"`
	MQTTUsername string `json:"mqtt_username"`
	MQTTPassword string `json:"mqtt_password"`
}

// Provision generates keys and registers the device via gateway admin API.
func (a *App) Provision(req ProvisionRequest) (*ProvisionResult, error) {
	if req.Serial == "" {
		return nil, fmt.Errorf("serial number is required")
	}
	hw := req.HWVersion
	if hw == "" {
		hw = "1.0"
	}

	a.mu.Lock()
	token := a.accessToken
	gatewayURL := a.cfg.GatewayURL
	a.mu.Unlock()
	if token == "" {
		return nil, fmt.Errorf("login required")
	}

	keypair, err := provision.GenerateRSA2048()
	if err != nil {
		return nil, err
	}

	result, err := provision.RegisterDevice(gatewayURL, token, req.Serial, hw, keypair.PublicKeyPEM)
	if err != nil {
		return nil, err
	}

	identityJSON, err := device.BuildIdentityJSON(
		result.DeviceID, req.Serial, hw, result.MQTTUsername, result.MQTTPassword,
	)
	if err != nil {
		return nil, err
	}

	a.mu.Lock()
	a.pendingPrivateKey = keypair.PrivateKeyPEM
	a.pendingIdentity = identityJSON
	a.pendingDeviceID = result.DeviceID
	a.pendingMQTTUser = result.MQTTUsername
	a.pendingMQTTPass = result.MQTTPassword
	a.mu.Unlock()

	return &ProvisionResult{
		DeviceID:     result.DeviceID,
		SerialNumber: req.Serial,
		MQTTUsername: result.MQTTUsername,
		MQTTPassword: result.MQTTPassword,
	}, nil
}

// InstallRequest configures SSH install step.
type InstallRequest struct {
	Host            string `json:"host"`
	SSHUser         string `json:"ssh_user"`
	SSHPort         int    `json:"ssh_port"`
	SSHPassword     string `json:"ssh_password"`
	DeployAgentEnv  bool   `json:"deploy_agent_env"`
}

// Install pushes credentials to the Pi over SSH.
func (a *App) Install(req InstallRequest) (*device.InstallResult, error) {
	a.mu.Lock()
	priv := append([]byte(nil), a.pendingPrivateKey...)
	identity := append([]byte(nil), a.pendingIdentity...)
	cfg := a.cfg
	a.mu.Unlock()

	if len(priv) == 0 || len(identity) == 0 {
		return nil, fmt.Errorf("provision the device first")
	}

	host := req.Host
	if host == "" {
		host = cfg.SSHHost
	}
	resolved, _ := discover.ResolveHost(host)

	sshUser := req.SSHUser
	if sshUser == "" {
		sshUser = cfg.SSHUser
	}
	port := req.SSHPort
	if port == 0 {
		port = cfg.SSHPort
	}

	agentEnv := ""
	if req.DeployAgentEnv {
		agentEnv = appcfg.AgentEnv(appcfg.BackendProfile(cfg.BackendProfile), cfg.MacIP)
	}

	return device.InstallCredentials(device.SSHConfig{
		Host:     resolved,
		Port:     port,
		User:     sshUser,
		Password: req.SSHPassword,
	}, priv, identity, agentEnv, req.DeployAgentEnv)
}

// VerifyRequest configures post-install checks.
type VerifyRequest struct {
	DeviceID     string `json:"device_id"`
	MQTTUsername string `json:"mqtt_username"`
	MQTTPassword string `json:"mqtt_password"`
}

// VerifyResult combines SSH and MQTT checks.
type VerifyResult struct {
	MQTT      verify.Result         `json:"mqtt"`
	Install   *device.InstallResult `json:"install,omitempty"`
}

// Verify runs MQTT smoke test after install.
func (a *App) Verify(req VerifyRequest) *VerifyResult {
	a.mu.Lock()
	deviceID := req.DeviceID
	if deviceID == "" {
		deviceID = a.pendingDeviceID
	}
	mqttUser := req.MQTTUsername
	if mqttUser == "" {
		mqttUser = a.pendingMQTTUser
	}
	mqttPass := req.MQTTPassword
	if mqttPass == "" {
		mqttPass = a.pendingMQTTPass
	}
	cfg := a.cfg
	a.mu.Unlock()

	broker := appcfg.MQTTBrokerURL(appcfg.BackendProfile(cfg.BackendProfile), cfg.MacIP)
	mqttResult := verify.TestMQTTConnection(broker, mqttUser, mqttPass, deviceID)

	return &VerifyResult{MQTT: *mqttResult}
}

// TestSSH checks SSH connectivity to the selected host.
func (a *App) TestSSH(host, user string, port int, password string) error {
	a.mu.Lock()
	cfg := a.cfg
	a.mu.Unlock()

	if host == "" {
		host = cfg.SSHHost
	}
	if user == "" {
		user = cfg.SSHUser
	}
	if port == 0 {
		port = cfg.SSHPort
	}
	resolved, _ := discover.ResolveHost(host)

	return device.TestSSH(device.SSHConfig{
		Host:     resolved,
		Port:     port,
		User:     user,
		Password: password,
	})
}
