package library

import (
	"image"
	"image/jpeg"
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestValidateImageBytesRejectsHTML(t *testing.T) {
	if err := validateImageBytes([]byte("<html>not art</html>")); err == nil {
		t.Fatal("html should not validate as image")
	}
}

func TestValidateImageBytesAcceptsJPEG(t *testing.T) {
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, image.NewRGBA(image.Rect(0, 0, 2, 2)), nil); err != nil {
		t.Fatal(err)
	}
	if err := validateImageBytes(buf.Bytes()); err != nil {
		t.Fatalf("jpeg should validate: %v", err)
	}
}

func TestApplyImageURLRejectsBadResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "nope", http.StatusForbidden)
	}))
	defer srv.Close()

	root := t.TempDir()
	env := Env{
		MusicRoot: root,
		ArtDir:    filepath.Join(root, "art"),
	}
	track := filepath.Join(root, "coki.mp3")
	if err := os.WriteFile(track, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := ApplyImageURL(env, track, srv.URL, "track"); err == nil {
		t.Fatal("expected apply to fail on bad http response")
	}
}
