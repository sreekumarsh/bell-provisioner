package device

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"strings"
)

// ClaimGrantFileName is the on-device name of the backend's claim-grant verify
// key, installed into the profile's etc dir alongside identity.json.
//
// vyooham-sense's control-agent reads it from CLAIM_GRANT_PUBKEY_PATH (default
// /etc/vyooham-sense/claim-grant.pub) and refuses to serve the setup flow at
// all when it is missing: an unclaimed box that cannot verify a cloud-signed
// grant has no way to tell the owner's app from anything else on the LAN, so it
// fails closed rather than trusting what arrives. That makes this file a
// provisioning gate, not just a claiming one — a Sense box shipped without it is
// dead on arrival.
//
// Contract: bell-docs/architecture/security-hardening.md § Design 5.
const ClaimGrantFileName = "claim-grant.pub"

// claimGrantMode is the on-device file mode. Public key, world-readable —
// unlike device.key and identity.json, there is nothing secret in it.
const claimGrantMode = "644"

// RequiresClaimGrantKey reports whether p's agent refuses to claim without a
// provisioned claim-grant verify key.
//
// Sense only, for now. Camera and NVR still accept the unsigned
// /setup/complete body — they have installed bases and need a transition period
// Sense did not, so requiring the key there would brick claiming on the fleet.
func RequiresClaimGrantKey(p InstallProfile) bool {
	return p == ProfileSense
}

// ParseClaimGrantPub validates a claim-grant verify key before it is shipped.
//
// The file is a PEM *bundle* — several PUBLIC KEY / RSA PUBLIC KEY blocks in
// one file — so a backend signing-key rotation can publish old+new for an
// overlap window with no device-side change. Unrelated block types are ignored
// and RSA is the only accepted algorithm, mirroring claimgrant.LoadPublicKeys on
// the device so this tool rejects exactly what the agent would.
func ParseClaimGrantPub(pemBytes []byte) ([]*rsa.PublicKey, error) {
	var keys []*rsa.PublicKey
	rest := pemBytes
	for {
		var block *pem.Block
		block, rest = pem.Decode(rest)
		if block == nil {
			break
		}
		switch block.Type {
		case "PUBLIC KEY":
			parsed, err := x509.ParsePKIXPublicKey(block.Bytes)
			if err != nil {
				return nil, fmt.Errorf("parse PKIX public key: %w", err)
			}
			key, ok := parsed.(*rsa.PublicKey)
			if !ok {
				return nil, fmt.Errorf("claim-grant key is %T, want RSA", parsed)
			}
			keys = append(keys, key)
		case "RSA PUBLIC KEY":
			key, err := x509.ParsePKCS1PublicKey(block.Bytes)
			if err != nil {
				return nil, fmt.Errorf("parse PKCS1 public key: %w", err)
			}
			keys = append(keys, key)
		}
	}
	if len(keys) == 0 {
		return nil, fmt.Errorf("no PUBLIC KEY or RSA PUBLIC KEY block found")
	}
	return keys, nil
}

// ResolveClaimGrantPub reads the claim-grant verify key from the operator's
// configured PEM file.
//
// This is a local file rather than something the provision response carries
// because bell-auth-service never publishes the public half over its API: the
// signing key lives in a KMS in production (CLAIM_GRANT_KEY_FILE only in dev)
// and `Minter.PublicKeyPEM` has no HTTP surface. Getting the public key onto
// the factory bench is deliberately an out-of-band operator step, so this tool
// has to be pointed at the file.
func ResolveClaimGrantPub(localPath string) ([]byte, error) {
	path := strings.TrimSpace(localPath)
	if path == "" {
		return nil, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read claim-grant key %s (config sense_claim_grant_pub_path): %w", path, err)
	}
	return data, nil
}

// verifyClaimGrantMaterial fails when a profile that cannot claim without a
// claim-grant verify key is about to be installed without one.
//
// Same intent as verifyMTLSMaterial: refuse before SSH rather than hand back a
// box that looks installed and can never be claimed.
func verifyClaimGrantMaterial(profile InstallProfile, claimGrantPubPEM []byte) error {
	if !RequiresClaimGrantKey(profile) {
		return nil
	}
	spec := profile.Spec()
	target := spec.EtcDir + "/" + ClaimGrantFileName

	if len(claimGrantPubPEM) == 0 {
		return fmt.Errorf(
			"refusing to install %s: no claim-grant verify key configured — "+
				"control-agent refuses to serve the claim flow without %s, so this box would be permanently unclaimable. "+
				"Set sense_claim_grant_pub_path in config.json to the public half of bell-auth-service's claim-grant "+
				"signing key — the same key its CLAIM_GRANT_KID names, from keys/generate-claim-grant.sh in dev or "+
				"exported from KMS in production. It must match, or grants this backend signs will not verify on the box",
			profile, target,
		)
	}
	if _, err := ParseClaimGrantPub(claimGrantPubPEM); err != nil {
		return fmt.Errorf(
			"refusing to install %s: claim-grant key destined for %s is not a usable PEM bundle (%w) — "+
				"control-agent would fail to load it and refuse to serve the claim flow. "+
				"Expect one or more PUBLIC KEY / RSA PUBLIC KEY blocks",
			profile, target, err,
		)
	}
	return nil
}
