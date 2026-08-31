package download

import "testing"

func TestNormalizeCollectionURL(t *testing.T) {
	got := NormalizeCollectionURL("https://soundcloud.com/seb-day")
	want := "https://soundcloud.com/seb-day/tracks"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
	if got := NormalizeCollectionURL("https://soundcloud.com/seb-day/sets/chill"); got != "https://soundcloud.com/seb-day/sets/chill" {
		t.Fatalf("playlist unchanged: %q", got)
	}
}
