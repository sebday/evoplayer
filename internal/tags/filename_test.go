package tags

import "testing"

func TestParseFilenameArtistTitle(t *testing.T) {
	tests := []struct {
		stem         string
		wantArtist   string
		wantTitle    string
	}{
		{"07-chase_and_status-eastern_jam", "chase and status", "eastern jam"},
		{"01-breakage-clarendon-xtc", "breakage", "clarendon-xtc"},
		{"Artist - Title", "Artist", "Title"},
		{"plain_name", "", "plain_name"},
	}
	for _, tc := range tests {
		artist, title := ParseFilenameArtistTitle(tc.stem)
		if artist != tc.wantArtist || title != tc.wantTitle {
			t.Fatalf("ParseFilenameArtistTitle(%q) = (%q, %q), want (%q, %q)",
				tc.stem, artist, title, tc.wantArtist, tc.wantTitle)
		}
	}
}
