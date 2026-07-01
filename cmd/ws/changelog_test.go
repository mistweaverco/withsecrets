package ws

import "testing"

func TestChangelogCommandRegistered(t *testing.T) {
	found := false
	for _, c := range rootCmd.Commands() {
		if c.Name() == "changelog" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected changelog command to be registered")
	}
}
