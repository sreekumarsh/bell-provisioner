package main

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	appcfg "bell-provisioner/internal/config"
	"bell-provisioner/internal/agentrelease"
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
	cfg := appcfg.Load()
	return &App{
		cfg:          cfg,
		accessToken:  cfg.AccessToken,
		refreshToken: cfg.RefreshToken,
	}
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
	a.cfg.AccessToken = resp.AccessToken
	a.cfg.RefreshToken = resp.RefreshToken
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

// SessionInfo describes the current operator session (in-memory JWT).
type SessionInfo struct {
	LoggedIn   bool   `json:"logged_in"`
	UserName   string `json:"user_name"`
	Phone      string `json:"phone"`
	Role       string `json:"role"`
	GatewayURL string `json:"gateway_url"`
}

// GetSession returns whether an admin JWT is held in memory.
func (a *App) GetSession() SessionInfo {
	a.mu.Lock()
	defer a.mu.Unlock()
	return SessionInfo{
		LoggedIn:   a.accessToken != "",
		UserName:   a.userName,
		Phone:      a.cfg.Phone,
		Role:       a.userRole,
		GatewayURL: a.cfg.GatewayURL,
	}
}

// Logout clears the in-memory admin session.
func (a *App) Logout() {
	a.mu.Lock()
	a.accessToken = ""
	a.refreshToken = ""
	a.userName = ""
	a.userRole = ""
	a.cfg.AccessToken = ""
	a.cfg.RefreshToken = ""
	a.pendingPrivateKey = nil
	a.pendingIdentity = nil
	a.pendingDeviceID = ""
	a.pendingMQTTUser = ""
	a.pendingMQTTPass = ""
	_ = appcfg.Save(a.cfg)
	a.mu.Unlock()
}

func (a *App) adminClient() (*gateway.Client, error) {
	a.mu.Lock()
	token := a.accessToken
	gatewayURL := a.cfg.GatewayURL
	a.mu.Unlock()
	if token == "" {
		return nil, fmt.Errorf("login required")
	}
	client := gateway.NewClient(gatewayURL)
	client.AccessToken = token
	return client, nil
}

// ListCapabilities returns registry capabilities.
func (a *App) ListCapabilities() ([]gateway.Capability, error) {
	client, err := a.adminClient()
	if err != nil {
		return nil, err
	}
	return client.ListCapabilities()
}

// CreateCapability adds a capability via POST /admin/capabilities.
func (a *App) CreateCapability(cap gateway.Capability) (*gateway.Capability, error) {
	if strings.TrimSpace(cap.CapID) == "" {
		return nil, fmt.Errorf("capid is required")
	}
	if strings.TrimSpace(cap.FriendlyName) == "" {
		return nil, fmt.Errorf("friendly_name is required")
	}
	if cap.Layer != "intrinsic" && cap.Layer != "runtime" {
		return nil, fmt.Errorf("layer must be intrinsic or runtime")
	}
	client, err := a.adminClient()
	if err != nil {
		return nil, err
	}
	return client.CreateCapability(cap)
}

// ListDeviceFamilies returns registry families.
func (a *App) ListDeviceFamilies() ([]gateway.DeviceFamily, error) {
	client, err := a.adminClient()
	if err != nil {
		return nil, err
	}
	return client.ListDeviceFamilies()
}

// CreateDeviceFamily adds a family via POST /admin/device-families.
func (a *App) CreateDeviceFamily(family gateway.DeviceFamily) (*gateway.DeviceFamily, error) {
	if strings.TrimSpace(family.DFID) == "" {
		return nil, fmt.Errorf("dfid is required")
	}
	if strings.TrimSpace(family.FriendlyName) == "" {
		return nil, fmt.Errorf("friendly_name is required")
	}
	client, err := a.adminClient()
	if err != nil {
		return nil, err
	}
	return client.CreateDeviceFamily(family)
}

// CreateDeviceType adds a type via POST /admin/device-types.
func (a *App) CreateDeviceType(typ gateway.DeviceType) (*gateway.DeviceType, error) {
	if strings.TrimSpace(typ.DTID) == "" {
		return nil, fmt.Errorf("dtid is required")
	}
	if strings.TrimSpace(typ.DFID) == "" {
		return nil, fmt.Errorf("dfid is required")
	}
	if strings.TrimSpace(typ.FriendlyName) == "" {
		return nil, fmt.Errorf("friendly_name is required")
	}
	if len(typ.Capabilities) == 0 {
		return nil, fmt.Errorf("at least one capability is required")
	}
	client, err := a.adminClient()
	if err != nil {
		return nil, err
	}
	return client.CreateDeviceType(typ)
}

