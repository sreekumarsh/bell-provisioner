package device

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"go.bug.st/serial"
)

const (
	uartBaud           = 115200
	uartIOTimeout      = 8 * time.Second
	uartLoginTimeout   = 45 * time.Second
	uartCommandTimeout = 60 * time.Second
	uartChunkSize      = 2000 // fallback for oversized payloads only
)

// SerialPortInfo is a host serial device suitable for Sense UART getty.
type SerialPortInfo struct {
	Path string `json:"path"`
	Name string `json:"name"`
}

// UARTConfig holds serial connection parameters (root getty).
type UARTConfig struct {
	Port     string
	Password string
}

var uartPortPreferRE = regexp.MustCompile(`(?i)(usbserial|wchusbserial|slab_usb|usbmodem|cp210|ch340|ftdi)`)

// ListSerialPorts returns candidate macOS/Linux serial devices for factory UART.
func ListSerialPorts() ([]SerialPortInfo, error) {
	paths, err := serial.GetPortsList()
	if err != nil {
		return nil, fmt.Errorf("list serial ports: %w", err)
	}
	out := make([]SerialPortInfo, 0, len(paths))
	for _, p := range paths {
		base := filepath.Base(p)
		// Prefer callout devices on macOS (cu.*); skip tty.* duplicates.
		if strings.HasPrefix(base, "tty.") {
			continue
		}
		name := base
		if uartPortPreferRE.MatchString(base) {
			name = base + " (USB)"
		}
		out = append(out, SerialPortInfo{Path: p, Name: name})
	}
	// Stable-ish order: USB-looking first, then path.
	prefer := make([]SerialPortInfo, 0, len(out))
	rest := make([]SerialPortInfo, 0, len(out))
	for _, p := range out {
		if uartPortPreferRE.MatchString(p.Path) {
			prefer = append(prefer, p)
		} else {
			rest = append(rest, p)
		}
	}
	return append(prefer, rest...), nil
}

