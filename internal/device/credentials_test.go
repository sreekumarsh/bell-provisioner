package device

import (
	"strings"
	"testing"
)

func TestBuildCredentialBundle_senseFiles(t *testing.T) {
	claim := testClaimGrantPub(t)
	bundle, err := BuildCredentialBundle(
		ProfileSense,
		[]byte("-----BEGIN PRIVATE KEY-----\nMII\n-----END PRIVATE KEY-----\n"),
		[]byte(`{"global_device_id":"gd","device_id":"1","dtid":"dt_sense_v1","claimed":false}`),
		[]byte("-----BEGIN CERTIFICATE-----\nCRT\n-----END CERTIFICATE-----\n"),
		[]byte("-----BEGIN CERTIFICATE-----\nCA\n-----END CERTIFICATE-----\n"),
		claim,
		"MQTT_URL=ssl://mqtt.vyooham.com:8883\nMQTT_TLS_ENABLED=true\n",
		true,
	)
	if err != nil {
		t.Fatal(err)
	}
	if bundle.EtcDir != "/etc/vyooham-sense" {
		t.Fatalf("etc dir: %s", bundle.EtcDir)
	}
	names := map[string]string{}
	for _, f := range bundle.Files {
		names[f.Name] = f.Mode
	}
	want := map[string]string{
		"device.key":         "600",
		"identity.json":      "600",
		"device.crt":         "644",
		"ca.crt":             "644",
		ClaimGrantFileName:   claimGrantMode,
		"control-agent.env":  "644",
	}
	for k, mode := range want {
		got, ok := names[k]
		if !ok {
			t.Fatalf("missing file %s", k)
		}
		if got != mode {
			t.Fatalf("%s mode=%s want %s", k, got, mode)
		}
	}
}

func TestBuildCredentialBundle_requiresClaimGrantForSense(t *testing.T) {
	_, err := BuildCredentialBundle(
		ProfileSense,
		[]byte("key"),
		[]byte(`{"global_device_id":"gd","device_id":"1","dtid":"dt_sense_v1","claimed":false}`),
		nil, nil, nil, "", false,
	)
	if err == nil {
		t.Fatal("expected claim-grant error")
	}
	if !strings.Contains(err.Error(), "claim-grant") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseExitCode(t *testing.T) {
	code, ok := parseExitCode("foo\nEXIT:0\n")
	if !ok || code != 0 {
		t.Fatalf("want 0,ok got %d,%v", code, ok)
	}
	code, ok = parseExitCode("EXIT:12")
	if !ok || code != 12 {
		t.Fatalf("want 12,ok got %d,%v", code, ok)
	}
	if _, ok := parseExitCode("nope"); ok {
		t.Fatal("want !ok")
	}
}

func TestParseExitCode_oscShellIntegration(t *testing.T) {
	// Real Sense getty noise: OSC 3008 wraps the prompt; EXIT:0 sits on the
	// same logical stream without a clean line prefix.
	out := "93282000\r \x1b]3008;start=9230c0e1-d90a-4af5-8c11-de63ab5b2e9e;machineid=e3f71ea7;user=root;hostname=vyooham-sense-mini;bootid=c6bad303;pid=00000000000000000366;type=command;cwd=/root\x07EXIT:0"
	code, ok := parseExitCode(out)
	if !ok || code != 0 {
		t.Fatalf("want EXIT:0 through OSC noise, got code=%d ok=%v", code, ok)
	}
}

func TestLineAnchoredIndex_ignoresEchoedCommand(t *testing.T) {
	marker := "BP_END_123"
	// Local echo of the typed line contains the marker mid-line.
	echoed := "root@host:~# mkdir -p /etc/x; printf 'EXIT:%s\\n' \"$?\"; printf '%s\\n' '" + marker + "'\r\n"
	if idx := lineAnchoredIndex(echoed, marker); idx >= 0 {
		t.Fatalf("must not match marker inside echoed command, got idx=%d", idx)
	}
	// Real completion: marker alone on a line after EXIT.
	real := echoed + "EXIT:0\r\n" + marker + "\r\n"
	idx := lineAnchoredIndex(real, marker)
	if idx < 0 {
		t.Fatal("expected line-anchored marker after command output")
	}
	before := real[:idx]
	code, ok := parseExitCode(before)
	if !ok || code != 0 {
		t.Fatalf("want EXIT:0 before marker, got %d ok=%v in %q", code, ok, before)
	}
}

func TestLooksLikeShellPrompt(t *testing.T) {
	if !looksLikeShellPrompt("root@vyooham-sense-mini:~#") {
		t.Fatal("expected prompt")
	}
	if looksLikeShellPrompt("login:") {
		t.Fatal("login is not a shell prompt")
	}
}

func TestWriteFileBase64Script_roundTripShape(t *testing.T) {
	script := WriteFileBase64Script("/etc/vyooham-sense", CredentialFile{
		Name: "identity.json",
		Mode: "600",
		Data: []byte(`{"ok":true}`),
	})
	if !strings.Contains(script, "base64 -d") {
		t.Fatal(script)
	}
	if !strings.Contains(script, "/etc/vyooham-sense/identity.json") {
		t.Fatal(script)
	}
}
