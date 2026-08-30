package secrets

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/pbkdf2"
	"crypto/sha1"
	"database/sql"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	_ "modernc.org/sqlite"
)

type cookieStore struct {
	Source  string
	Keyring string
	Path    string
}

func browserCookieStores(home string) []cookieStore {
	if home == "" {
		return nil
	}
	return []cookieStore{
		{
			Source:  "brave",
			Keyring: "Brave",
			Path:    filepath.Join(home, ".config", "BraveSoftware", "Brave-Browser", "Default", "Cookies"),
		},
		{
			Source:  "chromium",
			Keyring: "Chromium",
			Path:    filepath.Join(home, ".config", "chromium", "Default", "Cookies"),
		},
	}
}

func tokenFromCookies(store cookieStore, password []byte) (string, error) {
	copyPath, cleanup, err := copyCookiesDB(store.Path)
	if err != nil {
		return "", err
	}
	defer cleanup()
	return readSoundcloudCookie(copyPath, password)
}

func copyCookiesDB(src string) (string, func(), error) {
	dir, err := os.MkdirTemp("", "evoplayer-cookies-")
	if err != nil {
		return "", nil, err
	}
	cleanup := func() { _ = os.RemoveAll(dir) }
	dst := filepath.Join(dir, "Cookies")
	if err := copyFile(src, dst); err != nil {
		cleanup()
		return "", nil, err
	}
	for _, suf := range []string{"-wal", "-shm"} {
		if _, err := os.Stat(src + suf); err == nil {
			_ = copyFile(src+suf, dst+suf)
		}
	}
	return dst, cleanup, nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(out, in)
	closeErr := out.Close()
	if copyErr != nil {
		return copyErr
	}
	return closeErr
}

func readSoundcloudCookie(dbPath string, password []byte) (string, error) {
	db, err := sql.Open("sqlite", dbPath+"?mode=ro")
	if err != nil {
		return "", err
	}
	defer db.Close()

	rows, err := db.Query(`
		SELECT host_key, value, encrypted_value
		FROM cookies
		WHERE name = 'oauth_token'
		  AND (host_key = 'soundcloud.com' OR host_key = '.soundcloud.com')
		ORDER BY CASE WHEN host_key = 'soundcloud.com' THEN 0 ELSE 1 END
	`)
	if err != nil {
		return "", err
	}
	defer rows.Close()

	var lastErr error
	for rows.Next() {
		var host, value string
		var encrypted []byte
		if err := rows.Scan(&host, &value, &encrypted); err != nil {
			lastErr = err
			continue
		}
		if tok := strings.TrimSpace(value); tok != "" {
			return tok, nil
		}
		if len(encrypted) == 0 {
			continue
		}
		tok, err := decryptChromiumValue(encrypted, password)
		if err != nil {
			lastErr = err
			continue
		}
		if tok != "" {
			return tok, nil
		}
	}
	if lastErr != nil {
		return "", lastErr
	}
	return "", fmt.Errorf("no oauth_token cookie")
}

func decryptChromiumValue(encrypted, password []byte) (string, error) {
	if len(encrypted) < 3 {
		return "", fmt.Errorf("cookie ciphertext too short")
	}
	prefix := string(encrypted[:3])
	payload := encrypted
	switch prefix {
	case "v10", "v11":
		payload = encrypted[3:]
	case "v20":
		return "", fmt.Errorf("unsupported chromium cookie prefix v20")
	}
	if len(payload)%aes.BlockSize != 0 {
		return "", fmt.Errorf("cookie ciphertext not block-aligned")
	}
	key, err := pbkdf2.Key(sha1.New, string(password), []byte("saltysalt"), 1, 16)
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	plain := make([]byte, len(payload))
	cipher.NewCBCDecrypter(block, bytesOf(' ', aes.BlockSize)).CryptBlocks(plain, payload)
	plain, err = pkcs7Unpad(plain)
	if err != nil {
		return "", err
	}
	if tok := cookiePlaintext(plain); tok != "" {
		return tok, nil
	}
	return "", fmt.Errorf("empty cookie plaintext")
}

func cookiePlaintext(plain []byte) string {
	candidates := [][]byte{plain}
	if len(plain) > 32 {
		candidates = [][]byte{plain[32:], plain}
	}
	for _, raw := range candidates {
		if tok := normalizeSoundcloudOAuth(string(raw)); tok != "" {
			return tok
		}
	}
	return ""
}

func pkcs7Unpad(b []byte) ([]byte, error) {
	if len(b) == 0 {
		return nil, fmt.Errorf("empty padded data")
	}
	pad := int(b[len(b)-1])
	if pad == 0 || pad > len(b) || pad > aes.BlockSize {
		return nil, fmt.Errorf("invalid padding")
	}
	for _, c := range b[len(b)-pad:] {
		if int(c) != pad {
			return nil, fmt.Errorf("invalid padding")
		}
	}
	return b[:len(b)-pad], nil
}

func bytesOf(c byte, n int) []byte {
	b := make([]byte, n)
	for i := range b {
		b[i] = c
	}
	return b
}