// InstallCredentialsUART writes a credentials-only bundle over UART getty and
// restarts the profile agent. Agent binaries are assumed already on the image.
func InstallCredentialsUART(cfg UARTConfig, bundle CredentialBundle) (*InstallResult, error) {
	port := strings.TrimSpace(cfg.Port)
	if port == "" {
		return nil, fmt.Errorf("serial port is required")
	}
	if _, err := os.Stat(port); err != nil {
		return nil, fmt.Errorf("serial port %s: %w", port, err)
	}
	// Password may be empty — lab images often allow empty root on UART.

	mode := &serial.Mode{BaudRate: uartBaud, DataBits: 8, Parity: serial.NoParity, StopBits: serial.OneStopBit}
	portIO, err := serial.Open(port, mode)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", port, err)
	}
	defer portIO.Close()
	_ = portIO.SetReadTimeout(uartIOTimeout)

	sess := &uartSession{port: portIO, buf: bytes.NewBuffer(nil)}
	if err := sess.loginRoot(strings.TrimSpace(cfg.Password)); err != nil {
		return nil, fmt.Errorf("UART login: %w", err)
	}

	etc := shellSingleQuote(bundle.EtcDir)
	if err := sess.runChecked(fmt.Sprintf("mkdir -p %s", etc), uartCommandTimeout); err != nil {
		return nil, fmt.Errorf("mkdir %s: %w", bundle.EtcDir, err)
	}

	for _, f := range bundle.Files {
		if err := sess.writeFileBase64(bundle.EtcDir, f); err != nil {
			return nil, fmt.Errorf("write %s: %w", f.Name, err)
		}
	}

	svc := bundle.ServiceName
	// Never wait on systemctl restart synchronously: with Restart=always a
	// crash-looping agent (Flashed → no identity) can make `restart` hang past
	// the UART marker timeout. --no-block returns immediately.
	restart := fmt.Sprintf(
		"systemctl daemon-reload; systemctl reset-failed %s 2>/dev/null || true; systemctl restart --no-block %s 2>/dev/null || systemctl restart --no-block %s.service 2>/dev/null || systemctl start --no-block %s 2>/dev/null || true",
		shellSingleQuote(svc), shellSingleQuote(svc), shellSingleQuote(svc), shellSingleQuote(svc),
	)
	if err := sess.runChecked(restart, 30*time.Second); err != nil {
		hint := ""
		errText := err.Error()
		if strings.Contains(errText, "not found") || strings.Contains(errText, "exit 5") {
			hint = fmt.Sprintf(
				" — unit %s missing; check the provisioned DTID maps to this board (Sense → control-agent, doorbell → doorbell-agent, NVR → control-agent)",
				svc,
			)
		}
		return nil, fmt.Errorf("restart %s: %w%s", svc, err, hint)
	}

	// Give the agent a moment to come up before polling is-active.
	time.Sleep(3 * time.Second)

	activeOut, _ := sess.runCapture(
		fmt.Sprintf(`printf 'BP_ACTIVE_%%s\n' "$(systemctl is-active %s 2>/dev/null || systemctl is-active %s.service 2>/dev/null || echo unknown)"`,
			shellSingleQuote(svc), shellSingleQuote(svc)),
		uartCommandTimeout,
	)
	active := strings.Contains(stripTerminalNoise(activeOut), "BP_ACTIVE_active")

	lifecycle := ""
	if bundle.Profile == ProfileSense {
		lifecycleOut, _ := sess.runCapture(
			`if test -f /run/vyooham-sense/lifecycle; then printf 'BP_LIFE_%s\n' "$(cat /run/vyooham-sense/lifecycle)"; else printf 'BP_LIFE_none\n'; fi`,
			uartCommandTimeout,
		)
		lifecycle = tokenSuffix(stripTerminalNoise(lifecycleOut), "BP_LIFE_")
		if lifecycle == "none" {
			lifecycle = ""
		}
	}

	idPath := bundle.EtcDir + "/identity.json"
	idOKOut, _ := sess.runCapture(
		fmt.Sprintf(`if test -f %s; then printf 'BP_ID_OK\n'; else printf 'BP_ID_MISSING\n'; fi`, shellSingleQuote(idPath)),
		uartCommandTimeout,
	)
	idOK := strings.Contains(stripTerminalNoise(idOKOut), "BP_ID_OK")

	msg := fmt.Sprintf("Credentials installed to %s via UART (%s)", bundle.EtcDir, bundle.Profile)
	if lifecycle != "" {
		msg = fmt.Sprintf("%s (lifecycle=%s)", msg, lifecycle)
	}
	if !idOK {
		msg = fmt.Sprintf("UART install finished but %s is missing — check console", idPath)
	} else if !active {
		msg = fmt.Sprintf("Install finished but %s is not active — check journalctl on UART", svc)
	} else if lifecycle != "" && lifecycle != "setup" && lifecycle != "claimed" {
		msg = fmt.Sprintf("%s is active; lifecycle=%s (expected setup after provision)", svc, lifecycle)
	}

	return &InstallResult{
		Profile:       string(bundle.Profile),
		AgentActive:   active,
		AgentChecked:  false,
		FFmpegOK:      true,
		Go2rtcOK:      true,
		MotionOK:      true,
		SetupServerOK: false, // LAN :4444 not required for UART factory path
		Message:       msg,
	}, nil
}

type uartSession struct {
	port serial.Port
	buf  *bytes.Buffer
}

// uartLoginPhase tracks how far loginRoot has progressed so credentials are
// sent at most once per prompt (getty may keep showing "login:"/"password:"
// across multiple UART polls).
type uartLoginPhase int

const (
	uartLoginAwaitUser uartLoginPhase = iota
	uartLoginAwaitPassword
	uartLoginAwaitShell
)

type uartLoginAction int

const (
	uartLoginWait uartLoginAction = iota
	uartLoginSendUser
	uartLoginSendPassword
	uartLoginSuccess
	uartLoginAuthFailed
)

