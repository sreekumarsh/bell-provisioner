package device

import (
	"strings"
	"testing"
)

func TestIsRootSSHUser(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		user string
		want bool
	}{
		{"root", true},
		{"Root", true},
		{" ROOT ", true},
		{"pi", false},
		{"", false},
		{"rooty", false},
	} {
		if got := isRootSSHUser(tc.user); got != tc.want {
			t.Errorf("isRootSSHUser(%q) = %v, want %v", tc.user, got, tc.want)
		}
	}
}

func TestStripSudoTokens(t *testing.T) {
	t.Parallel()
	in := `
sudo mkdir -p /etc/vyooham-sense
sudo install -m 755 /tmp/bin /usr/local/bin/control-agent
sudo systemctl daemon-reload && (sudo systemctl restart control-agent || sudo systemctl restart control-agent.service)
`
	got := stripSudoTokens(in)
	for _, bad := range []string{"sudo mkdir", "sudo install", "sudo systemctl", "(sudo "} {
		if strings.Contains(got, bad) {
			t.Fatalf("stripSudoTokens left %q in:\n%s", bad, got)
		}
	}
	for _, want := range []string{
		"mkdir -p /etc/vyooham-sense",
		"install -m 755 /tmp/bin /usr/local/bin/control-agent",
		"systemctl daemon-reload",
		"(systemctl restart control-agent",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("stripSudoTokens missing %q in:\n%s", want, got)
		}
	}
}
