package secrets

import (
	"os"
	"os/exec"
	"strings"
)

var passEnvKeys = map[string]string{
	"LASTFM_API_KEY":     "lastfm/api-key",
	"LASTFM_API_SECRET":  "lastfm/api-secret",
	"LASTFM_SESSION_KEY": "lastfm/session-key",
	"DISCOGS_TOKEN":      "discogs/token",
}

func passPrefix() string {
	if v := strings.TrimSpace(os.Getenv("EVOPLAYER_PASS_PREFIX")); v != "" {
		return v
	}
	return "omarchy"
}

func passPath(rel string) string {
	return passPrefix() + "/" + rel
}

func Load() {
	if _, err := exec.LookPath("pass"); err != nil {
		return
	}
	for envKey, rel := range passEnvKeys {
		if os.Getenv(envKey) != "" {
			continue
		}
		out, err := exec.Command("pass", "show", passPath(rel)).Output()
		if err != nil {
			continue
		}
		value := strings.TrimSpace(string(out))
		if value == "" {
			continue
		}
		_ = os.Setenv(envKey, value)
	}
}

// LastfmConfigured reports whether Last.fm credentials are in the environment; call Load first.
func LastfmConfigured() bool {
	return os.Getenv("LASTFM_API_KEY") != "" &&
		os.Getenv("LASTFM_API_SECRET") != "" &&
		os.Getenv("LASTFM_SESSION_KEY") != ""
}
