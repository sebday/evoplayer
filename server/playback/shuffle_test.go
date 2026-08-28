package playback

import "testing"

func TestShuffledOrderPermutation(t *testing.T) {
	ord := shuffledOrder(8, 3)
	if len(ord) != 8 {
		t.Fatalf("expected 8, got %d", len(ord))
	}
	seen := map[int]bool{}
	for _, i := range ord {
		if i < 0 || i >= 8 {
			t.Fatalf("bad index %d", i)
		}
		seen[i] = true
	}
	if len(seen) != 8 {
		t.Fatalf("not a permutation: %v", ord)
	}
}
