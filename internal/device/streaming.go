package device

import (
	"fmt"
	"strings"

	"golang.org/x/crypto/ssh"
)

const (
	motionModelRemotePath   = "/var/lib/doorbell/models/yolov8n.onnx"
	motionScriptRemotePath  = "/usr/local/bin/motion-classify.py"
	motionClassifyTmpPath   = "/tmp/motion-classify.py"
	motionModelTmpPath      = "/tmp/yolov8n.onnx"
)

// DoorbellRuntimeDepsInstallScript returns the bash script that installs ffmpeg, v4l-utils,
// curl/ca-certificates, go2rtc, and the Python motion venv. Idempotent. Doorbell profile only.
func DoorbellRuntimeDepsInstallScript() string {
	return strings.TrimSpace(`
APT_PACKAGES=""
command -v ffmpeg >/dev/null 2>&1 || APT_PACKAGES="$APT_PACKAGES ffmpeg"
command -v v4l2-ctl >/dev/null 2>&1 || APT_PACKAGES="$APT_PACKAGES v4l-utils"
command -v curl >/dev/null 2>&1 || APT_PACKAGES="$APT_PACKAGES curl ca-certificates"
command -v python3 >/dev/null 2>&1 || APT_PACKAGES="$APT_PACKAGES python3 python3-venv python3-pip"
dpkg -s python3-venv >/dev/null 2>&1 || APT_PACKAGES="$APT_PACKAGES python3-venv"
dpkg -s python3-pip >/dev/null 2>&1 || APT_PACKAGES="$APT_PACKAGES python3-pip"
if [ -n "$APT_PACKAGES" ]; then
  export DEBIAN_FRONTEND=noninteractive
  echo "==> Installing apt packages:${APT_PACKAGES}"
  sudo apt-get update -qq
  sudo apt-get install -y -qq $APT_PACKAGES
fi

if [ ! -x /usr/local/bin/go2rtc ]; then
  if ! command -v curl >/dev/null 2>&1; then
    echo "curl is required to install go2rtc" >&2
    exit 1
  fi
  ARCH="$(uname -m)"
  case "$ARCH" in
    aarch64|arm64) GO2RTC_ASSET=go2rtc_linux_arm64 ;;
    x86_64|amd64)  GO2RTC_ASSET=go2rtc_linux_amd64 ;;
    *)
      echo "unsupported architecture for go2rtc: $ARCH" >&2
      exit 1
      ;;
  esac
  if [ -n "${GO2RTC_VERSION:-}" ]; then
    URL="https://github.com/AlexxIT/go2rtc/releases/download/v${GO2RTC_VERSION}/${GO2RTC_ASSET}"
    echo "==> Installing go2rtc ${GO2RTC_VERSION} (${GO2RTC_ASSET})"
  else
    URL="https://github.com/AlexxIT/go2rtc/releases/latest/download/${GO2RTC_ASSET}"
    echo "==> Installing go2rtc latest (${GO2RTC_ASSET})"
  fi
  curl -fsSL "$URL" -o /tmp/go2rtc
  sudo install -m 755 /tmp/go2rtc /usr/local/bin/go2rtc
  rm -f /tmp/go2rtc
fi

if [ ! -x /var/lib/doorbell/venv/bin/python3 ]; then
  echo "==> Creating motion Python venv"
  sudo mkdir -p /var/lib/doorbell/models
  sudo python3 -m venv /var/lib/doorbell/venv
  sudo /var/lib/doorbell/venv/bin/pip install -qq --upgrade pip onnxruntime pillow numpy
fi

echo "==> Agent runtime dependencies OK"
`)
}

// AgentRuntimeDepsInstallScript is an alias for DoorbellRuntimeDepsInstallScript (tests/CLI).
func AgentRuntimeDepsInstallScript() string {
	return DoorbellRuntimeDepsInstallScript()
}

func installDoorbellRuntimeDeps(client *ssh.Client, sudoPassword string) error {
	if err := runSudo(client, sudoPassword, DoorbellRuntimeDepsInstallScript()); err != nil {
		return fmt.Errorf("apt/ffmpeg/go2rtc/motion venv install failed (check Pi network and sudo password): %w", err)
	}
	if err := installMotionAssets(client, sudoPassword); err != nil {
		return err
	}
	return nil
}

func installMotionAssets(client *ssh.Client, sudoPassword string) error {
	model, err := runtimeAssets.ReadFile("assets/yolov8n.onnx")
	if err != nil {
		return fmt.Errorf("read embedded motion model: %w", err)
	}
	script, err := runtimeAssets.ReadFile("assets/motion-classify.py")
	if err != nil {
		return fmt.Errorf("read embedded motion-classify.py: %w", err)
	}

	if err := uploadFile(client, motionModelTmpPath, model, 0644); err != nil {
		return fmt.Errorf("upload motion model: %w", err)
	}
	if err := uploadFile(client, motionClassifyTmpPath, script, 0755); err != nil {
		return fmt.Errorf("upload motion-classify.py: %w", err)
	}

	installScript := fmt.Sprintf(`
sudo mkdir -p /var/lib/doorbell/models
if [ ! -f %s ]; then
  sudo install -m 644 %s %s
fi
if [ ! -x %s ]; then
  sudo install -m 755 %s %s
fi
rm -f %s %s
`, shellSingleQuote(motionModelRemotePath), shellSingleQuote(motionModelTmpPath), shellSingleQuote(motionModelRemotePath),
		shellSingleQuote(motionScriptRemotePath), shellSingleQuote(motionClassifyTmpPath), shellSingleQuote(motionScriptRemotePath),
		shellSingleQuote(motionModelTmpPath), shellSingleQuote(motionClassifyTmpPath))

	if err := runSudo(client, sudoPassword, installScript); err != nil {
		return fmt.Errorf("install motion assets on Pi: %w", err)
	}
	return nil
}

func verifyDoorbellRuntimeDeps(client *ssh.Client) (ffmpegOK, go2rtcOK, motionOK bool) {
	ffmpegOK = runCmd(client, "command -v ffmpeg >/dev/null 2>&1") == nil
	go2rtcOK = runCmd(client, "/usr/local/bin/go2rtc -version >/dev/null 2>&1") == nil
	motionOK = runCmd(client, "test -f /var/lib/doorbell/models/yolov8n.onnx && test -x /usr/local/bin/motion-classify.py && test -x /var/lib/doorbell/venv/bin/python3") == nil
	return ffmpegOK, go2rtcOK, motionOK
}
