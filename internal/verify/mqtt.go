package verify

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"strings"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// Result reports MQTT connectivity check.
type Result struct {
	Connected bool   `json:"connected"`
	Message   string `json:"message"`
}

// TLSMaterial is optional client mTLS material for ssl:// brokers.
type TLSMaterial struct {
	CertPEM []byte
	KeyPEM  []byte
	CAPEM   []byte
}

// TestMQTTConnection subscribes briefly to confirm broker auth for the device.
// When brokerURL is ssl:// or tls://, tlsMat supplies the client cert/key/CA
// (required for Sense mTLS verify).
func TestMQTTConnection(brokerURL, username, password, deviceID string, tlsMat *TLSMaterial) *Result {
	topic := fmt.Sprintf("devices/%s/status", deviceID)
	done := make(chan error, 1)

	opts := mqtt.NewClientOptions().
		AddBroker(brokerURL).
		SetUsername(username).
		SetPassword(password).
		SetClientID("bell-provisioner-" + deviceID).
		SetConnectTimeout(10 * time.Second).
		SetAutoReconnect(false)

	if isTLSBroker(brokerURL) {
		cfg, err := buildTLSConfig(tlsMat)
		if err != nil {
			return &Result{Connected: false, Message: err.Error()}
		}
		opts.SetTLSConfig(cfg)
	}

	opts.SetOnConnectHandler(func(c mqtt.Client) {
		token := c.Subscribe(topic, 0, nil)
		token.Wait()
		if token.Error() != nil {
			done <- token.Error()
			return
		}
		done <- nil
	})

	client := mqtt.NewClient(opts)
	token := client.Connect()
	if !token.WaitTimeout(15 * time.Second) {
		return &Result{Connected: false, Message: "MQTT connect timed out"}
	}
	if token.Error() != nil {
		return &Result{Connected: false, Message: token.Error().Error()}
	}

	select {
	case err := <-done:
		client.Disconnect(250)
		if err != nil {
			return &Result{Connected: false, Message: err.Error()}
		}
		return &Result{Connected: true, Message: "MQTT authentication succeeded"}
	case <-time.After(5 * time.Second):
		client.Disconnect(250)
		return &Result{Connected: true, Message: "MQTT connected (subscribe pending)"}
	}
}

func isTLSBroker(brokerURL string) bool {
	u := strings.ToLower(strings.TrimSpace(brokerURL))
	return strings.HasPrefix(u, "ssl://") || strings.HasPrefix(u, "tls://") || strings.HasPrefix(u, "mqtts://")
}

func buildTLSConfig(mat *TLSMaterial) (*tls.Config, error) {
	if mat == nil || len(mat.CertPEM) == 0 || len(mat.KeyPEM) == 0 {
		return nil, fmt.Errorf("mTLS verify needs device_crt and device.key from provision — re-provision after auth-service has MQTT_CA_CERT_FILE / MQTT_CA_KEY_FILE set")
	}
	cert, err := tls.X509KeyPair(mat.CertPEM, mat.KeyPEM)
	if err != nil {
		return nil, fmt.Errorf("parse device cert/key: %w", err)
	}
	cfg := &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12,
	}
	if len(mat.CAPEM) > 0 {
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM(mat.CAPEM) {
			return nil, fmt.Errorf("parse ca_crt from provision response")
		}
		cfg.RootCAs = pool
	}
	return cfg, nil
}
