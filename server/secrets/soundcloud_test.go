package secrets

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/pbkdf2"
	"crypto/sha1"
	"database/sql"
	"path/filepath"
	"testing"
)

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

func TestNormalizeSoundcloudOAuthExtractsEmbeddedToken(t *testing.T) {
	raw := "\xf8\xb5\x8fj{\xc9\x97\x7f{\x06ix\x98%0ws\v-;\x17\x1a\x7f\xef\xcf\xfb2-329470-4587087-OxdIQgBWgnHRifB8"
	got := normalizeSoundcloudOAuth(raw)
	want := "2-329470-4587087-OxdIQgBWgnHRifB8"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
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
