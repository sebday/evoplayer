package secrets

import (
	"os"
	"os/exec"
	"testing"
)

func TestLoadLastfmFromPass(t *testing.T) {
	if _, err := exec.LookPath("pass"); err != nil {
		t.Skip("pass not installed")
	}
	if os.Getenv("HOME") == "" {
		t.Skip("HOME not set")
	}
	for envKey := range lastfmKeys {
		os.Setenv(envKey, "")
	}
	Load()
	if !LastfmConfigured() {
		t.Skip("pass omarchy/lastfm/* entries not configured")
	}
}
