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
