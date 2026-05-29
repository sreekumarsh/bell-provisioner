package config

import "testing"

func TestGitHubRepo(t *testing.T) {
	owner, repo, err := GitHubRepo("git@github.com:sreekumarsh/pi-streamer.git")
	if err != nil || owner != "sreekumarsh" || repo != "pi-streamer" {
		t.Fatalf("ssh url: %q %q %v", owner, repo, err)
	}
	owner, repo, err = GitHubRepo("https://github.com/sreekumarsh/pi-streamer")
	if err != nil || owner != "sreekumarsh" || repo != "pi-streamer" {
		t.Fatalf("https url: %q %q %v", owner, repo, err)
	}
}