// nextUARTLoginAction decides the next login step from buffered UART text.
// Credentials are only offered while still awaiting that prompt; after the
// password is sent, a fresh "login:" means auth failed rather than a resend.
func nextUARTLoginAction(text string, phase uartLoginPhase) uartLoginAction {
	lower := strings.ToLower(text)
	if strings.Contains(lower, "login incorrect") || strings.Contains(lower, "authentication failure") {
		return uartLoginAuthFailed
	}
	if looksLikeShellPrompt(text) {
		return uartLoginSuccess
	}
	switch phase {
	case uartLoginAwaitUser:
		if strings.Contains(lower, "login:") {
			return uartLoginSendUser
		}
		// Password-only getty (rare): accept password prompt without a prior login:
		if strings.Contains(lower, "password:") {
			return uartLoginSendPassword
		}
	case uartLoginAwaitPassword:
		if strings.Contains(lower, "password:") {
			return uartLoginSendPassword
		}
	case uartLoginAwaitShell:
		// Wrong password typically redisplays login: — do not resubmit root.
		if strings.Contains(lower, "login:") {
			return uartLoginAuthFailed
		}
	}
	return uartLoginWait
}

func (s *uartSession) loginRoot(password string) error {
	deadline := time.Now().Add(uartLoginTimeout)
	_ = s.port.ResetInputBuffer()
	_ = s.port.ResetOutputBuffer()
	s.buf.Reset()

	// Wake getty / interrupt boot chatter.
	_, _ = s.port.Write([]byte("\r\n"))
	time.Sleep(300 * time.Millisecond)

	phase := uartLoginAwaitUser
	for time.Now().Before(deadline) {
		chunk, err := s.readAvailable()
		if len(chunk) > 0 {
			s.buf.Write(chunk)
		}
		text := s.buf.String()

		switch nextUARTLoginAction(text, phase) {
		case uartLoginSendUser:
			s.buf.Reset()
			phase = uartLoginAwaitPassword
			if err := s.writeLine("root"); err != nil {
				return err
			}
			time.Sleep(400 * time.Millisecond)
		case uartLoginSendPassword:
			s.buf.Reset()
			phase = uartLoginAwaitShell
			// Empty string is valid for lab empty-root images (send bare newline).
			if err := s.writeLine(password); err != nil {
				return err
			}
			time.Sleep(500 * time.Millisecond)
		case uartLoginSuccess:
			// Clear leftover input and prove the shell with a real EXIT:0
			// round-trip. Don't look for a magic token mid-stream — Sense getty
			// glues printf output onto the prompt line after OSC stripping
			// (e.g. `root@host:~# UART_OK`), which breaks line-anchored checks.
			s.buf.Reset()
			_ = s.port.ResetInputBuffer()
			if err := s.runChecked("stty -echo 2>/dev/null || true", 15*time.Second); err != nil {
				return fmt.Errorf("shell not responding after login: %w", err)
			}
			return nil
		case uartLoginAuthFailed:
			return fmt.Errorf("login failed — check root password")
		}

		if err != nil && err != io.EOF {
			// Read timeout is normal while waiting for prompts.
			if !isTimeoutErr(err) {
				return err
			}
		}
		// Nudge again if silent.
		if s.buf.Len() == 0 {
			_, _ = s.port.Write([]byte("\r\n"))
		}
		time.Sleep(200 * time.Millisecond)
	}
	return fmt.Errorf("timed out waiting for root shell on UART (open screen at 115200 and confirm getty)")
}

func looksLikeShellPrompt(s string) bool {
	// Sense getty appends OSC shell-integration sequences immediately after
	// the prompt character (e.g. `root@host:~#` + ESC]633;…BEL), so strip
	// before checking line endings.
	s = stripTerminalNoise(s)
	s = strings.TrimSpace(s)
	if s == "" {
		return false
	}
	lines := strings.Split(s, "\n")
	last := strings.TrimSpace(lines[len(lines)-1])
	last = strings.TrimSuffix(last, "\r")
	if last == "" && len(lines) > 1 {
		last = strings.TrimSpace(lines[len(lines)-2])
	}
	return strings.HasSuffix(last, "#") || strings.HasSuffix(last, "$") || strings.HasSuffix(last, "# ") || strings.HasSuffix(last, "$ ")
}

