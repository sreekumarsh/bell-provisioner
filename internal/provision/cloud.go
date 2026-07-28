package provision

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Result is the gateway provision response (v2).
type Result struct {
	DeviceID       string `json:"device_id"`
	GlobalDeviceID string `json:"global_device_id"`
	DSID           string `json:"dsid"`
	SerialNumber   string `json:"serial_number"`
	MQTTUsername   string `json:"mqtt_username"`
	MQTTPassword   string `json:"mqtt_password"`
	DeviceCrt      string `json:"device_crt,omitempty"`
	CaCrt          string `json:"ca_crt,omitempty"`
	ProvisionedAt  string `json:"provisioned_at"`
}

// APIError carries HTTP status and error code from the gateway/auth.
type APIError struct {
	Status  int
	Code    string
	Message string
}

func (e *APIError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return fmt.Sprintf("provision failed: HTTP %d (%s)", e.Status, e.Code)
}

// RegisterDevice calls POST /admin/devices/provision on the API gateway (v2).
func RegisterDevice(gatewayURL, accessToken, dtid, deviceID, serial, hwVersion, publicKeyPEM string, overwrite bool) (*Result, error) {
	base := strings.TrimRight(gatewayURL, "/")
	body, err := json.Marshal(map[string]any{
		"dtid":             dtid,
		"device_id":        deviceID,
		"serial_number":    serial,
		"hardware_version": hwVersion,
		"public_key_pem":   publicKeyPEM,
		"overwrite":        overwrite,
	})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodPost, base+"/admin/devices/provision", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+accessToken)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("gateway request: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusCreated {
		var errResp struct {
			Error   string `json:"error"`
			Message string `json:"message"`
		}
		_ = json.Unmarshal(respBody, &errResp)
		code := errResp.Error
		if code == "" {
			code = "provision_failed"
		}
		msg := errResp.Message
		if msg == "" {
			msg = strings.TrimSpace(string(respBody))
		}
		return nil, &APIError{Status: resp.StatusCode, Code: code, Message: msg}
	}

	var result Result
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	if result.GlobalDeviceID == "" {
		result.GlobalDeviceID = result.DeviceID
	}
	return &result, nil
}
