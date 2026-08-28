package secrets

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/pbkdf2"
	"crypto/sha1"
	"database/sql"
	"os"
	"path/filepath"
	"testing"
)

func TestDecryptChromiumValueRoundTrip(t *testing.T) {
	password := []byte("test-keyring-secret")
	want := "soundcloud-oauth-token-value"
	enc, err := encryptChromiumValue(want, password)
	if err != nil {
		t.Fatal(err)
	}
	got, err := decryptChromiumValue(enc, password)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestReadSoundcloudCookieFromFixture(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "Cookies")
	password := []byte("peanuts")
	enc, err := encryptChromiumValue("cookie-token", password)
	if err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`CREATE TABLE cookies (
		host_key TEXT,
		name TEXT,
		value TEXT,
		encrypted_value BLOB
	)`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO cookies (host_key, name, value, encrypted_value)
		VALUES ('soundcloud.com', 'oauth_token', '', ?)`, enc); err != nil {
		t.Fatal(err)
	}
	db.Close()

	got, err := readSoundcloudCookie(dbPath, password)
	if err != nil {
		t.Fatal(err)
	}
	if got != "cookie-token" {
		t.Fatalf("got %q", got)
	}
}

func TestResolveSoundcloudPrefersBraveCookie(t *testing.T) {
	home := t.TempDir()
	password := []byte("peanuts")
	enc, err := encryptChromiumValue("from-brave", password)
	if err != nil {
		t.Fatal(err)
	}
	storeDir := filepath.Join(home, ".config", "BraveSoftware", "Brave-Browser", "Default")
	if err := os.MkdirAll(storeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeCookieDB(t, filepath.Join(storeDir, "Cookies"), enc)

	got := resolveSoundcloud(home, func(string) (string, error) {
		return "from-pass", nil
	}, func(string) []byte { return password })
	if got.Source != "brave" || got.Token != "from-brave" {
		t.Fatalf("got %+v", got)
	}
}

func TestResolveSoundcloudFallsBackToPass(t *testing.T) {
	home := t.TempDir()
	got := resolveSoundcloud(home, func(rel string) (string, error) {
		if rel != soundcloudPassRel {
			t.Fatalf("pass rel = %q", rel)
		}
		return "pass-token\n", nil
	}, func(string) []byte { return []byte("peanuts") })
	if got.Source != "pass" || got.Token != "pass-token" {
		t.Fatalf("got %+v", got)
	}
}

func TestPassShowUsesFakeBinary(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "pass")
	if err := os.WriteFile(script, []byte("#!/bin/sh\n[ \"$1\" = show ] && [ \"$2\" = testhome/soundcloud/oauth-token ] && printf 'line-one\\nline-two\\n' && exit 0\nexit 1\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("EVOPLAYER_PASS_PREFIX", "testhome")
	got, err := passShow(soundcloudPassRel)
	if err != nil {
		t.Fatal(err)
	}
	if got != "line-one" {
		t.Fatalf("got %q", got)
	}
}

func TestLiveBraveSoundcloudCookie(t *testing.T) {
	if os.Getenv("EVOPLAYER_TEST_LIVE_COOKIES") == "" {
		t.Skip("set EVOPLAYER_TEST_LIVE_COOKIES=1")
	}
	tok := SoundcloudOAuth()
	if tok.Token == "" || tok.Source == "" {
		t.Fatalf("source=%q token_len=%d", tok.Source, len(tok.Token))
	}
	t.Logf("source=%s token_len=%d", tok.Source, len(tok.Token))
}

func writeCookieDB(t *testing.T, path string, encrypted []byte) {
	t.Helper()
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(`CREATE TABLE cookies (
		host_key TEXT,
		name TEXT,
		value TEXT,
		encrypted_value BLOB
	)`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO cookies (host_key, name, value, encrypted_value)
		VALUES ('soundcloud.com', 'oauth_token', '', ?)`, encrypted); err != nil {
		t.Fatal(err)
	}
}

func encryptChromiumValue(plain string, password []byte) ([]byte, error) {
	key, err := pbkdf2.Key(sha1.New, string(password), []byte("saltysalt"), 1, 16)
	if err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	raw := []byte(plain)
	pad := aes.BlockSize - (len(raw) % aes.BlockSize)
	for i := 0; i < pad; i++ {
		raw = append(raw, byte(pad))
	}
	out := make([]byte, len(raw))
	cipher.NewCBCEncrypter(block, bytesOf(' ', aes.BlockSize)).CryptBlocks(out, raw)
	return append([]byte("v10"), out...), nil
}
