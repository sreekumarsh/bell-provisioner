package config

import "strings"

// ValidGitHubToken reports whether t looks like a real PAT (not a UI placeholder).
func ValidGitHubToken(t string) bool {
	t = strings.TrimSpace(t)
	if len(t) < 40 {
		return false
	}
	return strings.HasPrefix(t, "ghp_") ||
		strings.HasPrefix(t, "github_pat_") ||
		strings.HasPrefix(t, "gho_")
}