// ApplyRegistryResult wraps gateway apply output for the UI.
type ApplyRegistryResult struct {
	Added    []string `json:"added"`
	Updated  []string `json:"updated"`
	Rejected []string `json:"rejected"`
}

// ApplyRegistry applies the server-side registry YAML (POST /admin/device-registry/apply).
func (a *App) ApplyRegistry(dryRun bool) (*ApplyRegistryResult, error) {
	client, err := a.adminClient()
	if err != nil {
		return nil, err
	}
	result, err := client.ApplyRegistry(dryRun)
	if err != nil {
		return nil, err
	}
	return &ApplyRegistryResult{
		Added:    result.Added,
		Updated:  result.Updated,
		Rejected: result.Rejected,
	}, nil
}

// DiscoverDevices probes Pi hosts. fullLAN=true scans the subnet (slow); false only checks configured hosts.
func (a *App) DiscoverDevices(manualHost string, fullLAN bool) []discover.Candidate {
	a.mu.Lock()
	hosts := []string{}
	if manualHost != "" {
		hosts = append(hosts, manualHost)
	}
	if a.cfg.SSHHost != "" {
		hosts = append(hosts, a.cfg.SSHHost)
	}
	a.mu.Unlock()

	timeout := 12 * time.Second
	if fullLAN {
		timeout = 45 * time.Second
	}
	ctx, cancel := context.WithTimeout(a.ctx, timeout)
	defer cancel()

	return discover.Scan(ctx, discover.ScanOptions{
		ManualHosts: hosts,
		Timeout:     2 * time.Second,
		FullSubnet:  fullLAN,
	})
}

var factoryDeviceIDRE = regexp.MustCompile(`^[a-zA-Z0-9]+$`)

// ProvisionRequest holds v2 factory provision inputs.
type ProvisionRequest struct {
	DTID      string `json:"dtid"`
	DeviceID  string `json:"device_id"`
	Serial    string `json:"serial"`
	HWVersion string `json:"hw_version"`
	Overwrite *bool  `json:"overwrite"`
}

func provisionOverwrite(flag *bool) bool {
	if flag == nil {
		return true
	}
	return *flag
}

// ProvisionResult holds cloud registration outcome (no private key exposed).
type ProvisionResult struct {
	GlobalDeviceID string `json:"global_device_id"`
	DeviceID       string `json:"device_id"`
	DTID           string `json:"dtid"`
	DSID           string `json:"dsid"`
	SerialNumber   string `json:"serial_number"`
	MQTTUsername   string `json:"mqtt_username"`
	MQTTPassword   string `json:"mqtt_password"`
}

// ListDeviceTypes returns registry device types for the provision DTID picker.
func (a *App) ListDeviceTypes() ([]gateway.DeviceType, error) {
	client, err := a.adminClient()
	if err != nil {
		return nil, err
	}
	return client.ListDeviceTypes()
}

// Provision generates keys and registers the device via gateway admin API (v2).
func (a *App) Provision(req ProvisionRequest) (*ProvisionResult, error) {
	if strings.TrimSpace(req.DTID) == "" {
		return nil, fmt.Errorf("device type (dtid) is required — apply registry before first v2 unit")
	}
	deviceIDShort := strings.TrimSpace(req.DeviceID)
	if deviceIDShort == "" {
		return nil, fmt.Errorf("factory device_id is required")
	}
	if strings.Contains(deviceIDShort, "_") || !factoryDeviceIDRE.MatchString(deviceIDShort) {
		return nil, fmt.Errorf("device_id must be alphanumeric with no underscore (e.g. 00042)")
	}
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

	result, err := provision.RegisterDevice(
		gatewayURL, token, req.DTID, deviceIDShort, req.Serial, hw, keypair.PublicKeyPEM, provisionOverwrite(req.Overwrite),
	)
	if err != nil {
		var apiErr *provision.APIError
		if errors.As(err, &apiErr) {
			switch apiErr.Code {
			case "serial_already_provisioned":
				if !provisionOverwrite(req.Overwrite) {
					return nil, fmt.Errorf("serial %q already provisioned — enable Overwrite or use a different serial", req.Serial)
				}
			case "unknown_device_type":
				return nil, fmt.Errorf("dtid %q not in registry — run device-service registry apply before provisioning", req.DTID)
			case "device_id_conflict":
				return nil, fmt.Errorf("device_id %q already used for dtid %q", deviceIDShort, req.DTID)
			case "invalid_provision_request", "invalid_device_id":
				return nil, fmt.Errorf("invalid provision request — check dtid and device_id")
			case "device_service_unavailable", "device_service_register_failed":
				return nil, fmt.Errorf("device service unavailable — ensure registry is applied and device-service is running")
			}
		}
		return nil, err
	}

	globalID := result.GlobalDeviceID
	if globalID == "" {
		globalID = result.DeviceID
	}

	identityJSON, err := device.BuildIdentityJSON(
		globalID, deviceIDShort, req.DTID, result.DSID, req.Serial, hw, result.MQTTUsername, result.MQTTPassword,
	)
	if err != nil {
		return nil, err
	}

	a.mu.Lock()
	a.pendingPrivateKey = keypair.PrivateKeyPEM
	a.pendingIdentity = identityJSON
	a.pendingDeviceID = globalID
	a.pendingMQTTUser = result.MQTTUsername
	a.pendingMQTTPass = result.MQTTPassword
	a.mu.Unlock()

	return &ProvisionResult{
		GlobalDeviceID: globalID,
		DeviceID:       deviceIDShort,
		DTID:           req.DTID,
		DSID:           result.DSID,
		SerialNumber:   req.Serial,
		MQTTUsername:   result.MQTTUsername,
		MQTTPassword:   result.MQTTPassword,
	}, nil
}

