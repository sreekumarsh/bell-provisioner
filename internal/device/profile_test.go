package device_test

import (
	"testing"

	"bell-provisioner/internal/device"
	"bell-provisioner/internal/gateway"
)

func TestResolveProfile_byFamily(t *testing.T) {
	p, err := device.ResolveProfile(gateway.DeviceType{DTID: "dt_wired0001", DFID: device.FamilyDoorbells})
	if err != nil || p != device.ProfileDoorbell {
		t.Fatalf("doorbell family: got %q err=%v", p, err)
	}
	p, err = device.ResolveProfile(gateway.DeviceType{DTID: "dt_nvr0001", DFID: device.FamilyNVR})
	if err != nil || p != device.ProfileNVR {
		t.Fatalf("nvr family: got %q err=%v", p, err)
	}
}

func TestResolveProfile_capFallback(t *testing.T) {
	p, err := device.ResolveProfile(gateway.DeviceType{
		DTID: "dt_outdoor", DFID: "df_unknown",
		Capabilities: []string{device.CapCamera},
	})
	if err != nil || p != device.ProfileDoorbell {
		t.Fatalf("cap_cam: got %q err=%v", p, err)
	}
	p, err = device.ResolveProfile(gateway.DeviceType{
		DTID: "dt_nvr_x", DFID: "df_unknown",
		Capabilities: []string{device.CapMultichannelRecord},
	})
	if err != nil || p != device.ProfileNVR {
		t.Fatalf("cap_mcr: got %q err=%v", p, err)
	}
}

func TestResolveProfile_unsupported(t *testing.T) {
	_, err := device.ResolveProfile(gateway.DeviceType{DTID: "dt_x", DFID: "df_other"})
	if err == nil {
		t.Fatal("expected error for unknown family/caps")
	}
}

func TestProfileSpec_paths(t *testing.T) {
	d := device.ProfileDoorbell.Spec()
	if d.EtcDir != "/etc/doorbell" || d.ServiceName != "doorbell-agent" || !d.CameraDeps {
		t.Fatalf("doorbell spec: %+v", d)
	}
	n := device.ProfileNVR.Spec()
	if n.EtcDir != "/etc/vyooham" || n.ServiceName != "control-agent" || n.CameraDeps {
		t.Fatalf("nvr spec: %+v", n)
	}
}
