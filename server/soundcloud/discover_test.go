package soundcloud

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPickMatchDurationAndTitle(t *testing.T) {
	tracks := []Track{
		scTrack(1, "Someone", "Other Song", 180000),
		scTrack(2, "Ada", "Night Drive (Original Mix)", 200000),
		scTrack(3, "Ada", "Night Drive", 400000),
	}
	tracks[2].Policy = "SNIP"
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
			Username  string `json:"username"`
			AvatarURL string `json:"avatar_url"`
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
			Username  string `json:"username"`
			AvatarURL string `json:"avatar_url"`
		}{Username: "Ada"},
	}}
	if _, err := pickMatch(tracks, "Ada", "Night Drive", 200); err == nil {
		t.Fatal("expected duration mismatch")
	}
}

func scTrack(id int64, user, title string, ms int64) Track {
	t := Track{ID: id, Title: title, Duration: ms}
	t.User.Username = user
	return t
}

func TestArtworkThumb(t *testing.T) {
	art := scTrack(1, "NOISIA", "Meditation", 1)
	art.ArtworkURL = "https://i1.sndcdn.com/artworks-abc-large.jpg"
	art.User.AvatarURL = "https://i1.sndcdn.com/avatars-xyz-large.jpg"
	if got := artworkThumb(art); got != "https://i1.sndcdn.com/artworks-abc-t300x300.jpg" {
		t.Fatalf("artwork = %s", got)
	}
	missing := scTrack(2, "NOISIA", "Meditation", 1)
	missing.User.AvatarURL = "https://i1.sndcdn.com/avatars-xyz-t500x500.jpg"
	if got := artworkThumb(missing); got != "https://i1.sndcdn.com/avatars-xyz-t300x300.jpg" {
		t.Fatalf("avatar = %s", got)
	}
}

func TestArtistTokens(t *testing.T) {
	got := artistTokens("Noisia Maldini And Vegas feat. MC X")
	want := []string{"noisia", "maldini", "vegas"}
	if len(got) != len(want) {
		t.Fatalf("tokens = %v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("tokens = %v", got)
		}
	}
}

func TestLooseMatchAcceptsLabelUpload(t *testing.T) {
	tracks := []Track{
		scTrack(1, "Bad Taste Recordings", "Noisia, Maldini And Vegas - Meditation", 426000),
		scTrack(2, "Random", "Vindication Of Hope", 3668000),
	}
	if _, err := pickMatch(tracks, "Noisia Maldini And Vegas", "Meditation", 425.6); err == nil {
		t.Fatal("strict match should reject a label upload")
	}
	got := pickLooseMatch(tracks, artistTokens("Noisia Maldini And Vegas"), "Meditation", 425.6)
	if got == nil || got.ID != 1 {
		t.Fatalf("got %+v, want label upload", got)
	}
}

func TestLooseMatchPrefersArtistUploadAndRejectsMashup(t *testing.T) {
	tracks := []Track{
		scTrack(1, "Bone", "Noisia & Electric Six - Meditation At The Gay Bar [MASHUP]", 109000),
		scTrack(2, "Bad Taste Recordings", "Noisia, Maldini And Vegas - Meditation", 426000),
		scTrack(3, "NOISIA", "Meditation", 426000),
		scTrack(4, "Nautic", "Noisia & Bad Company - Meditation", 432000),
	}
	got := pickLooseMatch(tracks, artistTokens("Noisia Maldini And Vegas"), "Meditation", 425.6)
	if got == nil || got.ID != 3 {
		t.Fatalf("got %+v, want artist upload", got)
	}
	if pickLooseMatch(tracks[:1], artistTokens("Noisia"), "Meditation", 425.6) != nil {
		t.Fatal("mashup with a different length should not match")
	}
}

func TestLooseMatchNeedsDuration(t *testing.T) {
	tracks := []Track{scTrack(1, "NOISIA", "Meditation", 426000)}
	if pickLooseMatch(tracks, artistTokens("Noisia"), "Meditation", 0) != nil {
		t.Fatal("loose match without a local duration should be refused")
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