// InstallRequest configures SSH install step.
type InstallRequest struct {
	Host            string `json:"host"`
	SSHUser         string `json:"ssh_user"`
	SSHPort         int    `json:"ssh_port"`
	SSHPassword     string `json:"ssh_password"`
	DeployAgentEnv  bool   `json:"deploy_agent_env"`
	CheckoutAgent   *bool  `json:"checkout_agent"`
	GitHubToken     string `json:"github_token"`
}

func checkoutAgentEnabled(flag *bool) bool {
	if flag == nil {
		return true
	}
	return *flag
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

	sshUser := strings.ToLower(strings.TrimSpace(req.SSHUser))
	if sshUser == "" {
		sshUser = strings.ToLower(strings.TrimSpace(cfg.SSHUser))
	}
	port := req.SSHPort
	if port == 0 {
		port = cfg.SSHPort
	}
	sshPass := strings.TrimSpace(req.SSHPassword)
	if sshPass == "" {
		sshPass = strings.TrimSpace(cfg.SSHPassword)
	}

	agentEnv := ""
	if req.DeployAgentEnv {
		agentEnv = appcfg.AgentEnv(appcfg.BackendProfile(cfg.BackendProfile), cfg.MacIP)
	}

	ghToken := strings.TrimSpace(req.GitHubToken)
	if ghToken == "" {
		ghToken = strings.TrimSpace(cfg.GitHubToken)
	}

	var bundle *agentrelease.Bundle
	if checkoutAgentEnabled(req.CheckoutAgent) {
		if ghToken == "" {
			return nil, fmt.Errorf("GitHub token is required — set it in Environment or config.json (github_token)")
		}
		if !appcfg.ValidGitHubToken(ghToken) {
			return nil, fmt.Errorf("GitHub token in config is invalid or was overwritten — paste a new fine-grained PAT (github_pat_…) or classic token (ghp_…) in Environment and continue")
		}
		owner, repo, err := appcfg.GitHubRepo(cfg.AgentRepoURL)
		if err != nil {
			return nil, err
		}
		bundle, err = agentrelease.FetchLatestDefaultTimeout(owner, repo, ghToken, cfg.AgentArtifactName)
		if err != nil {
			return nil, fmt.Errorf("download agent artifact: %w", err)
		}
	}

	return device.InstallCredentials(device.SSHConfig{
		Host:     resolved,
		Port:     port,
		User:     sshUser,
		Password: sshPass,
	}, priv, identity, agentEnv, req.DeployAgentEnv, device.AgentInstallOptions{
		Enabled: checkoutAgentEnabled(req.CheckoutAgent),
		Bundle:  bundle,
	})
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
	user = strings.ToLower(strings.TrimSpace(user))
	if port == 0 {
		port = cfg.SSHPort
	}
	resolved, _ := discover.ResolveHost(host)

	sshPass := strings.TrimSpace(password)
	if sshPass == "" {
		sshPass = strings.TrimSpace(cfg.SSHPassword)
	}

	return device.TestSSH(device.SSHConfig{
		Host:     resolved,
		Port:     port,
		User:     user,
		Password: sshPass,
	})
}
