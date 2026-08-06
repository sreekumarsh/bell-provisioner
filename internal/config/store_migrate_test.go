package config

import "testing"

func TestMigrateAgentRepoURL(t *testing.T) {
	got := migrateAgentRepoURL("git@github.com:sreekumarsh/vyooham-sense.git", "vyooham-sense", DefaultSenseAgentRepoURL)
	if got != DefaultSenseAgentRepoURL {
		t.Fatalf("stale sense URL: got %q want %q", got, DefaultSenseAgentRepoURL)
	}
	got = migrateAgentRepoURL("", "vyooham-sense", DefaultSenseAgentRepoURL)
	if got != DefaultSenseAgentRepoURL {
		t.Fatalf("empty sense URL: got %q", got)
	}
	custom := "git@github.com:vyooham/vyooham-sense.git"
	if migrateAgentRepoURL(custom, "vyooham-sense", DefaultSenseAgentRepoURL) != custom {
		t.Fatalf("vyooham URL should stay put")
	}
	fork := "git@github.com:other/vyooham-sense.git"
	if migrateAgentRepoURL(fork, "vyooham-sense", DefaultSenseAgentRepoURL) != fork {
		t.Fatalf("unrelated fork should stay put")
	}
}
