package config_test

import (
	"strings"
	"testing"

	"bell-provisioner/internal/config"
)

func TestControlAgentEnv_noCameraKeys(t *testing.T) {
	env := config.ControlAgentEnv(config.ProfileVPS, "")
	for _, forbidden := range []string{"CAMERA_", "MOTION_", "GPIO_", "/etc/doorbell"} {
		if strings.Contains(env, forbidden) {
			t.Errorf("ControlAgentEnv must not contain %q", forbidden)
		}
	}
	for _, want := range []string{
		"IDENTITY_PATH=/etc/vyooham/identity.json",
		"MQTT_TLS_CLIENT_KEY=/etc/vyooham/device.key",
		"SETUP_SERVER_PORT=4444",
		"MQTT_BROKER_URL=tcp://api.vyooham.com:1883",
	} {
		if !strings.Contains(env, want) {
			t.Errorf("ControlAgentEnv missing %q", want)
		}
	}
}

func TestControlAgentEnv_macLAN(t *testing.T) {
	env := config.ControlAgentEnv(config.ProfileMacLAN, "10.0.0.5")
	if !strings.Contains(env, "MQTT_BROKER_URL=tcp://10.0.0.5:1883") {
		t.Fatalf("expected Mac LAN broker, got:\n%s", env)
	}
}
