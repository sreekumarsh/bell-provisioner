package device

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"bell-provisioner/internal/config"
)

// testClaimGrantPub returns a valid single-key claim-grant PEM bundle.
func testClaimGrantPub(t *testing.T) []byte {
	t.Helper()
	return claimGrantPEM(t, "PUBLIC KEY")
}

func claimGrantPEM(t *testing.T, blockType string) []byte {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	var der []byte
	switch blockType {
	case "PUBLIC KEY":
		der, err = x509.MarshalPKIXPublicKey(&key.PublicKey)
		if err != nil {
			t.Fatalf("marshal PKIX: %v", err)
		}
	case "RSA PUBLIC KEY":
		der = x509.MarshalPKCS1PublicKey(&key.PublicKey)
	default:
		t.Fatalf("unsupported block type %q", blockType)
	}
	return pem.EncodeToMemory(&pem.Block{Type: blockType, Bytes: der})
}

func TestParseClaimGrantPub_acceptsBundleAndBothBlockTypes(t *testing.T) {
	// A rotation ships old+new in one file; the device tries each, so the
	// provisioner must not reject a multi-block bundle.
	bundle := append(claimGrantPEM(t, "PUBLIC KEY"), claimGrantPEM(t, "RSA PUBLIC KEY")...)
	keys, err := ParseClaimGrantPub(bundle)
	if err != nil {
		t.Fatalf("bundle should parse: %v", err)
	}
	if len(keys) != 2 {
		t.Fatalf("got %d keys, want 2", len(keys))
	}

	// Unrelated blocks are ignored rather than fatal, matching the device.
	withCert := append([]byte("-----BEGIN CERTIFICATE-----\nAAAA\n-----END CERTIFICATE-----\n"), testClaimGrantPub(t)...)
	if _, err := ParseClaimGrantPub(withCert); err != nil {
		t.Fatalf("unrelated PEM blocks should be ignored: %v", err)
	}
}

func TestParseClaimGrantPub_rejectsUnusableInput(t *testing.T) {
	cases := map[string][]byte{
		"empty":          nil,
		"not PEM":        []byte("ssh-rsa AAAAB3Nza..."),
		"private key":    pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: []byte("nope")}),
		"corrupt PKIX":   pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: []byte("garbage")}),
		"only unrelated": []byte("-----BEGIN CERTIFICATE-----\nAAAA\n-----END CERTIFICATE-----\n"),
	}
	for name, in := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := ParseClaimGrantPub(in); err == nil {
				t.Fatal("expected an error")
			}
		})
	}
}

func TestParseClaimGrantPub_rejectsNonRSA(t *testing.T) {
	// The device's verifier is RSASSA-PKCS1v15 only; an ECDSA key would load
	// here and fail there, which is exactly the dead-on-arrival box the guard
	// exists to prevent.
	ec, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate ECDSA key: %v", err)
	}
	der, err := x509.MarshalPKIXPublicKey(&ec.PublicKey)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	block := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der})
	if _, err := ParseClaimGrantPub(block); err == nil {
		t.Fatal("expected non-RSA key to be rejected")
	}
}

func TestRequiresClaimGrantKey_senseOnly(t *testing.T) {
	// Camera and NVR still accept the unsigned /setup/complete body. Requiring
	// the key there would abort installs for products whose agents do not read
	// it yet.
	if !RequiresClaimGrantKey(ProfileSense) {
		t.Error("Sense must require a claim-grant key")
	}
	for _, p := range []InstallProfile{ProfileDoorbell, ProfileNVR} {
		if RequiresClaimGrantKey(p) {
			t.Errorf("%s must not require a claim-grant key yet", p)
		}
	}
}

func TestVerifyClaimGrantMaterial(t *testing.T) {
	valid := testClaimGrantPub(t)

	err := verifyClaimGrantMaterial(ProfileSense, nil)
	if err == nil {
		t.Fatal("Sense with no claim-grant key must abort")
	}
	for _, want := range []string{"/etc/vyooham-sense/claim-grant.pub", "sense_claim_grant_pub_path", "CLAIM_GRANT_KID"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error should mention %q, got: %v", want, err)
		}
	}

	if err := verifyClaimGrantMaterial(ProfileSense, []byte("not a pem file")); err == nil {
		t.Fatal("Sense with an unparseable key must abort")
	}
	if err := verifyClaimGrantMaterial(ProfileSense, valid); err != nil {
		t.Fatalf("Sense with a valid key must pass: %v", err)
	}

	// Doorbell and NVR install unchanged with or without the key.
	for _, p := range []InstallProfile{ProfileDoorbell, ProfileNVR} {
		if err := verifyClaimGrantMaterial(p, nil); err != nil {
			t.Errorf("%s must install without a claim-grant key: %v", p, err)
		}
	}
}

func TestResolveClaimGrantPub(t *testing.T) {
	dir := t.TempDir()
	local := filepath.Join(dir, ClaimGrantFileName)
	localPEM := testClaimGrantPub(t)
	if err := os.WriteFile(local, localPEM, 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	got, err := ResolveClaimGrantPub(local)
	if err != nil {
		t.Fatalf("resolve local: %v", err)
	}
	if string(got) != string(localPEM) {
		t.Error("configured key should be read verbatim")
	}

	// Unconfigured: no key, and the install guard is what reports why.
	got, err = ResolveClaimGrantPub("")
	if err != nil || got != nil {
		t.Fatalf("want (nil, nil), got (%q, %v)", got, err)
	}

	// A configured-but-missing file is an operator error worth naming, not a
	// silent fall-through to the generic "no key" message.
	if _, err := ResolveClaimGrantPub(filepath.Join(dir, "absent.pub")); err == nil {
		t.Fatal("missing configured file must error")
	} else if !strings.Contains(err.Error(), "sense_claim_grant_pub_path") {
		t.Errorf("error should name the config key, got: %v", err)
	}
}

// The Sense env the tool writes must point control-agent at the file the tool
// installs; a mismatch means the box loads no key and refuses to claim.
func TestSenseEnvClaimGrantPathMatchesInstallTarget(t *testing.T) {
	spec := ProfileSense.Spec()
	want := "CLAIM_GRANT_PUBKEY_PATH=" + spec.EtcDir + "/" + ClaimGrantFileName
	env := config.SenseControlAgentEnv(config.ProfileVPS, "", config.MQTTPlain)
	if !strings.Contains(env, want) {
		t.Fatalf("generated Sense env should contain %q, got:\n%s", want, env)
	}
}
