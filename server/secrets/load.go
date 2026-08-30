package secrets

import (
	"os"
	"os/exec"
	"strings"
)

var lastfmKeys = map[string]string{
	"LASTFM_API_KEY":     "lastfm/api-key",
	"LASTFM_API_SECRET":  "lastfm/api-secret",
	"LASTFM_SESSION_KEY": "lastfm/session-key",
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
	for envKey, rel := range lastfmKeys {
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

func LastfmConfigured() bool {
	Load()
	return os.Getenv("LASTFM_API_KEY") != "" &&
		os.Getenv("LASTFM_API_SECRET") != "" &&
		os.Getenv("LASTFM_SESSION_KEY") != ""
}
