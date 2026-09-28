package soundcloud

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPickMatchDurationAndTitle(t *testing.T) {
	tracks := []Track{
		{ID: 1, Title: "Other Song", Duration: 180000, User: struct {
			Username string `json:"username"`
		}{Username: "Someone"}},
		{ID: 2, Title: "Night Drive (Original Mix)", Duration: 200000, User: struct {
			Username string `json:"username"`
		}{Username: "Ada"}},
		{ID: 3, Title: "Night Drive", Duration: 400000, Policy: "SNIP", User: struct {
			Username string `json:"username"`
		}{Username: "Ada"}},
	}
	got, err := pickMatch(tracks, "Ada", "Night Drive", 201)
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != 2 {
		t.Fatalf("id = %d, want 2", got.ID)
	}
}

func TestPickMatchRejectsWrongArtist(t *testing.T) {
	tracks := []Track{{
		ID: 2, Title: "B2. The Prodigy - Charly (Original Mix)", Duration: 239000,
		User: struct {
			Username string `json:"username"`
		}{Username: "Alisa Lee"},
	}}
	if _, err := pickMatch(tracks, "The Prodigy", "Charly (Original Mix)", 239); err == nil {
		t.Fatal("expected artist mismatch")
	}
}

func TestPickMatchRejectsDuration(t *testing.T) {
	tracks := []Track{{
		ID: 2, Title: "Night Drive", Duration: 400000,
		User: struct {
			Username string `json:"username"`
		}{Username: "Ada"},
	}}
	if _, err := pickMatch(tracks, "Ada", "Night Drive", 200); err == nil {
		t.Fatal("expected duration mismatch")
	}
}

func TestListableDropsSnippetAndDRM(t *testing.T) {
	snip := Track{ID: 1, Policy: "SNIP"}
	if listable(&snip) {
		t.Fatal("snippet track should be dropped")
	}
	drm := Track{ID: 2, Media: struct {
		Transcodings []Transcoding `json:"transcodings"`
	}{Transcodings: []Transcoding{{Format: struct {
		Protocol string `json:"protocol"`
		MimeType string `json:"mime_type"`
	}{Protocol: "cbc-encrypted-hls"}}}}}
	if listable(&drm) {
		t.Fatal("encrypted-only track should be dropped")
	}
	open := Track{ID: 3}
	if !listable(&open) {
		t.Fatal("track without media should stay")
	}
}

func TestDecodeTrackListShapes(t *testing.T) {
	direct := []byte(`{"collection":[{"id":9,"title":"A"},{"track":{"id":8,"title":"B"}}]}`)
	tracks, err := decodeTrackList(direct)
	if err != nil {
		t.Fatal(err)
	}
	if len(tracks) != 2 || tracks[0].ID != 9 || tracks[1].ID != 8 {
		t.Fatalf("tracks = %+v", tracks)
	}
	bare := []byte(`[{"id":4,"title":"C"}]`)
	tracks, err = decodeTrackList(bare)
	if err != nil {
		t.Fatal(err)
	}
	if len(tracks) != 1 || tracks[0].ID != 4 {
		t.Fatalf("tracks = %+v", tracks)
	}
}

func TestDismissRoundTrip(t *testing.T) {
	dir := t.TempDir()
	if err := DismissID(dir, 42); err != nil {
		t.Fatal(err)
	}
	if err := DismissID(dir, 42); err != nil {
		t.Fatal(err)
	}
	set, err := dismissedSet(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !set[42] || len(set) != 1 {
		t.Fatalf("set = %v", set)
	}
	b, err := os.ReadFile(filepath.Join(dir, "discover-dismissed.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(b) == 0 || b[len(b)-1] != '\n' {
		t.Fatalf("file = %q", b)
	}
}

func TestIsPreviewPath(t *testing.T) {
	cache := filepath.Join(t.TempDir(), "cache")
	preview := PreviewPath(cache, 7)
	if !IsPreviewPath(cache, preview) {
		t.Fatal("preview path not recognized")
	}
	if IsPreviewPath(cache, filepath.Join(cache, "tracks", "7.mp3")) {
		t.Fatal("other cache path recognized as preview")
	}
}
