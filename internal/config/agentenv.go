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

// AgentEnv returns agent.env content for the Pi.
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
