package nvrrelease

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"os"
	"path/filepath"
	"testing"
)

func TestExtractTarGz_skipsPAXGlobalHeader(t *testing.T) {
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)

	// Mimic GitHub: a leading pax_global_header regular entry, then the repo tree.
	writeTarFile(t, tw, "pax_global_header", []byte("pax"), 0644)
	writeTarDir(t, tw, "vyooham-vyooham-sense-abc123/")
	writeTarDir(t, tw, "vyooham-vyooham-sense-abc123/services/")
	writeTarDir(t, tw, "vyooham-vyooham-sense-abc123/services/control-agent/")
	writeTarFile(t, tw, "vyooham-vyooham-sense-abc123/services/control-agent/go.mod", []byte("module sense\n"), 0644)

	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}

	dest := t.TempDir()
	top, err := extractTarGz(buf.Bytes(), dest)
	if err != nil {
		t.Fatal(err)
	}
	if top != "vyooham-vyooham-sense-abc123" {
		t.Fatalf("topDir = %q, want repo root (not pax_global_header)", top)
	}
	mod := filepath.Join(dest, top, "services", "control-agent", "go.mod")
	if _, err := os.Stat(mod); err != nil {
		t.Fatalf("expected go.mod at %s: %v", mod, err)
	}
}

func writeTarDir(t *testing.T, tw *tar.Writer, name string) {
	t.Helper()
	if err := tw.WriteHeader(&tar.Header{Name: name, Mode: 0755, Typeflag: tar.TypeDir}); err != nil {
		t.Fatal(err)
	}
}

func writeTarFile(t *testing.T, tw *tar.Writer, name string, body []byte, mode int64) {
	t.Helper()
	hdr := &tar.Header{Name: name, Mode: mode, Size: int64(len(body)), Typeflag: tar.TypeReg}
	if err := tw.WriteHeader(hdr); err != nil {
		t.Fatal(err)
	}
	if _, err := tw.Write(body); err != nil {
		t.Fatal(err)
	}
}
