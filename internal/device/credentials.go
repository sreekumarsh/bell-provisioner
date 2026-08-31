package device

import (
	"fmt"
)

// CredentialFile is one etc-dir file to install (SSH or UART).
type CredentialFile struct {
	Name string
	Mode string // octal string without leading 0, e.g. "600"
	Data []byte
}

// CredentialBundle is the credentials-only install set for a profile.
type CredentialBundle struct {
	Profile     InstallProfile
	EtcDir      string
	ServiceName string
	EnvFileName string
	Files       []CredentialFile
}

// BuildCredentialBundle validates Sense/mTLS gates and returns the files to
// write under the profile etc dir. It does not include agent binaries.
func BuildCredentialBundle(
	profile InstallProfile,
	privateKeyPEM, identityJSON, deviceCertPEM, caCertPEM, claimGrantPubPEM []byte,
	agentEnvContent string,
	deployAgentEnv bool,
) (CredentialBundle, error) {
	spec := profile.Spec()
	if deployAgentEnv {
		if err := verifyMTLSMaterial(profile, agentEnvContent, deviceCertPEM, caCertPEM); err != nil {
			return CredentialBundle{}, err
		}
	}
	if err := verifyClaimGrantMaterial(profile, claimGrantPubPEM); err != nil {
		return CredentialBundle{}, err
	}
	if _, err := ParseIdentity(identityJSON); err != nil {
		return CredentialBundle{}, err
	}
	if len(privateKeyPEM) == 0 {
		return CredentialBundle{}, fmt.Errorf("device private key is empty")
	}

	files := []CredentialFile{
		{Name: "device.key", Mode: "600", Data: privateKeyPEM},
		{Name: "identity.json", Mode: "600", Data: identityJSON},
	}
	if len(deviceCertPEM) > 0 {
		files = append(files, CredentialFile{Name: "device.crt", Mode: "644", Data: deviceCertPEM})
	}
	if len(caCertPEM) > 0 {
		files = append(files, CredentialFile{Name: "ca.crt", Mode: "644", Data: caCertPEM})
	}
	if len(claimGrantPubPEM) > 0 {
		files = append(files, CredentialFile{Name: ClaimGrantFileName, Mode: claimGrantMode, Data: claimGrantPubPEM})
	}
	if deployAgentEnv && agentEnvContent != "" {
		files = append(files, CredentialFile{Name: spec.EnvFileName, Mode: "644", Data: []byte(agentEnvContent)})
	}

	return CredentialBundle{
		Profile:     profile,
		EtcDir:      spec.EtcDir,
		ServiceName: spec.ServiceName,
		EnvFileName: spec.EnvFileName,
		Files:       files,
	}, nil
}
