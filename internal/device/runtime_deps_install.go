package device

import "fmt"

// InstallRuntimeDeps installs profile-specific OS dependencies over SSH.
// profile must be "doorbell" or "nvr" (NVR installs no camera deps).
func InstallRuntimeDeps(cfg SSHConfig, profile InstallProfile) error {
	client, err := dialSSH(cfg)
	if err != nil {
		return err
	}
	defer client.Close()

	switch profile {
	case ProfileNVR:
		return nil
	case ProfileDoorbell, "":
		if err := installDoorbellRuntimeDeps(client, cfg.Password); err != nil {
			return err
		}
		ffmpegOK, go2rtcOK, motionOK := verifyDoorbellRuntimeDeps(client)
		if !ffmpegOK || !go2rtcOK || !motionOK {
			return fmt.Errorf("verification failed: ffmpeg=%v go2rtc=%v motion=%v", ffmpegOK, go2rtcOK, motionOK)
		}
		return nil
	default:
		return fmt.Errorf("unknown install profile %q — use doorbell or nvr", profile)
	}
}
