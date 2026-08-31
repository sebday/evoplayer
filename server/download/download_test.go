package download

import "testing"

func TestClassifyURL(t *testing.T) {
	cases := []struct {
		url  string
		kind string
	}{
		{"https://www.youtube.com/watch?v=m7EwpN1jWTg", KindYouTube},
		{"https://youtu.be/m7EwpN1jWTg", KindYouTube},
		{"https://soundcloud.com/you/likes", KindSCLikes},
		{"https://soundcloud.com/seb-day/likes", KindSCPlaylist},
		{"https://soundcloud.com/seb-day/sets/chill", KindSCPlaylist},
		{"https://soundcloud.com/storm-queen-official", KindSCArtist},
		{"https://soundcloud.com/storm-queen-official/tracks", KindSCArtist},
		{"https://soundcloud.com/t-shirtssweats/anything", KindSCTrack},
		{"https://example.com/foo", ""},
	}
	for _, c := range cases {
		got := ClassifyURL(c.url)
		if got != c.kind {
			t.Fatalf("ClassifyURL(%q) = %q, want %q", c.url, got, c.kind)
		}
	}
}
