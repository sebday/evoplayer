package art

import (
	"testing"
)

func TestLooksLikeCatno(t *testing.T) {
	if !looksLikeCatno("ABC123") {
		t.Fatal("expected catalog number")
	}
	if looksLikeCatno("ab") {
		t.Fatal("too short")
	}
	if looksLikeCatno("has space") {
		t.Fatal("spaces should fail")
	}
}

func TestNormCatno(t *testing.T) {
	if got := normCatno("AB-12/x"); got != "ab12x" {
		t.Fatalf("normCatno = %q", got)
	}
}

func TestDiscogsJoinQuery(t *testing.T) {
	got := discogsJoinQuery("  CAT1 ", "", " Artist ")
	if got != "CAT1 - Artist" {
		t.Fatalf("join = %q", got)
	}
}

func TestDedupe(t *testing.T) {
	rows := []Result{
		{URL: "http://a", Label: "1"},
		{URL: "http://a", Label: "dup"},
		{URL: "http://b", Label: "2"},
	}
	out := dedupe(rows)
	if len(out) != 2 {
		t.Fatalf("dedupe len = %d", len(out))
	}
}

func TestDiscogsArtistIDFromQuery(t *testing.T) {
	if got := discogsArtistIDFromQuery("https://www.discogs.com/artist/42-name"); got != "42" {
		t.Fatalf("id = %q", got)
	}
}

func TestDiscogsReleaseIDFromQuery(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"https://www.discogs.com/release/1144918-SNO-Cant-Touch-Dis", "1144918"},
		{"https://www.discogs.com/release/1144918-SNO-Cant-Touch-Dis/image/SW1hZ2U6MzMxNjcwODE=", "1144918"},
		{"https://api.discogs.com/releases/1144918", "1144918"},
	}
	for _, tc := range cases {
		if got := discogsReleaseIDFromQuery(tc.in); got != tc.want {
			t.Fatalf("discogsReleaseIDFromQuery(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
