package device_test

import (
	"strings"
	"testing"

	"bell-provisioner/internal/device"
)

func TestAgentRuntimeDepsInstallScript_idempotentChecks(t *testing.T) {
	script := device.AgentRuntimeDepsInstallScript()

	required := []string{
		"command -v ffmpeg",
		"command -v v4l2-ctl",
		"command -v curl",
		"command -v python3",
		"ffmpeg",
		"v4l-utils",
		"python3-venv",
		"curl ca-certificates",
		"DEBIAN_FRONTEND=noninteractive",
		"[ ! -x /usr/local/bin/go2rtc ]",
		"/usr/local/bin/go2rtc",
		"/var/lib/doorbell/venv",
		"onnxruntime pillow numpy",
		"GO2RTC_VERSION",
		"releases/latest/download",
		"go2rtc_linux_arm64",
	}
	for _, want := range required {
		if !strings.Contains(script, want) {
			t.Errorf("script missing %q", want)
		}
	}
}

func TestAgentRuntimeDepsInstallScript_doesNotExitEarlyWhenGo2rtcPresent(t *testing.T) {
	script := device.AgentRuntimeDepsInstallScript()
	if strings.Contains(script, "if [ -x /usr/local/bin/go2rtc ]; then\n  exit 0\nfi") {
		t.Fatal("script must not skip apt packages when go2rtc is already installed")
	}
}
