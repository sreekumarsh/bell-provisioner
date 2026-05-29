package agentrelease

import (
	"os"
	"testing"
)

func TestBundleFromTarGz(t *testing.T) {
	path := "/tmp/prov-art-test/doorbell-agent-linux-arm64.tar.gz"
	data, err := os.ReadFile(path)
	if err != nil {
		t.Skip(err)
	}
	b, err := bundleFromTarGz(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(b.Binary) < 1_000_000 || len(b.ServiceUnit) < 50 {
		t.Fatalf("unexpected sizes binary=%d service=%d", len(b.Binary), len(b.ServiceUnit))
	}
}

func TestExtractBundleFromArtifactZip(t *testing.T) {
	path := os.Getenv("PI_STREAMER_ARTIFACT_ZIP")
	if path == "" {
		path = "/tmp/prov-art-test/artifact.zip"
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Skip("artifact zip not available:", err)
	}
	b, err := extractBundle(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(b.Binary) < 1_000_000 {
		t.Fatalf("binary too small: %d bytes", len(b.Binary))
	}
	if len(b.ServiceUnit) < 50 {
		t.Fatalf("service unit too small: %d bytes", len(b.ServiceUnit))
	}
}
