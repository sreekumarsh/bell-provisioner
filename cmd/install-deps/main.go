package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"bell-provisioner/internal/device"
)

type appConfig struct {
	SSHUser     string `json:"ssh_user"`
	SSHHost     string `json:"ssh_host"`
	SSHPort     int    `json:"ssh_port"`
	SSHPassword string `json:"ssh_password"`
}

func main() {
	profileFlag := flag.String("profile", "doorbell", "install profile: doorbell | nvr")
	flag.Parse()

	profile := device.InstallProfile(*profileFlag)
	switch profile {
	case device.ProfileDoorbell, device.ProfileNVR:
	default:
		fmt.Fprintf(os.Stderr, "unknown --profile %q (use doorbell or nvr)\n", *profileFlag)
		os.Exit(2)
	}

	cfg, err := loadConfig()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	host := envOr("SSH_HOST", cfg.SSHHost)
	user := envOr("SSH_USER", cfg.SSHUser)
	pass := envOr("SSH_PASSWORD", cfg.SSHPassword)
	port := cfg.SSHPort
	if port == 0 {
		port = 22
	}

	fmt.Printf("Installing %s runtime deps on %s@%s:%d …\n", profile, user, host, port)
	if err := device.InstallRuntimeDeps(device.SSHConfig{
		Host:     host,
		Port:     port,
		User:     user,
		Password: pass,
	}, profile); err != nil {
		fmt.Fprintln(os.Stderr, "install failed:", err)
		os.Exit(1)
	}
	switch profile {
	case device.ProfileNVR:
		fmt.Println("NVR profile: no camera OS packages required for control-agent.")
	default:
		fmt.Println("All doorbell runtime dependencies installed and verified.")
		fmt.Println("  ffmpeg, v4l-utils, go2rtc, motion venv, yolov8n.onnx, motion-classify.py")
	}
}

func loadConfig() (appConfig, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return appConfig{}, err
	}
	path := filepath.Join(dir, "bell-provisioner", "config.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return appConfig{}, fmt.Errorf("read config %s: %w", path, err)
	}
	var cfg appConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return appConfig{}, err
	}
	return cfg, nil
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
