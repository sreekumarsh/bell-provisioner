package device_test

import (
	"encoding/json"
	"testing"

	"bell-provisioner/internal/device"
)

func TestBuildIdentityJSON_v2(t *testing.T) {
	data, err := device.BuildIdentityJSON(
		"dt_8f3k2m9x1p_00042", "00042", "dt_8f3k2m9x1p", "ds_7q2w9e4r",
		"DB-2605-0042", "1.1", "dt_8f3k2m9x1p_00042", "secret",
	)
	if err != nil {
		t.Fatal(err)
	}

	var id map[string]any
	if err := json.Unmarshal(data, &id); err != nil {
		t.Fatal(err)
	}
	if id["global_device_id"] != "dt_8f3k2m9x1p_00042" {
		t.Fatalf("global_device_id: %v", id["global_device_id"])
	}
	if id["device_id"] != "00042" {
		t.Fatalf("device_id: %v", id["device_id"])
	}
	if id["dtid"] != "dt_8f3k2m9x1p" {
		t.Fatalf("dtid: %v", id["dtid"])
	}
	if id["dsid"] != "ds_7q2w9e4r" {
		t.Fatalf("dsid: %v", id["dsid"])
	}
	if id["mqtt_username"] != "dt_8f3k2m9x1p_00042" {
		t.Fatalf("mqtt_username: %v", id["mqtt_username"])
	}
	if id["claimed"] != false {
		t.Fatalf("claimed: %v", id["claimed"])
	}
}
