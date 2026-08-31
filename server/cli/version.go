package cli

import (
	"os"
	"time"
)

func BuildVersion(exe string) string {
	if exe == "" {
		exe, _ = os.Executable()
	}
	if st, err := os.Stat(exe); err == nil {
		return st.ModTime().UTC().Format(time.RFC3339)
	}
	return "unknown"
}
