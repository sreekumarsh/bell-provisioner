package device

import (
	"fmt"
	"strings"

	"bell-provisioner/internal/gateway"
)

// InstallProfile selects on-device paths, agent binary, and runtime deps.
type InstallProfile string

const (
	ProfileDoorbell InstallProfile = "doorbell"
	ProfileNVR      InstallProfile = "nvr"
	ProfileSense    InstallProfile = "sense"
)

// Known registry family IDs (dev seed + production).
const (
	FamilyDoorbells = "df_door0001"
	FamilyNVR       = "df_nvr0001"
	FamilySense     = "df_sense"
)

// Capability IDs used as fallback when dfid is unfamiliar.
const (
	CapCamera             = "cap_cam00001"
	CapMultichannelRecord = "cap_mcr00012"
	CapLANRelay           = "cap_lan00014"
	CapNPU                = "cap_npu00015"
)

// ProfileSpec holds install targets for a profile.
type ProfileSpec struct {
	Profile     InstallProfile
	EtcDir      string
	ServiceName string
	BinaryName  string
	EnvFileName string
	CameraDeps  bool
}

// Spec returns install targets for p.
func (p InstallProfile) Spec() ProfileSpec {
	switch p {
	case ProfileSense:
		// Sense keeps its own identity dir, separate from the NVR's, and its
		// systemd unit reads control-agent.env (not agent.env) — see
		// vyooham-sense/services/control-agent/control-agent.service.
		return ProfileSpec{
			Profile:     ProfileSense,
			EtcDir:      "/etc/vyooham-sense",
			ServiceName: "control-agent",
			BinaryName:  "control-agent",
			EnvFileName: "control-agent.env",
			CameraDeps:  false,
		}
	case ProfileNVR:
		return ProfileSpec{
			Profile:     ProfileNVR,
			EtcDir:      "/etc/vyooham",
			ServiceName: "control-agent",
			BinaryName:  "control-agent",
			EnvFileName: "agent.env",
			CameraDeps:  false,
		}
	default:
		return ProfileSpec{
			Profile:     ProfileDoorbell,
			EtcDir:      "/etc/doorbell",
			ServiceName: "doorbell-agent",
			BinaryName:  "doorbell-agent",
			EnvFileName: "agent.env",
			CameraDeps:  true,
		}
	}
}

// ResolveProfile maps a registry device type to an install profile by family,
// with capability fallback for future SKUs under the same product line.
func ResolveProfile(dt gateway.DeviceType) (InstallProfile, error) {
	dfid := strings.TrimSpace(dt.DFID)
	switch dfid {
	case FamilyDoorbells:
		return ProfileDoorbell, nil
	case FamilyNVR:
		return ProfileNVR, nil
	case FamilySense:
		return ProfileSense, nil
	}

	caps := make(map[string]struct{}, len(dt.Capabilities))
	for _, c := range dt.Capabilities {
		caps[strings.TrimSpace(c)] = struct{}{}
	}
	if _, ok := caps[CapCamera]; ok {
		return ProfileDoorbell, nil
	}
	if _, ok := caps[CapMultichannelRecord]; ok {
		return ProfileNVR, nil
	}
	// NPU must be tested before LAN relay: dt_sense_v1 carries both
	// cap_npu00015 and cap_lan00014, and the NVR has no NPU. Checking LAN
	// relay first would silently install a Sense box as an NVR, putting its
	// identity and mTLS material in /etc/vyooham where its agent never looks.
	if _, ok := caps[CapNPU]; ok {
		return ProfileSense, nil
	}
	if _, ok := caps[CapLANRelay]; ok {
		return ProfileNVR, nil
	}

	name := strings.TrimSpace(dt.FriendlyName)
	if name == "" {
		name = strings.TrimSpace(dt.DTID)
	}
	return "", fmt.Errorf("unsupported install profile for device type %q (dfid=%q) — expected doorbell (%s), NVR (%s), or Sense (%s) family",
		name, dfid, FamilyDoorbells, FamilyNVR, FamilySense)
}

// FindDeviceType returns the registry type with the given dtid.
func FindDeviceType(types []gateway.DeviceType, dtid string) (gateway.DeviceType, error) {
	dtid = strings.TrimSpace(dtid)
	if dtid == "" {
		return gateway.DeviceType{}, fmt.Errorf("dtid is empty in identity.json — provision the device first")
	}
	for _, t := range types {
		if strings.TrimSpace(t.DTID) == dtid {
			return t, nil
		}
	}
	return gateway.DeviceType{}, fmt.Errorf("dtid %q not in registry — apply registry or re-login", dtid)
}
