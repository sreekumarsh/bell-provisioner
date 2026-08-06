package nvrrelease

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const (
	githubAPI          = "https://api.github.com"
	controlAgentRelDir = "services/control-agent"
	controlAgentPkg    = "./cmd/control-agent"
	defaultRef         = "main"
)

// Bundle is a linux/amd64 control-agent build plus systemd unit.
type Bundle struct {
	Version     string
	Binary      []byte
	ServiceUnit []byte
}

// BuildOptions configures a control-agent cross-compile.
type BuildOptions struct {
	Owner, Repo, Token, Ref string
	// GOARCH defaults to amd64 (the NVR's x86 box). Sense is RK3566 — arm64.
	GOARCH string
	// ServiceUnit is the systemd unit shipped with the binary. Each product
	// supplies its own because the unit's EnvironmentFile must match the env
	// path the provisioner writes for that profile.
	ServiceUnit string
}

// BuildLatestDefaultTimeout downloads the repo and cross-compiles control-agent.
func BuildLatestDefaultTimeout(owner, repo, token, ref string) (*Bundle, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	return BuildLatest(ctx, owner, repo, token, ref)
}

// BuildSenseDefaultTimeout builds the Sense control-agent (linux/arm64, RK3566).
func BuildSenseDefaultTimeout(owner, repo, token, ref string) (*Bundle, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	return Build(ctx, BuildOptions{
		Owner: owner, Repo: repo, Token: token, Ref: ref,
		GOARCH:      "arm64",
		ServiceUnit: SenseControlAgentServiceUnit,
	})
}

// BuildLatest fetches owner/repo at ref and builds linux/amd64 control-agent.
func BuildLatest(ctx context.Context, owner, repo, token, ref string) (*Bundle, error) {
	return Build(ctx, BuildOptions{Owner: owner, Repo: repo, Token: token, Ref: ref})
}

// Build fetches the repo at opts.Ref and cross-compiles control-agent.
func Build(ctx context.Context, opts BuildOptions) (*Bundle, error) {
	owner, repo, token := opts.Owner, opts.Repo, opts.Token
	ref := opts.Ref
	if strings.TrimSpace(owner) == "" || strings.TrimSpace(repo) == "" {
		return nil, fmt.Errorf("github owner and repo are required")
	}
	if strings.TrimSpace(ref) == "" {
		ref = defaultRef
	}
	goarch := strings.TrimSpace(opts.GOARCH)
	if goarch == "" {
		goarch = "amd64"
	}
	serviceUnit := opts.ServiceUnit
	if serviceUnit == "" {
		serviceUnit = ControlAgentServiceUnit
	}
	if _, err := exec.LookPath("go"); err != nil {
		return nil, fmt.Errorf("go toolchain required on Mac to build control-agent: %w", err)
	}

	tmpRoot, err := os.MkdirTemp("", "bell-nvr-build-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tmpRoot)

	tarball, err := downloadRepoTarball(ctx, owner, repo, token, ref)
	if err != nil {
		return nil, err
	}
	extractDir := filepath.Join(tmpRoot, "src")
	if err := os.MkdirAll(extractDir, 0755); err != nil {
		return nil, err
	}
	topDir, err := extractTarGz(tarball, extractDir)
	if err != nil {
		return nil, err
	}
	moduleDir := filepath.Join(extractDir, topDir, controlAgentRelDir)
	if _, err := os.Stat(filepath.Join(moduleDir, "go.mod")); err != nil {
		return nil, fmt.Errorf("control-agent module not found at %s: %w", moduleDir, err)
	}

	outBin := filepath.Join(tmpRoot, "control-agent")
	cmd := exec.CommandContext(ctx, "go", "build", "-o", outBin, controlAgentPkg)
	cmd.Dir = moduleDir
	cmd.Env = append(os.Environ(),
		"CGO_ENABLED=0",
		"GOOS=linux",
		"GOARCH="+goarch,
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("go build control-agent: %w\n%s", err, strings.TrimSpace(string(out)))
	}
	binary, err := os.ReadFile(outBin)
	if err != nil {
		return nil, err
	}

	return &Bundle{
		Version:     ref,
		Binary:      binary,
		ServiceUnit: []byte(serviceUnit),
	}, nil
}

func downloadRepoTarball(ctx context.Context, owner, repo, token, ref string) ([]byte, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/tarball/%s", githubAPI, owner, repo, ref)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	if strings.TrimSpace(token) != "" {
		req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(token))
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, fmt.Errorf("download %s/%s@%s: %s: %s", owner, repo, ref, resp.Status, strings.TrimSpace(string(body)))
	}
	return io.ReadAll(resp.Body)
}

func extractTarGz(data []byte, dest string) (topDir string, err error) {
	gz, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return "", fmt.Errorf("gzip: %w", err)
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}
		// GitHub repo tarballs often lead with a PAX global header entry named
		// "pax_global_header". That must not become the extracted top directory.
		if hdr.Typeflag == tar.TypeXHeader || hdr.Typeflag == tar.TypeXGlobalHeader {
			continue
		}
		name := hdr.Name
		if name == "" || strings.Contains(name, "..") || isTarMetaName(name) {
			continue
		}
		if topDir == "" {
			parts := strings.SplitN(name, "/", 2)
			if parts[0] != "" && !isTarMetaName(parts[0]) {
				topDir = parts[0]
			}
		}
		target := filepath.Join(dest, name)
		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0755); err != nil {
				return "", err
			}
		case tar.TypeReg, tar.TypeRegA:
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return "", err
			}
			f, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(hdr.Mode)&0777)
			if err != nil {
				return "", err
			}
			if _, err := io.Copy(f, tr); err != nil {
				f.Close()
				return "", err
			}
			f.Close()
		}
	}
	if topDir == "" {
		return "", fmt.Errorf("empty tarball")
	}
	return topDir, nil
}

// isTarMetaName reports archive metadata entries that are not part of the repo tree.
func isTarMetaName(name string) bool {
	base := strings.TrimSuffix(strings.TrimPrefix(name, "./"), "/")
	switch base {
	case "pax_global_header", "PaxHeader", ".git":
		return true
	}
	return strings.HasPrefix(base, "PaxHeader/")
}

// SenseControlAgentServiceUnit is the systemd unit deployed with the Sense
// control-agent. EnvironmentFile must stay in sync with ProfileSense's
// EtcDir/EnvFileName; it mirrors vyooham-sense's own control-agent.service.
const SenseControlAgentServiceUnit = `[Unit]
Description=Vyooham Sense cloud control agent (MQTT control-plane)
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
ExecStart=/usr/local/bin/control-agent
Restart=always
RestartSec=3
User=root
EnvironmentFile=-/etc/vyooham-sense/control-agent.env
StandardOutput=journal
StandardError=journal
StartLimitBurst=5
StartLimitIntervalSec=30

[Install]
WantedBy=multi-user.target
`

// ControlAgentServiceUnit is the systemd unit deployed with control-agent.
const ControlAgentServiceUnit = `[Unit]
Description=Vyooham NVR Control Agent
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
ExecStart=/usr/local/bin/control-agent
Restart=always
RestartSec=5
User=root
EnvironmentFile=-/etc/vyooham/agent.env
StandardOutput=journal
StandardError=journal
StartLimitBurst=5
StartLimitIntervalSec=30

[Install]
WantedBy=multi-user.target
`
