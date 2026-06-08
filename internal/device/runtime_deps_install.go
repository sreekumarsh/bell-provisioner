package device

import "fmt"

// InstallRuntimeDeps installs ffmpeg, v4l-utils, go2rtc, and motion assets on the Pi over SSH.
func InstallRuntimeDeps(cfg SSHConfig) error {
	client, err := dialSSH(cfg)
	if err != nil {
		return err
	}
	defer client.Close()

	if err := installAgentRuntimeDeps(client, cfg.Password); err != nil {
		return err
	}

	ffmpegOK, go2rtcOK, motionOK := verifyAgentRuntimeDeps(client)
	if !ffmpegOK || !go2rtcOK || !motionOK {
		return fmt.Errorf("verification failed: ffmpeg=%v go2rtc=%v motion=%v", ffmpegOK, go2rtcOK, motionOK)
	}
	return nil
}
