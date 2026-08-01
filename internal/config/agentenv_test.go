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

func TestSenseControlAgentEnv_paths(t *testing.T) {
	env := config.SenseControlAgentEnv(config.ProfileVPS, "", "")
	for _, want := range []string{
		"IDENTITY_PATH=/etc/vyooham-sense/identity.json",
		"MQTT_TLS_CA_FILE=/etc/vyooham-sense/ca.crt",
		"MQTT_TLS_CLIENT_CERT=/etc/vyooham-sense/device.crt",
		"MQTT_TLS_CLIENT_KEY=/etc/vyooham-sense/device.key",
		"HEARTBEAT_SEC=30",
		"SETUP_SERVER_PORT=4444",
		"EVENT_INGEST_ADDR=127.0.0.1:9105",
		"WEBUI_ADDR=http://127.0.0.1:8080",
	} {
		if !strings.Contains(env, want) {
			t.Errorf("SenseControlAgentEnv missing %q", want)
		}
	}
	// Must never point at the NVR's or the camera's directory.
	for _, forbidden := range []string{"/etc/vyooham/", "/etc/doorbell", "CAMERA_", "GPIO_"} {
		if strings.Contains(env, forbidden) {
			t.Errorf("SenseControlAgentEnv must not contain %q", forbidden)
		}
	}
}

func TestSenseControlAgentEnv_transport(t *testing.T) {
	// Default is plain until the mTLS preconditions hold.
	def := config.SenseControlAgentEnv(config.ProfileVPS, "", "")
	if !strings.Contains(def, "MQTT_BROKER_URL=tcp://api.vyooham.com:1883") ||
		!strings.Contains(def, "MQTT_TLS_ENABLED=false") {
		t.Fatalf("default transport should be plain 1883, got:\n%s", def)
	}

	mtls := config.SenseControlAgentEnv(config.ProfileVPS, "", config.MQTTMutualTLS)
	if !strings.Contains(mtls, "MQTT_BROKER_URL=ssl://mqtt.vyooham.com:8883") ||
		!strings.Contains(mtls, "MQTT_TLS_ENABLED=true") {
		t.Fatalf("mtls transport should be ssl 8883, got:\n%s", mtls)
	}

	// Mac LAN dev broker has no CA — always plain, even if mtls is asked for.
	mac := config.SenseControlAgentEnv(config.ProfileMacLAN, "10.0.0.5", config.MQTTMutualTLS)
	if !strings.Contains(mac, "MQTT_BROKER_URL=tcp://10.0.0.5:1883") ||
		!strings.Contains(mac, "MQTT_TLS_ENABLED=false") {
		t.Fatalf("mac LAN should stay plain, got:\n%s", mac)
	}
}

func TestControlAgentEnv_macLAN(t *testing.T) {
	env := config.ControlAgentEnv(config.ProfileMacLAN, "10.0.0.5")
	if !strings.Contains(env, "MQTT_BROKER_URL=tcp://10.0.0.5:1883") {
		t.Fatalf("expected Mac LAN broker, got:\n%s", env)
	}
}
