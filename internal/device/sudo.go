package device

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"

	"golang.org/x/crypto/ssh"
)

var sudoTokenRE = regexp.MustCompile(`(^|[\s;|&(])sudo(\s+)`)

func shellSingleQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'"'"'`) + "'"
}

func isRootSSHUser(user string) bool {
	return strings.EqualFold(strings.TrimSpace(user), "root")
}

// stripSudoTokens removes sudo invocations so a root shell can run the same scripts.
func stripSudoTokens(script string) string {
	return sudoTokenRE.ReplaceAllString(script, "$1")
}

// runSudo runs a shell script with root privileges on the device.
// When sshUser is root (typical for Sense), commands run directly — sudo is
// stripped and no sudo password prompt is used. Otherwise authenticates sudo
// once via stdin (-S).
func runSudo(client *ssh.Client, sshUser, password, script string) error {
	script = strings.TrimSpace(script)
	if isRootSSHUser(sshUser) {
		return runCmd(client, "set -e\n"+stripSudoTokens(script))
	}
	password = strings.TrimSpace(password)
	if password == "" {
		return fmt.Errorf("sudo password is required (same as SSH login password)")
	}
	full := fmt.Sprintf(`set -e
printf '%%s\n' %s | sudo -S -v
%s
`, shellSingleQuote(password), script)
	return runCmd(client, full)
}

func runCmd(client *ssh.Client, script string) error {
	session, err := client.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()

	var stderr bytes.Buffer
	session.Stderr = &stderr
	if err := session.Run(script); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg != "" {
			return fmt.Errorf("%w: %s", err, msg)
		}
		return err
	}
	return nil
}
