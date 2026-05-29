package device

import "golang.org/x/crypto/ssh"

// installStreamingDeps ensures curl is available and go2rtc is installed on the Pi.
const installStreamingDepsScript = `if [ -x /usr/local/bin/go2rtc ]; then
  exit 0
fi
if ! command -v curl >/dev/null 2>&1; then
  export DEBIAN_FRONTEND=noninteractive
  sudo apt-get update -qq
  sudo apt-get install -y -qq curl ca-certificates
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
GO2RTC_VERSION="${GO2RTC_VERSION:-1.9.9}"
URL="https://github.com/AlexxIT/go2rtc/releases/download/v${GO2RTC_VERSION}/${GO2RTC_ASSET}"
echo "==> Installing go2rtc ${GO2RTC_VERSION} (${GO2RTC_ASSET})"
curl -fsSL "$URL" -o /tmp/go2rtc
sudo install -m 755 /tmp/go2rtc /usr/local/bin/go2rtc
rm -f /tmp/go2rtc
`

func installStreamingDeps(client *ssh.Client, sudoPassword string) error {
	return runSudo(client, sudoPassword, installStreamingDepsScript)
}
