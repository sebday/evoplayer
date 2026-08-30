package art

import "testing"

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

func TestDiscogsJoinQuery(t *testing.T) {
	got := discogsJoinQuery("  CAT1 ", "", " Artist ")
	if got != "CAT1 - Artist" {
		t.Fatalf("join = %q", got)
	}
}

func TestPreviewURLUses600FitIn(t *testing.T) {
	got := PreviewURL(Result{
		Thumb: "https://i.discogs.com/x/fit-in/150x150/R-1.jpg",
		URL:   "https://i.discogs.com/x/fit-in/150x150/R-1.jpg",
	})
	if got != "https://i.discogs.com/x/fit-in/600x600/R-1.jpg" {
		t.Fatalf("preview url = %q", got)
	}
}

func TestPreviewURLPrefersThumbForFitInRsFitCombo(t *testing.T) {
	thumb := "https://i.discogs.com/t/rs:fit/g:sm/q:40/h:150/w:150/foo.jpeg"
	full := "https://i.discogs.com/t/fit-in/600x600/rs:fit/g:sm/q:90/h:480/w:496/foo.jpeg"
	got := PreviewURL(Result{URL: full, Thumb: thumb})
	if got != thumb {
		t.Fatalf("preview url = %q", got)
	}
}

func TestPreviewURLKeepsSignedDiscogsURL(t *testing.T) {
	signed := "https://i.discogs.com/token/rs:fit/g:sm/q:90/h:600/w:600/czM6Ly9kaXNjb2dz/R-1.jpeg"
	got := PreviewURL(Result{
		Thumb: "https://i.discogs.com/x/fit-in/150x150/R-1.jpg",
		URL:   signed,
	})
	if got != signed {
		t.Fatalf("preview url = %q", got)
	}
}

func TestPreviewURLPrefersFullCoverUnchanged(t *testing.T) {
	full := "https://i.discogs.com/x/discogs-images/R-1.jpg"
	got := PreviewURL(Result{
		Thumb: "https://i.discogs.com/x/fit-in/150x150/R-1.jpg",
		URL:   full,
	})
	if got != full {
		t.Fatalf("preview url = %q", got)
	}
}
