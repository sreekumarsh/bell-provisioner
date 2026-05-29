package config

import (
	"fmt"
	"strings"
)

// GitHubRepo parses owner/repo from an agent_repo_url (SSH or HTTPS).
func GitHubRepo(repoURL string) (owner, repo string, err error) {
	repoURL = strings.TrimSpace(repoURL)
	repoURL = strings.TrimSuffix(repoURL, ".git")

	const sshPrefix = "git@github.com:"
	if strings.HasPrefix(repoURL, sshPrefix) {
		parts := strings.SplitN(strings.TrimPrefix(repoURL, sshPrefix), "/", 2)
		if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
			return "", "", fmt.Errorf("invalid git SSH URL: %s", repoURL)
		}
		return parts[0], parts[1], nil
	}

	const httpsPrefix = "https://github.com/"
	if strings.HasPrefix(repoURL, httpsPrefix) {
		parts := strings.SplitN(strings.TrimPrefix(repoURL, httpsPrefix), "/", 2)
		if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
			return "", "", fmt.Errorf("invalid git HTTPS URL: %s", repoURL)
		}
		return parts[0], parts[1], nil
	}
	return "", "", fmt.Errorf("unsupported agent_repo_url: %s", repoURL)
}
