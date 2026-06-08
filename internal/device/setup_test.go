package device_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"bell-provisioner/internal/device"
)

func TestWaitForSetupServer(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/setup/identity", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]string{
			"device_id":     "00042",
			"serial_number": "DB-2605-0042",
			"hw_version":    "1.1",
		})
	})
	ts := httptest.NewServer(mux)
	defer ts.Close()

	host := ts.Listener.Addr().String()
	if !device.ProbeSetupServerForTest(host, "00042") {
		t.Fatal("expected setup server probe to succeed")
	}
	if device.ProbeSetupServerForTest(host, "99999") {
		t.Fatal("expected device_id mismatch to fail")
	}
}
