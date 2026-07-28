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
)

// Known registry family IDs (dev seed + production).
const (
	FamilyDoorbells = "df_door0001"
	FamilyNVR       = "df_nvr0001"
)

// Capability IDs used as fallback when dfid is unfamiliar.
const (
	CapCamera             = "cap_cam00001"
	CapMultichannelRecord = "cap_mcr00012"
	CapLANRelay           = "cap_lan00014"
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
	if _, ok := caps[CapLANRelay]; ok {
		return ProfileNVR, nil
	}

	name := strings.TrimSpace(dt.FriendlyName)
	if name == "" {
		name = strings.TrimSpace(dt.DTID)
	}
	return "", fmt.Errorf("unsupported install profile for device type %q (dfid=%q) — expected doorbell (%s) or NVR (%s) family",
		name, dfid, FamilyDoorbells, FamilyNVR)
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