func (s *uartSession) writeFileBase64(etcDir string, f CredentialFile) error {
	tmp := "/tmp/bp-" + f.Name
	tmpRaw := tmp + ".raw"
	dest := etcDir + "/" + f.Name
	b64 := base64.StdEncoding.EncodeToString(f.Data)

	// Credential files are small (≤ a few KB). Prefer one UART round-trip so
	// install is not minutes of 512-byte printf chunks.
	const maxOneShot = 6000
	if len(b64) <= maxOneShot {
		cmd := fmt.Sprintf(
			"printf '%%s' %s | base64 -d > %s && install -m %s -o root -g root %s %s && rm -f %s",
			shellSingleQuote(b64),
			shellSingleQuote(tmpRaw),
			f.Mode,
			shellSingleQuote(tmpRaw),
			shellSingleQuote(dest),
			shellSingleQuote(tmpRaw),
		)
		return s.runChecked(cmd, uartCommandTimeout)
	}

	if err := s.runChecked(fmt.Sprintf(": > %s", shellSingleQuote(tmp)), uartCommandTimeout); err != nil {
		return err
	}
	for i := 0; i < len(b64); i += uartChunkSize {
		end := i + uartChunkSize
		if end > len(b64) {
			end = len(b64)
		}
		chunk := b64[i:end]
		cmd := fmt.Sprintf("printf '%%s' %s >> %s", shellSingleQuote(chunk), shellSingleQuote(tmp))
		if err := s.runChecked(cmd, uartCommandTimeout); err != nil {
			return fmt.Errorf("chunk at %d: %w", i, err)
		}
	}
	install := fmt.Sprintf(
		"base64 -d %s > %s && install -m %s -o root -g root %s %s && rm -f %s %s",
		shellSingleQuote(tmp),
		shellSingleQuote(tmpRaw),
		f.Mode,
		shellSingleQuote(tmpRaw),
		shellSingleQuote(dest),
		shellSingleQuote(tmp),
		shellSingleQuote(tmpRaw),
	)
	return s.runChecked(install, uartCommandTimeout)
}

func (s *uartSession) runChecked(cmd string, timeout time.Duration) error {
	// Print EXIT and the end marker on their own lines. Do not put the marker
	// token only after `echo MARKER` in a way we accept mid-line — see runCapture.
	out, err := s.runCapture(cmd+`; printf 'EXIT:%s\n' "$?"`, timeout)
	if err != nil {
		return err
	}
	code, ok := parseExitCode(out)
	if !ok {
		return fmt.Errorf("command failed (no EXIT status in UART output): %s", summarizeUARTOut(out))
	}
	if code != 0 {
		return fmt.Errorf("command failed (exit %d): %s", code, summarizeUARTOut(out))
	}
	return nil
}

func (s *uartSession) runCapture(cmd string, timeout time.Duration) (string, error) {
	marker := fmt.Sprintf("BP_END_%d", time.Now().UnixNano())
	// Marker must be printed as its own line. Matching is line-anchored so a
	// locally-echoed "… echo BP_END_…" typed line is ignored until the real
	// printf output arrives after the command finishes.
	wrapped := fmt.Sprintf("%s; printf '%%s\\n' %s", cmd, shellSingleQuote(marker))
	s.buf.Reset()
	_ = s.port.ResetInputBuffer()
	if err := s.writeLine(wrapped); err != nil {
		return "", err
	}
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		chunk, err := s.readAvailable()
		if len(chunk) > 0 {
			s.buf.Write(chunk)
		}
		// Strip OSC/CSI before marker search: Sense getty often injects
		// shell-integration sequences immediately before printf output on the
		// same line, so a raw line-anchored match never sees BP_END_*.
		cleaned := stripTerminalNoise(s.buf.String())
		if idx := lineAnchoredIndex(cleaned, marker); idx >= 0 {
			return cleaned[:idx], nil
		}
		if err != nil && !isTimeoutErr(err) && err != io.EOF {
			return stripTerminalNoise(s.buf.String()), err
		}
		time.Sleep(50 * time.Millisecond)
	}
	return stripTerminalNoise(s.buf.String()), fmt.Errorf("timed out waiting for command marker")
}

func (s *uartSession) writeLine(line string) error {
	_, err := s.port.Write([]byte(line + "\n"))
	return err
}

func (s *uartSession) readAvailable() ([]byte, error) {
	buf := make([]byte, 1024)
	n, err := s.port.Read(buf)
	if n > 0 {
		return buf[:n], err
	}
	return nil, err
}

