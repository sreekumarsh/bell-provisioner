package config

import (
	"fmt"
	"strings"
)

// BackendProfile selects agent.env template values.
type BackendProfile string

const (
	ProfileVPS    BackendProfile = "vps"
	ProfileMacLAN BackendProfile = "mac"
)

// AgentEnv returns doorbell-agent.env content for the Pi.
func AgentEnv(profile BackendProfile, macIP string) string {
	switch profile {
	case ProfileMacLAN:
		ip := macIP
		if ip == "" {
			ip = "192.168.4.66"
		}
		return fmt.Sprintf(`MQTT_BROKER_URL=tcp://%s:1883
MQTT_TLS_ENABLED=false

DEVICE_SERVICE_URL=http://%s:8081
DEVICE_SERVICE_INTERNAL_SECRET=internal-secret-change-in-prod

GO2RTC_BINARY=/usr/local/bin/go2rtc
GO2RTC_CONFIG_FILE=/etc/doorbell/go2rtc.yaml
GO2RTC_API_URL=http://localhost:1984
GO2RTC_RTSP_PORT=8554
GO2RTC_WEBRTC_PORT=8555

CAMERA_DEVICE=/dev/video0
CAMERA_VIDEO_FORMAT=MJPEG
CAMERA_SOURCE_TYPE=v4l2

GPIO_PIR_PIN=4
GPIO_PIR_AVAILABLE=true
MOTION_MODEL_PATH=/var/lib/doorbell/models/yolov8n.onnx
MOTION_USE_EXEC=true
MOTION_INFER_SCRIPT=/usr/local/bin/motion-classify.py
MOTION_PYTHON=/var/lib/doorbell/venv/bin/python3
MOTION_CLASSIFY_MIN_CONF=0.5
MOTION_CLASSIFY_INPUT_SIZE=640

GPIO_BUTTON_PIN=17
GPIO_IR_LED_PIN=22
GPIO_LED_RED_PIN=5
GPIO_LED_GREEN_PIN=6
GPIO_LED_BLUE_PIN=13

POWER_MODE=wired
LOG_LEVEL=info
LOG_FORMAT=json
HEARTBEAT_INTERVAL_SECONDS=30
`, ip, ip)
	default:
		return strings.TrimSpace(`MQTT_BROKER_URL=tcp://api.vyooham.com:1883
MQTT_TLS_ENABLED=false

DEVICE_SERVICE_URL=https://api.vyooham.com
STREAM_SERVER_HOST=api.vyooham.com
STREAM_SERVER_RTSP_PORT=8554

GO2RTC_BINARY=/usr/local/bin/go2rtc
GO2RTC_CONFIG_FILE=/etc/doorbell/go2rtc.yaml
GO2RTC_API_URL=http://localhost:1984
GO2RTC_RTSP_PORT=8554
GO2RTC_WEBRTC_PORT=8555

CAMERA_DEVICE=/dev/video0
CAMERA_VIDEO_FORMAT=MJPEG
CAMERA_SOURCE_TYPE=v4l2

GPIO_BUTTON_AVAILABLE=false
GPIO_PIR_PIN=4
GPIO_PIR_AVAILABLE=true
MOTION_PIXEL_ALWAYS=false
MOTION_MODEL_PATH=/var/lib/doorbell/models/yolov8n.onnx
MOTION_USE_EXEC=true
MOTION_INFER_SCRIPT=/usr/local/bin/motion-classify.py
MOTION_PYTHON=/var/lib/doorbell/venv/bin/python3
MOTION_CLASSIFY_MIN_CONF=0.5
MOTION_CLASSIFY_INPUT_SIZE=640
MOTION_NOTIFY_COOLDOWN_SEC=45

GPIO_IR_LED_PIN=22
GPIO_LED_RED_PIN=5
GPIO_LED_GREEN_PIN=6
GPIO_LED_BLUE_PIN=13

POWER_MODE=wired
LOG_LEVEL=info
LOG_FORMAT=json
HEARTBEAT_INTERVAL_SECONDS=30
WATERMARK_ENABLED=true
`) + "\n"
	}
}

// ControlAgentEnv returns agent.env for the NVR control-agent (/etc/vyooham/).
func ControlAgentEnv(profile BackendProfile, macIP string) string {
	switch profile {
	case ProfileMacLAN:
		ip := macIP
		if ip == "" {
			ip = "192.168.4.66"
		}
		return fmt.Sprintf(`MQTT_BROKER_URL=tcp://%s:1883
MQTT_TLS_ENABLED=false
IDENTITY_PATH=/etc/vyooham/identity.json
MQTT_TLS_CA_FILE=/etc/vyooham/ca.crt
MQTT_TLS_CLIENT_CERT=/etc/vyooham/device.crt
MQTT_TLS_CLIENT_KEY=/etc/vyooham/device.key
HEARTBEAT_SEC=30
SETUP_SERVER_PORT=4444
LOG_LEVEL=info
LOG_FORMAT=json
FW_VERSION=dev
`, ip)
	default:
		return strings.TrimSpace(`MQTT_BROKER_URL=tcp://api.vyooham.com:1883
MQTT_TLS_ENABLED=false
IDENTITY_PATH=/etc/vyooham/identity.json
MQTT_TLS_CA_FILE=/etc/vyooham/ca.crt
MQTT_TLS_CLIENT_CERT=/etc/vyooham/device.crt
MQTT_TLS_CLIENT_KEY=/etc/vyooham/device.key
HEARTBEAT_SEC=30
SETUP_SERVER_PORT=4444
LOG_LEVEL=info
LOG_FORMAT=json
FW_VERSION=dev
`) + "\n"
	}
}

