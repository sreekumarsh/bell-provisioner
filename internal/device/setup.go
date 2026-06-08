package device

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"time"

	"bell-provisioner/internal/discover"
)

const setupServerPort = 4444

type setupIdentityResponse struct {
	DeviceID     string `json:"device_id"`
	SerialNumber string `json:"serial_number"`
	HWVersion    string `json:"hw_version"`
}

// waitForSetupServer polls the agent claim/setup HTTP server until it responds or timeout.
func waitForSetupServer(host, expectedDeviceID string, timeout time.Duration) bool {
	if timeout <= 0 {
		timeout = 45 * time.Second
	}
	resolved, _ := discover.ResolveHost(host)
	deadline := time.Now().Add(timeout)
	client := &http.Client{Timeout: 3 * time.Second}

	for time.Now().Before(deadline) {
		if probeSetupServer(client, resolved, expectedDeviceID) {
			return true
		}
		time.Sleep(2 * time.Second)
	}
	return probeSetupServer(client, resolved, expectedDeviceID)
}

func probeSetupServer(client *http.Client, host, expectedDeviceID string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	url := setupIdentityURL(host)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return false
	}
	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return false
	}

	var id setupIdentityResponse
	if err := json.NewDecoder(resp.Body).Decode(&id); err != nil {
		return false
	}
	if expectedDeviceID != "" && id.DeviceID != expectedDeviceID {
		return false
	}
	return id.DeviceID != ""
}

// ProbeSetupServerForTest verifies the setup HTTP identity endpoint (tests only).
func ProbeSetupServerForTest(host, expectedDeviceID string) bool {
	client := &http.Client{Timeout: 2 * time.Second}
	return probeSetupServer(client, host, expectedDeviceID)
}

func setupIdentityURL(host string) string {
	if _, _, err := net.SplitHostPort(host); err == nil {
		return fmt.Sprintf("http://%s/setup/identity", host)
	}
	return fmt.Sprintf("http://%s:%d/setup/identity", host, setupServerPort)
}
