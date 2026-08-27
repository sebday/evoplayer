package secrets

import (
	"os"
	"os/exec"
	"strings"
)

const soundcloudPassRel = "soundcloud/oauth-token"

type Token struct {
	Token  string
	Source string
}

func SoundcloudPassPath() string {
	return passPath(soundcloudPassRel)
}

func SoundcloudOAuth() Token {
	home, _ := os.UserHomeDir()
	return resolveSoundcloud(home, passShow, keyringPassword)
}

func resolveSoundcloud(home string, show passFunc, keyring func(string) []byte) Token {
	for _, store := range browserCookieStores(home) {
		if _, err := os.Stat(store.Path); err != nil {
			continue
		}
		passwords := [][]byte{keyring(store.Keyring), []byte("peanuts")}
		seen := map[string]bool{}
		for _, password := range passwords {
			if len(password) == 0 {
				continue
			}
			key := string(password)
			if seen[key] {
				continue
			}
			seen[key] = true
			tok, err := tokenFromCookies(store, password)
			if err != nil || strings.TrimSpace(tok) == "" {
				continue
			}
			return Token{Token: strings.TrimSpace(tok), Source: store.Source}
		}
	}
	if show != nil {
		if tok, err := show(soundcloudPassRel); err == nil {
			tok = strings.TrimSpace(tok)
			if tok != "" {
				return Token{Token: tok, Source: "pass"}
			}
		}
	}
	return Token{}
}

type passFunc func(rel string) (string, error)

func passShow(rel string) (string, error) {
	if _, err := exec.LookPath("pass"); err != nil {
		return "", err
	}
	out, err := exec.Command("pass", "show", passPath(rel)).Output()
	if err != nil {
		return "", err
	}
	text := strings.ReplaceAll(string(out), "\r\n", "\n")
	if i := strings.IndexByte(text, '\n'); i >= 0 {
		text = text[:i]
	}
	return strings.TrimSpace(text), nil
}
