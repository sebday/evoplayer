package daemon

import (
	"testing"

	"github.com/sebday/evoplayer/server/soundcloud"
)

func TestQueueSeedsSpreadsTheList(t *testing.T) {
	paths := []string{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j"}
	got := queueSeeds(paths, 4)
	want := []string{"a", "c", "f", "h"}
	if len(got) != len(want) {
		t.Fatalf("seeds = %v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("seeds = %v", got)
		}
	}
	short := queueSeeds([]string{"a", " /discover/1.mp3", "a", "b"}, 8)
	if len(short) != 2 || short[0] != "a" || short[1] != "b" {
		t.Fatalf("short = %v", short)
	}
}

func TestDropOwnedSimilar(t *testing.T) {
	tracks := []soundcloud.SimilarTrack{
		{ID: 1, Artist: "Alias", Title: "Gladiator II"},
		{ID: 2, Artist: "Alias", Title: "Fiction II"},
		{ID: 3, Artist: "Someone", Title: "New One"},
	}
	got := dropOwnedSimilar(tracks, func(id int64) bool { return id == 2 }, map[string]struct{}{
		ownedTitleKey("alias", "Gladiator  II"): {},
	})
	if len(got) != 1 || got[0].ID != 3 {
		t.Fatalf("kept = %+v", got)
	}
}

func TestMergeSimilarRoundRobin(t *testing.T) {
	lists := [][]soundcloud.SimilarTrack{
		{{ID: 1, Title: "a1"}, {ID: 2, Title: "a2"}},
		{{ID: 1, Title: "dup"}, {ID: 3, Title: "b2"}},
		{{ID: 4, Title: "c1"}},
	}
	got := mergeSimilar(lists, 8)
	if len(got) != 4 || got[0].ID != 1 || got[1].ID != 4 || got[2].ID != 2 || got[3].ID != 3 {
		t.Fatalf("merged = %+v", got)
	}
}