// MQTTTransport selects the broker endpoint written into a generated agent.env.
//
// Phase A of bell-docs/architecture/mqtt-tls.md keeps the installed fleet on
// plain 1883 while the broker runs dual-stack. Sense has no installed base, so
// it is the one product that could start on 8883 — but only once the mTLS
// rollout is actually complete. See SenseControlAgentEnv.
type MQTTTransport string

const (
	// MQTTPlain is Phase A: tcp://…:1883, MQTT_TLS_ENABLED=false.
	MQTTPlain MQTTTransport = "plain"
	// MQTTMutualTLS is the design target: ssl://mqtt.vyooham.com:8883 + mTLS.
	MQTTMutualTLS MQTTTransport = "mtls"
)

// SenseTLSHost is the mTLS broker hostname from mqtt-tls.md (must match the
// broker cert SAN). As of this writing it has no DNS A record.
const SenseTLSHost = "mqtt.vyooham.com"

// DefaultSenseTransport is what Sense boxes are provisioned with unless the
// operator opts in to mTLS (config sense_mqtt_transport / --transport).
//
// This is deliberately MQTTPlain rather than MQTTMutualTLS despite Sense having
// zero installed base, because both preconditions for 8883 are unmet today:
//
//  1. mqtt.vyooham.com has no DNS A record (NXDOMAIN) — a box provisioned with
//     ssl://mqtt.vyooham.com:8883 cannot resolve its broker at all.
//  2. auth-service sets neither MQTT_CA_CERT_FILE nor MQTT_CA_KEY_FILE in any
//     deployed config, so its mqttca.Signer is nil and provision responses omit
//     device_crt/ca_crt (both are `omitempty`) — there is no client cert to
//     present even if the endpoint resolved.
//
// Flip this to MQTTMutualTLS once both hold; nothing else needs to change.
const DefaultSenseTransport = MQTTPlain

// SenseControlAgentEnv returns control-agent.env for Vyooham Sense
// (/etc/vyooham-sense/). Var set and defaults mirror
// vyooham-sense/services/control-agent/.env.example and its README § Config.
//
// transport selects plain 1883 vs mTLS 8883; empty means DefaultSenseTransport.
// The Mac LAN backend profile is always plain — the dev broker has no CA.
func SenseControlAgentEnv(profile BackendProfile, macIP string, transport MQTTTransport) string {
	if transport == "" {
		transport = DefaultSenseTransport
	}

	brokerURL := "tcp://api.vyooham.com:1883"
	tlsEnabled := "false"
	switch {
	case profile == ProfileMacLAN:
		ip := macIP
		if ip == "" {
			ip = "192.168.4.66"
		}
		brokerURL = "tcp://" + ip + ":1883"
	case transport == MQTTMutualTLS:
		brokerURL = "ssl://" + SenseTLSHost + ":8883"
		tlsEnabled = "true"
	}

	return fmt.Sprintf(`MQTT_BROKER_URL=%s
MQTT_TLS_ENABLED=%s
IDENTITY_PATH=/etc/vyooham-sense/identity.json
MQTT_TLS_CA_FILE=/etc/vyooham-sense/ca.crt
MQTT_TLS_CLIENT_CERT=/etc/vyooham-sense/device.crt
MQTT_TLS_CLIENT_KEY=/etc/vyooham-sense/device.key
CLAIM_GRANT_PUBKEY_PATH=/etc/vyooham-sense/claim-grant.pub
HEARTBEAT_SEC=30
SETUP_SERVER_PORT=4444
DETECTOR_ADDR=http://127.0.0.1:9101
CLIPBUFFER_ADDR=http://127.0.0.1:9102
WEBUI_ADDR=http://127.0.0.1:8080
EVENT_INGEST_ADDR=127.0.0.1:9105
LOG_LEVEL=info
LOG_FORMAT=json
FW_VERSION=dev
`, brokerURL, tlsEnabled)
}

// MQTTBrokerURL returns the broker URL for verify step.
func MQTTBrokerURL(profile BackendProfile, macIP string) string {
	switch profile {
	case ProfileMacLAN:
		ip := macIP
		if ip == "" {
			ip = "192.168.4.66"
		}
		return "tcp://" + ip + ":1883"
	default:
		return "tcp://api.vyooham.com:1883"
	}
}
