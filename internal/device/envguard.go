package device

import (
	"bufio"
	"fmt"
	"strings"
)

// envValue returns the value of key in a KEY=VALUE env file body, ignoring
// blank lines and # comments. Last assignment wins, matching systemd's
// EnvironmentFile semantics.
func envValue(envContent, key string) (string, bool) {
	var val string
	var found bool
	sc := bufio.NewScanner(strings.NewReader(envContent))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		name, v, ok := strings.Cut(line, "=")
		if !ok || strings.TrimSpace(name) != key {
			continue
		}
		val = strings.Trim(strings.TrimSpace(v), `"'`)
		found = true
	}
	return val, found
}

// envEnablesMTLS reports whether an agent env body turns on MQTT TLS.
func envEnablesMTLS(envContent string) bool {
	v, ok := envValue(envContent, "MQTT_TLS_ENABLED")
	return ok && strings.EqualFold(v, "true")
}

// verifyMTLSMaterial fails when an agent env is about to be written with
// MQTT_TLS_ENABLED=true but the provision response carried no client cert or CA.
//
// Without this the tool happily ships a box that cannot connect to the broker at
// all: the agent would dial ssl://…:8883, find MQTT_TLS_CLIENT_CERT pointing at a
// file that was never installed, and fail its TLS handshake forever. auth-service
// omits device_crt/ca_crt (`omitempty`) whenever its MQTT CA signer is unset, so
// this is a silent, easy-to-hit condition — not a theoretical one.
func verifyMTLSMaterial(profile InstallProfile, envContent string, deviceCertPEM, caCertPEM []byte) error {
	if !envEnablesMTLS(envContent) {
		return nil
	}
	var missing []string
	if len(deviceCertPEM) == 0 {
		missing = append(missing, "device_crt")
	}
	if len(caCertPEM) == 0 {
		missing = append(missing, "ca_crt")
	}
	if len(missing) == 0 {
		return nil
	}

	broker, _ := envValue(envContent, "MQTT_BROKER_URL")
	if broker == "" {
		broker = "the TLS broker"
	}
	return fmt.Errorf(
		"refusing to install %s: agent env sets MQTT_TLS_ENABLED=true (broker %s) but the provision response carried no %s — "+
			"the device would dial the broker with no client certificate and never connect. "+
			"auth-service only returns device_crt/ca_crt when it runs with MQTT_CA_CERT_FILE and MQTT_CA_KEY_FILE set; "+
			"either deploy the CA-issuing config and re-provision this device, or provision it with the plain-1883 transport",
		profile, broker, strings.Join(missing, " and "),
	)
}
