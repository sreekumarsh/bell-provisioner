package agentrelease

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"
)

const githubAPI = "https://api.github.com"

// FetchLatest downloads the newest Actions artifact matching name from owner/repo.
func FetchLatest(ctx context.Context, owner, repo, token, artifactName string) (*Bundle, error) {
	if strings.TrimSpace(owner) == "" || strings.TrimSpace(repo) == "" {
		return nil, fmt.Errorf("github owner and repo are required")
	}
	if strings.TrimSpace(artifactName) == "" {
		artifactName = "doorbell-agent-linux-arm64"
	}

	id, err := latestArtifactID(ctx, owner, repo, token, artifactName)
	if err != nil {
		return nil, err
	}

	zipData, err := downloadArtifactZip(ctx, owner, repo, token, id)
	if err != nil {
		return nil, err
	}
	return extractBundle(zipData)
}

type artifactsListResponse struct {
	Artifacts []struct {
		ID        int64  `json:"id"`
		Name      string `json:"name"`
		CreatedAt string `json:"created_at"`
	} `json:"artifacts"`
}

func latestArtifactID(ctx context.Context, owner, repo, token, name string) (int64, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/actions/artifacts?name=%s&per_page=30",
		githubAPI, owner, repo, name)
	var list artifactsListResponse
	if err := githubGET(ctx, url, token, &list); err != nil {
		return 0, err
	}
	if len(list.Artifacts) == 0 {
		return 0, fmt.Errorf("no artifact named %q found — run the Agent workflow on pi-streamer main", name)
	}
	sort.Slice(list.Artifacts, func(i, j int) bool {
		return list.Artifacts[i].CreatedAt > list.Artifacts[j].CreatedAt
	})
	return list.Artifacts[0].ID, nil
}

func downloadArtifactZip(ctx context.Context, owner, repo, token string, artifactID int64) ([]byte, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/actions/artifacts/%d/zip", githubAPI, owner, repo, artifactID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	setGitHubHeaders(req, token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, fmt.Errorf("download artifact: %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}
	return io.ReadAll(resp.Body)
}

func githubGET(ctx context.Context, url, token string, dest any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	setGitHubHeaders(req, token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("github api %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}
	return json.NewDecoder(resp.Body).Decode(dest)
}

func setGitHubHeaders(req *http.Request, token string) {
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	if strings.TrimSpace(token) != "" {
		req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(token))
	}
}

func extractBundle(zipData []byte) (*Bundle, error) {
	zr, err := zip.NewReader(bytes.NewReader(zipData), int64(len(zipData)))
	if err != nil {
		return nil, fmt.Errorf("read artifact zip: %w", err)
	}

	var tgz []byte
	for _, f := range zr.File {
		if f.Name == "doorbell-agent-linux-arm64.tar.gz" ||
			strings.HasSuffix(f.Name, "/doorbell-agent-linux-arm64.tar.gz") {
			rc, err := f.Open()
			if err != nil {
				return nil, err
			}
			tgz, err = io.ReadAll(rc)
			rc.Close()
			if err != nil {
				return nil, err
			}
			break
		}
	}
	if len(tgz) > 0 {
		if b, err := bundleFromTarGz(tgz); err == nil {
			return b, nil
		}
	}

	// Fallback: flat paths inside the Actions artifact zip.
	b := &Bundle{}
	for _, f := range zr.File {
		switch {
		case strings.HasSuffix(f.Name, "pi-release/doorbell-agent") && !strings.Contains(f.Name, ".service"):
			data, err := readZipEntry(f)
			if err != nil {
				return nil, err
			}
			b.Binary = data
		case strings.HasSuffix(f.Name, "doorbell-agent.service"):
			data, err := readZipEntry(f)
			if err != nil {
				return nil, err
			}
			b.ServiceUnit = data
		case strings.HasSuffix(f.Name, "pi-release/VERSION"):
			data, err := readZipEntry(f)
			if err != nil {
				return nil, err
			}
			b.Version = strings.TrimSpace(string(data))
		}
	}
	if len(b.Binary) == 0 {
		return nil, fmt.Errorf("artifact zip missing doorbell-agent binary")
	}
	if len(b.ServiceUnit) == 0 {
		return nil, fmt.Errorf("artifact zip missing doorbell-agent.service")
	}
	return b, nil
}

func bundleFromTarGz(tgz []byte) (*Bundle, error) {
	gzr, err := gzip.NewReader(bytes.NewReader(tgz))
	if err != nil {
		return nil, fmt.Errorf("gzip release tarball: %w", err)
	}
	defer gzr.Close()
	tr := tar.NewReader(gzr)
	b := &Bundle{}
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("read release tarball: %w", err)
		}
		if hdr.Typeflag != tar.TypeReg {
			continue
		}
		name := hdr.Name
		data, err := io.ReadAll(tr)
		if err != nil {
			return nil, err
		}
		switch {
		case strings.HasSuffix(name, "/doorbell-agent") || name == "doorbell-agent":
			b.Binary = data
		case strings.HasSuffix(name, "doorbell-agent.service"):
			b.ServiceUnit = data
		case strings.HasSuffix(name, "VERSION"):
			b.Version = strings.TrimSpace(string(data))
		}
	}
	if len(b.Binary) == 0 || len(b.ServiceUnit) == 0 {
		return nil, fmt.Errorf("release tarball missing binary or systemd unit")
	}
	return b, nil
}

func readZipEntry(f *zip.File) ([]byte, error) {
	rc, err := f.Open()
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	return io.ReadAll(rc)
}

// FetchLatestDefaultTimeout wraps FetchLatest with a 5-minute timeout.
func FetchLatestDefaultTimeout(owner, repo, token, artifactName string) (*Bundle, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	return FetchLatest(ctx, owner, repo, token, artifactName)
}
