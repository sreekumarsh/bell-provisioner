package device

import (
	"strings"
	"testing"
)

func TestLooksLikeShellPrompt_stripsOSCSuffix(t *testing.T) {
	// Sense getty: prompt char then OSC shell-integration (BEL-terminated).
	raw := "root@vyooham-sense-mini:~#\x1b]633;A\x07"
	if !looksLikeShellPrompt(raw) {
		t.Fatalf("expected prompt with OSC suffix to match, got cleaned=%q", stripTerminalNoise(raw))
	}
	// ST-terminated OSC variant.
	rawST := "root@host:~#\x1b]3008;start=1\x1b\\"
	if !looksLikeShellPrompt(rawST) {
		t.Fatalf("expected ST-terminated OSC prompt to match, got cleaned=%q", stripTerminalNoise(rawST))
	}
	if looksLikeShellPrompt("login:") {
		t.Fatal("login is not a shell prompt")
	}
	if !looksLikeShellPrompt("root@vyooham-sense-mini:~#") {
		t.Fatal("expected plain prompt")
	}
}

func TestNextUARTLoginAction_sendsCredentialsOnce(t *testing.T) {
	cases := []struct {
		name  string
		text  string
		phase uartLoginPhase
		want  uartLoginAction
	}{
		{"login prompt", "Debian GNU/Linux\r\nhost login: ", uartLoginAwaitUser, uartLoginSendUser},
		{"login already sent — do not resend", "host login: ", uartLoginAwaitPassword, uartLoginWait},
		{"password prompt", "Password: ", uartLoginAwaitPassword, uartLoginSendPassword},
		{"password linger after send — do not resend", "Password: ", uartLoginAwaitShell, uartLoginWait},
		{"shell after login", "root@host:~#\x1b]633;A\x07", uartLoginAwaitShell, uartLoginSuccess},
		{"auth failure message", "Login incorrect\r\n", uartLoginAwaitShell, uartLoginAuthFailed},
		{"login redisplayed after password", "host login: ", uartLoginAwaitShell, uartLoginAuthFailed},
		{"already at shell", "root@host:~# ", uartLoginAwaitUser, uartLoginSuccess},
		{"password-only getty", "Password:", uartLoginAwaitUser, uartLoginSendPassword},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := nextUARTLoginAction(tc.text, tc.phase)
			if got != tc.want {
				t.Fatalf("nextUARTLoginAction(%q, %d) = %d, want %d", tc.text, tc.phase, got, tc.want)
			}
		})
	}
}

func TestNextUARTLoginAction_pollResendRegression(t *testing.T) {
	// Simulate several polls where getty still shows login:/password: —
	// after the first send, further polls must Wait, not resend.
	phase := uartLoginAwaitUser
	actions := make([]uartLoginAction, 0, 6)
	for i := 0; i < 3; i++ {
		a := nextUARTLoginAction("host login: ", phase)
		actions = append(actions, a)
		if a == uartLoginSendUser {
			phase = uartLoginAwaitPassword
		}
	}
	for i := 0; i < 3; i++ {
		a := nextUARTLoginAction("Password: ", phase)
		actions = append(actions, a)
		if a == uartLoginSendPassword {
			phase = uartLoginAwaitShell
		}
	}
	var sendUser, sendPass int
	for _, a := range actions {
		if a == uartLoginSendUser {
			sendUser++
		}
		if a == uartLoginSendPassword {
			sendPass++
		}
	}
	if sendUser != 1 || sendPass != 1 {
		t.Fatalf("want exactly one user + one password send, got user=%d pass=%d actions=%v",
			sendUser, sendPass, actions)
	}
	if phase != uartLoginAwaitShell {
		t.Fatalf("want AwaitShell after password, got %d", phase)
	}
}

func TestStripTerminalNoise_OSC(t *testing.T) {
	in := "EXIT:0\x1b]633;B\x07\r\n"
	out := stripTerminalNoise(in)
	if strings.Contains(out, "\x1b") || strings.Contains(out, "]633") {
		t.Fatalf("OSC not stripped: %q", out)
	}
	if !strings.Contains(out, "EXIT:0") {
		t.Fatalf("lost EXIT token: %q", out)
	}
}

func TestLineAnchoredIndex_findsMarkerAfterOSC(t *testing.T) {
	marker := "BP_END_999"
	// Sense: OSC immediately before the marker on the same visual line.
	raw := "EXIT:0\r\n\x1b]3008;start=abc;type=command\x07" + marker + "\r\n"
	cleaned := stripTerminalNoise(raw)
	idx := lineAnchoredIndex(cleaned, marker)
	if idx < 0 {
		t.Fatalf("expected marker after OSC strip, cleaned=%q", cleaned)
	}
	before := cleaned[:idx]
	code, ok := parseExitCode(before)
	if !ok || code != 0 {
		t.Fatalf("want EXIT:0 before marker, got %d ok=%v in %q", code, ok, before)
	}
	// Echoed command line must still be ignored even after strip.
	echoed := "root@host:~# printf '%s\\n' '" + marker + "'\r\n"
	if lineAnchoredIndex(stripTerminalNoise(echoed), marker) >= 0 {
		t.Fatal("must not match marker inside echoed command after OSC strip")
	}
}