// lineAnchoredIndex finds token only at the start of the buffer or right after
// a CR/LF. That way an echoed shell command line containing the token mid-line
// is not treated as command completion.
func lineAnchoredIndex(text, token string) int {
	if token == "" {
		return -1
	}
	for offset := 0; offset <= len(text); {
		rel := strings.Index(text[offset:], token)
		if rel < 0 {
			return -1
		}
		abs := offset + rel
		if abs == 0 || text[abs-1] == '\n' || text[abs-1] == '\r' {
			return abs
		}
		offset = abs + 1
	}
	return -1
}

// oscSequenceRE matches ECMA-48 OSC sequences (BEL or ST terminated), including
// systemd/VTE shell-integration noise like ]3008;start=… that Sense getty emits.
var oscSequenceRE = regexp.MustCompile("\x1b\\][^\x07\x1b]*(?:\x07|\x1b\\\\)")

// csiSequenceRE matches common CSI / ESC styling sequences.
var csiSequenceRE = regexp.MustCompile("\x1b\\[[0-9;?]*[A-Za-z]")

var exitCodeRE = regexp.MustCompile(`EXIT:(-?\d+)`)

// stripTerminalNoise removes OSC/CSI escape sequences so markers and EXIT:
// lines can be parsed from interactive UART shells.
func stripTerminalNoise(s string) string {
	s = oscSequenceRE.ReplaceAllString(s, "")
	s = csiSequenceRE.ReplaceAllString(s, "")
	// Lone ESC leftovers.
	s = strings.ReplaceAll(s, "\x1b", "")
	return s
}

// parseExitCode finds the last EXIT:N token in UART command output.
// Sense getty embeds OSC shell-integration sequences on the same line as
// EXIT:0, so a line-prefix check is not enough.
func parseExitCode(out string) (code int, ok bool) {
	cleaned := stripTerminalNoise(out)
	matches := exitCodeRE.FindAllStringSubmatch(cleaned, -1)
	if len(matches) == 0 {
		return 0, false
	}
	last := matches[len(matches)-1]
	var n int
	if _, err := fmt.Sscanf(last[1], "%d", &n); err != nil {
		return 0, false
	}
	return n, true
}

func summarizeUARTOut(out string) string {
	out = strings.TrimSpace(stripTerminalNoise(out))
	// Collapse whitespace so OSC leftovers don't dominate the UI.
	out = strings.Join(strings.Fields(out), " ")
	if len(out) > 240 {
		return out[len(out)-240:]
	}
	return out
}

// tokenSuffix returns the text after the last occurrence of prefix in s.
func tokenSuffix(s, prefix string) string {
	idx := strings.LastIndex(s, prefix)
	if idx < 0 {
		return ""
	}
	rest := s[idx+len(prefix):]
	rest = strings.TrimSpace(rest)
	if i := strings.IndexAny(rest, "\r\n \t"); i >= 0 {
		rest = rest[:i]
	}
	return rest
}

func isTimeoutErr(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "timeout") || strings.Contains(msg, "deadline")
}

// WriteFileBase64Script is exported for tests — builds the install shell for one file.
func WriteFileBase64Script(etcDir string, f CredentialFile) string {
	tmp := "/tmp/bp-" + f.Name
	tmpRaw := tmp + ".raw"
	dest := etcDir + "/" + f.Name
	b64 := base64.StdEncoding.EncodeToString(f.Data)
	var b strings.Builder
	fmt.Fprintf(&b, ": > %s\n", shellSingleQuote(tmp))
	for i := 0; i < len(b64); i += uartChunkSize {
		end := i + uartChunkSize
		if end > len(b64) {
			end = len(b64)
		}
		fmt.Fprintf(&b, "printf '%%s' %s >> %s\n", shellSingleQuote(b64[i:end]), shellSingleQuote(tmp))
	}
	fmt.Fprintf(&b,
		"base64 -d %s > %s && install -m %s -o root -g root %s %s && rm -f %s %s\n",
		shellSingleQuote(tmp), shellSingleQuote(tmpRaw), f.Mode, shellSingleQuote(tmpRaw),
		shellSingleQuote(dest), shellSingleQuote(tmp), shellSingleQuote(tmpRaw),
	)
	return b.String()
}
