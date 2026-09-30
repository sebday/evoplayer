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
	png := scTrack(3, "Playaz", "Return", 1)
	png.ArtworkURL = "https://i1.sndcdn.com/artworks-abc-t500x500.png"
	if got := artworkThumb(png); got != "https://i1.sndcdn.com/artworks-abc-t300x300.jpg" {
		t.Fatalf("png artwork = %s", got)
	}
	def := scTrack(4, "nobody", "Track", 1)
	def.User.AvatarURL = "https://a1.sndcdn.com/images/default_avatar_large.png"
	if got := artworkThumb(def); got != "https://a1.sndcdn.com/images/default_avatar_large.png" {
		t.Fatalf("default avatar = %s", got)
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

func TestFallbackMatchPrefersCleanTitleAndCloserLength(t *testing.T) {
	tracks := []Track{
		scTrack(1, "Jeremy Wicks", "J - MAJIK - SOLARIZE - RUN - DMT - AND - BIRD - PETERSON - REMIX", 324500),
		scTrack(2, "KnownDnB", "J Majik - Solarize", 517300),
		scTrack(3, "szomfa", "J Majik - Solarize", 470200),
		scTrack(4, "Maduk", "Solarize (feat. Logistics)", 273300),
		scTrack(5, "The Renegades/Dynamix", "J Majik - Solarize (The Renegades' Mix)", 181500),
	}
	got := pickFallbackMatch(tracks, artistTokens("J. Majik"), "Solarize", 384.1)
	if got == nil || got.ID != 3 {
		t.Fatalf("got %+v, want closest titled upload", got)
	}
}

func TestFallbackMatchRejectsUncreditedTitle(t *testing.T) {
	tracks := []Track{scTrack(4, "Maduk", "Solarize (feat. Logistics)", 273300)}
	if pickFallbackMatch(tracks, artistTokens("J. Majik"), "Solarize", 384.1) != nil {
		t.Fatal("title-only upload should not match another artist")
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

func TestPickMatchAcceptsGluedDJName(t *testing.T) {
	tracks := []Track{scTrack(9, "DJHatcha", "Roadtrip", 322332)}
	got, err := pickMatch(tracks, "DJ Hatcha", "Roadtrip", 322.3)
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != 9 {
		t.Fatalf("id = %d", got.ID)
	}
}

func TestLooseAndFallbackAcceptGluedDJName(t *testing.T) {
	tracks := []Track{scTrack(9, "DJHatcha", "Roadtrip", 322332)}
	tokens := artistTokens("DJ Hatcha")
	got := pickLooseMatch(tracks, tokens, "Roadtrip", 322.3)
	if got == nil || got.ID != 9 {
		t.Fatalf("loose = %+v", got)
	}
	off := []Track{scTrack(9, "DJHatcha", "Roadtrip", 500000)}
	got = pickFallbackMatch(off, tokens, "Roadtrip", 322.3)
	if got == nil || got.ID != 9 {
		t.Fatalf("fallback = %+v", got)
	}
	if tokenInSlug("hatchamania", "hatcha") {
		t.Fatal("hatcha should not match inside hatchamania")
	}
}

func TestFallbackRejectsLabelWordInTitle(t *testing.T) {
	got := artistTokens("Horsepower Productions")
	if len(got) != 1 || got[0] != "horsepower" {
		t.Fatalf("tokens = %v", got)
	}
	tracks := []Track{scTrack(1, "MAUI PETE", "VSP - 181 -VOODOO SPELL PRODUCTIONS", 3438707)}
	if pickFallbackMatch(tracks, got, "Voodoo Spell", 320) != nil {
		t.Fatal("a label word in the title should not credit the artist")
	}
}

func TestPickArtistUserGluedName(t *testing.T) {
	users := []scUser{
		{ID: 1, Username: "DJ MATCHÄ", FollowersCount: 1000},
		{ID: 2, Username: "Dj Catcha", FollowersCount: 5000},
		{ID: 3, Username: "DJHatcha", FollowersCount: 100},
		{ID: 4, Username: "Hatchamania", FollowersCount: 9000},
	}
	got := pickArtistUser(users, "DJ Hatcha")
	if got == nil || got.ID != 3 {
		t.Fatalf("got %+v", got)
	}
	if pickArtistUser(users[3:], "Hatcha") != nil {
		t.Fatal("substring username should not win")
	}
}

func TestCatalogTrackSkipsMixesAndClips(t *testing.T) {
	short := scTrack(1, "DJHatcha", "Roadtrip", 322332)
	if !catalogTrack(&short) {
		t.Fatal("song-length upload should stay")
	}
	clip := scTrack(2, "DJHatcha", "Clip", 82000)
	if catalogTrack(&clip) {
		t.Fatal("clip should be dropped")
	}
	mix := scTrack(3, "DJHatcha", "Oldskool Mix", 3600*1000)
	if catalogTrack(&mix) {
		t.Fatal("hour-long mix should be dropped")
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
