package device

import (
	"strings"
	"testing"

	"bell-provisioner/internal/config"
)

func TestEnvEnablesMTLS(t *testing.T) {
	cases := []struct {
		name string
		env  string
		want bool
	}{
		{"plain", "MQTT_TLS_ENABLED=false\n", false},
		{"true", "MQTT_TLS_ENABLED=true\n", true},
		{"mixed case", "MQTT_TLS_ENABLED=True\n", true},
		{"quoted", "MQTT_TLS_ENABLED=\"true\"\n", true},
		{"commented out", "# MQTT_TLS_ENABLED=true\nMQTT_TLS_ENABLED=false\n", false},
		{"last wins", "MQTT_TLS_ENABLED=false\nMQTT_TLS_ENABLED=true\n", true},
		{"absent", "LOG_LEVEL=info\n", false},
		{"substring only", "MQTT_TLS_ENABLED_EXTRA=true\n", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := envEnablesMTLS(tc.env); got != tc.want {
				t.Fatalf("envEnablesMTLS(%q) = %v, want %v", tc.env, got, tc.want)
			}
		})
	}
}

func TestVerifyMTLSMaterial_failsLoudlyWhenCertsMissing(t *testing.T) {
	env := "MQTT_BROKER_URL=ssl://mqtt.vyooham.com:8883\nMQTT_TLS_ENABLED=true\n"

	err := verifyMTLSMaterial(ProfileSense, env, nil, nil)
	if err == nil {
		t.Fatal("expected an error when TLS is on and both certs are missing")
	}
	for _, want := range []string{"device_crt", "ca_crt", "MQTT_CA_CERT_FILE", "ssl://mqtt.vyooham.com:8883"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error should mention %q, got: %v", want, err)
		}
	}

	if err := verifyMTLSMaterial(ProfileSense, env, []byte("cert"), nil); err == nil {
		t.Fatal("expected an error when only ca_crt is missing")
	} else if !strings.Contains(err.Error(), "carried no ca_crt") {
		t.Errorf("should name only the missing ca_crt, got: %v", err)
	}
}

func TestVerifyMTLSMaterial_allowsPlainAndCompleteTLS(t *testing.T) {
	plain := "MQTT_BROKER_URL=tcp://api.vyooham.com:1883\nMQTT_TLS_ENABLED=false\n"
	if err := verifyMTLSMaterial(ProfileSense, plain, nil, nil); err != nil {
		t.Fatalf("plain 1883 needs no certs: %v", err)
	}

	tls := "MQTT_TLS_ENABLED=true\n"
	if err := verifyMTLSMaterial(ProfileSense, tls, []byte("cert"), []byte("ca")); err != nil {
		t.Fatalf("TLS with both certs should pass: %v", err)
	}
}

// The generator and the guard must agree: the env config actually writes for a
// Sense box on mTLS has to be the thing the guard rejects when certs are absent.
// Testing them separately would let the two drift apart silently.
func TestVerifyMTLSMaterial_matchesGeneratedSenseEnv(t *testing.T) {
	mtlsEnv := config.SenseControlAgentEnv(config.ProfileVPS, "", config.MQTTMutualTLS)
	if err := verifyMTLSMaterial(ProfileSense, mtlsEnv, nil, nil); err == nil {
		t.Fatal("generated mTLS Sense env with no certs must be rejected")
	}
	if err := verifyMTLSMaterial(ProfileSense, mtlsEnv, []byte("cert"), []byte("ca")); err != nil {
		t.Fatalf("generated mTLS Sense env with certs must pass: %v", err)
	}

	plainEnv := config.SenseControlAgentEnv(config.ProfileVPS, "", config.MQTTPlain)
	if err := verifyMTLSMaterial(ProfileSense, plainEnv, nil, nil); err != nil {
		t.Fatalf("generated plain Sense env needs no certs: %v", err)
	}

	// Default Sense transport is mTLS — install must refuse without certs.
	defEnv := config.SenseControlAgentEnv(config.ProfileVPS, "", "")
	if err := verifyMTLSMaterial(ProfileSense, defEnv, nil, nil); err == nil {
		t.Fatal("DefaultSenseTransport (mtls) must refuse without device_crt/ca_crt")
	}
}

// The guard must fire before the SSH dial, so a doomed install never half-writes
// credentials to a real box.
func TestInstallCredentials_abortsBeforeSSHDial(t *testing.T) {
	mtlsEnv := config.SenseControlAgentEnv(config.ProfileVPS, "", config.MQTTMutualTLS)

	// 192.0.2.0/24 is TEST-NET-1 (RFC 5737) — guaranteed unroutable. If the
	// guard did not short-circuit, this would fail with a network error instead.
	_, err := InstallCredentials(
		SSHConfig{Host: "192.0.2.1", Port: 22, User: "root", Password: "x"},
		ProfileSense, []byte("key"), []byte(`{"device_id":"x"}`), nil, nil, testClaimGrantPub(t),
		mtlsEnv, true, AgentInstallOptions{},
	)
	if err == nil {
		t.Fatal("expected install to abort")
	}
	if !strings.Contains(err.Error(), "MQTT_TLS_ENABLED=true") {
		t.Fatalf("expected the mTLS guard to fire before dialing, got: %v", err)
	}
}
