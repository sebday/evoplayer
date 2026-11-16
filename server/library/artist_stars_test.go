package library

import (
	"os"
	"path/filepath"
	"testing"
)

func TestToggleArtistStar(t *testing.T) {
	env := Env{StateDir: t.TempDir()}
	on, err := ToggleArtistStar(env, "Deep Dark & Dangerous")
	if err != nil {
		t.Fatal(err)
	}
	if !on || !artistStarred(env, "deep dark & dangerous") {
		t.Fatalf("expected favorite, on=%v", on)
	}
	on, err = ToggleArtistStar(env, "Deep Dark & Dangerous")
	if err != nil {
		t.Fatal(err)
	}
	if on || artistStarred(env, "Deep Dark & Dangerous") {
		t.Fatal("expected unfavorite")
	}
	if _, err := os.Stat(filepath.Join(env.StateDir, "artist-stars.json")); err != nil {
		t.Fatal(err)
	}
}

func TestToggleAlbumStar(t *testing.T) {
	env := Env{StateDir: t.TempDir()}
	on, err := ToggleAlbumStar(env, "Temples of Boom")
	if err != nil {
		t.Fatal(err)
	}
	if !on || !albumStarred(env, "temples of boom") {
		t.Fatalf("expected favorite, on=%v", on)
	}
	on, err = ToggleAlbumStar(env, "Temples of Boom")
	if err != nil {
		t.Fatal(err)
	}
	if on || albumStarred(env, "Temples of Boom") {
		t.Fatal("expected unfavorite")
	}
	if _, err := os.Stat(filepath.Join(env.StateDir, "album-stars.json")); err != nil {
		t.Fatal(err)
	}
}
