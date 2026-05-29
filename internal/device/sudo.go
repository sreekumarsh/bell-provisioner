package device

import (
	"bytes"
	"fmt"
	"strings"

	"golang.org/x/crypto/ssh"
)

func shellSingleQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'"'"'`) + "'"
}

// runSudo runs a shell script on the Pi, authenticating sudo once via stdin (-S).
func runSudo(client *ssh.Client, password, script string) error {
	password = strings.TrimSpace(password)
	if password == "" {
		return fmt.Errorf("sudo password is required (same as SSH login password)")
	}
	script = strings.TrimSpace(script)
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
