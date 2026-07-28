package device

import (
	"encoding/json"
	"fmt"
)

// Identity is written to the device etc dir (v2 factory provision).
// Doorbell: /etc/doorbell/identity.json — NVR: /etc/vyooham/identity.json.
type Identity struct {
	GlobalDeviceID  string `json:"global_device_id"`
	DeviceID        string `json:"device_id"`
	DTID            string `json:"dtid"`
	DSID            string `json:"dsid"`
	SerialNumber    string `json:"serial_number,omitempty"`
	HardwareVersion string `json:"hardware_version"`
	MQTTUsername    string `json:"mqtt_username"`
	MQTTPassword    string `json:"mqtt_password"`
	Claimed         bool   `json:"claimed"`
}

// BuildIdentityJSON returns formatted v2 identity.json bytes.
func BuildIdentityJSON(globalID, deviceIDShort, dtid, dsid, serial, hw, mqttUser, mqttPass string) ([]byte, error) {
	id := Identity{
		GlobalDeviceID:  globalID,
		DeviceID:        deviceIDShort,
		DTID:            dtid,
		DSID:            dsid,
		SerialNumber:    serial,
		HardwareVersion: hw,
		MQTTUsername:    mqttUser,
		MQTTPassword:    mqttPass,
		Claimed:         false,
	}
	return json.MarshalIndent(id, "", "  ")
}

// ParseIdentity unmarshals identity.json bytes.
func ParseIdentity(identityJSON []byte) (Identity, error) {
	var id Identity
	if err := json.Unmarshal(identityJSON, &id); err != nil {
		return Identity{}, fmt.Errorf("parse identity.json: %w", err)
	}
	return id, nil
}
