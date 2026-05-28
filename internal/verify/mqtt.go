package verify

import (
	"fmt"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// Result reports MQTT connectivity check.
type Result struct {
	Connected bool   `json:"connected"`
	Message   string `json:"message"`
}

// TestMQTTConnection subscribes briefly to confirm broker auth for the device.
func TestMQTTConnection(brokerURL, username, password, deviceID string) *Result {
	topic := fmt.Sprintf("devices/%s/status", deviceID)
	done := make(chan error, 1)

	opts := mqtt.NewClientOptions().
		AddBroker(brokerURL).
		SetUsername(username).
		SetPassword(password).
		SetClientID("bell-provisioner-" + deviceID).
		SetConnectTimeout(10 * time.Second).
		SetAutoReconnect(false)

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
