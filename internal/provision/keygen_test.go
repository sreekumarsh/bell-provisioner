package provision_test

import (
	"testing"

	"bell-provisioner/internal/provision"
)

func TestGenerateRSA2048(t *testing.T) {
	kp, err := provision.GenerateRSA2048()
	if err != nil {
		t.Fatal(err)
	}
	if len(kp.PrivateKeyPEM) == 0 {
		t.Fatal("expected private key PEM")
	}
	if kp.PublicKeyPEM == "" {
		t.Fatal("expected public key PEM")
	}
}
