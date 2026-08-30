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

func TestPreviewURLPrefersFullCoverAt600(t *testing.T) {
	got := PreviewURL(Result{
		Thumb: "https://i.discogs.com/x/fit-in/150x150/R-1.jpg",
		URL:   "https://i.discogs.com/x/discogs-images/R-1.jpg",
	})
	if got != "https://i.discogs.com/x/fit-in/600x600/discogs-images/R-1.jpg" {
		t.Fatalf("preview url = %q", got)
	}
}
