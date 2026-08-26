package art

import (
	"os"
	"strings"

	"github.com/sebday/evoplayer/internal/secrets"
)

func loadSecrets() {
	secrets.Load()
}

func discogsToken() string {
	return strings.TrimSpace(os.Getenv("DISCOGS_TOKEN"))
}
