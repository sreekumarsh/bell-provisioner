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

func TestResolveProfile_sense(t *testing.T) {
	p, err := device.ResolveProfile(gateway.DeviceType{DTID: "dt_sense_v1", DFID: device.FamilySense})
	if err != nil || p != device.ProfileSense {
		t.Fatalf("sense family: got %q err=%v", p, err)
	}

	// dt_sense_v1 declares LAN relay as well; NPU must win, or a Sense box gets
	// installed as an NVR into /etc/vyooham.
	p, err = device.ResolveProfile(gateway.DeviceType{
		DTID: "dt_sense_v2", DFID: "df_unknown",
		Capabilities: []string{device.CapNPU, device.CapLANRelay},
	})
	if err != nil || p != device.ProfileSense {
		t.Fatalf("npu+lan fallback: got %q err=%v", p, err)
	}

	// An NVR-ish type with LAN relay but no NPU still resolves to NVR.
	p, err = device.ResolveProfile(gateway.DeviceType{
		DTID: "dt_nvr_y", DFID: "df_unknown",
		Capabilities: []string{device.CapLANRelay},
	})
	if err != nil || p != device.ProfileNVR {
		t.Fatalf("lan-only fallback: got %q err=%v", p, err)
	}
}

func TestProfileSpec_sensePaths(t *testing.T) {
	s := device.ProfileSense.Spec()
	if s.EtcDir != "/etc/vyooham-sense" {
		t.Errorf("sense EtcDir = %q", s.EtcDir)
	}
	if s.EnvFileName != "control-agent.env" {
		t.Errorf("sense EnvFileName = %q (unit reads control-agent.env)", s.EnvFileName)
	}
	if s.ServiceName != "control-agent" || s.BinaryName != "control-agent" || s.CameraDeps {
		t.Errorf("sense spec: %+v", s)
	}
	if s.EtcDir == device.ProfileNVR.Spec().EtcDir {
		t.Error("sense must not share the NVR's identity dir")
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
