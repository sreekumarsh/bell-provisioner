package device

import (
	"encoding/json"
)

// Identity is written to /etc/doorbell/identity.json on the Pi.
type Identity struct {
	DeviceID     string `json:"device_id"`
	SerialNumber string `json:"serial_number"`
	HWVersion    string `json:"hw_version"`
	MQTTUsername string `json:"mqtt_username"`
	MQTTPassword string `json:"mqtt_password"`
	Claimed      bool   `json:"claimed"`
}

// BuildIdentityJSON returns formatted identity.json bytes.
func BuildIdentityJSON(deviceID, serial, hw, mqttUser, mqttPass string) ([]byte, error) {
	id := Identity{
		DeviceID:     deviceID,
		SerialNumber: serial,
		HWVersion:    hw,
		MQTTUsername: mqttUser,
		MQTTPassword: mqttPass,
		Claimed:      false,
	}
	return json.MarshalIndent(id, "", "  ")
}
