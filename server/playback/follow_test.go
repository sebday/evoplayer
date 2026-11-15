package playback

import "testing"

func TestFollowAdvance(t *testing.T) {
	if mode, i := followAdvance(-1, 3, false, true); mode != "follow" || i != 0 {
		t.Fatalf("end of queue = %s %d", mode, i)
	}
	if mode, _ := followAdvance(-1, 3, true, true); mode != "queue" {
		t.Fatalf("repeat stays on the queue, got %s", mode)
	}
	if mode, _ := followAdvance(-1, 0, false, true); mode != "queue" {
		t.Fatalf("empty follow wraps the queue, got %s", mode)
	}
	if mode, i := followAdvance(0, 3, false, false); mode != "follow" || i != 1 {
		t.Fatalf("next related = %s %d", mode, i)
	}
	if mode, i := followAdvance(2, 3, false, false); mode != "stop" || i != -1 {
		t.Fatalf("last related = %s %d", mode, i)
	}
	if mode, i := followAdvance(1, 3, true, false); mode != "replay" || i != 1 {
		t.Fatalf("repeat related = %s %d", mode, i)
	}
}

func TestFollowSkip(t *testing.T) {
	if mode, i := followSkip(-1, 2, true, 1); mode != "follow" || i != 0 {
		t.Fatalf("next at end = %s %d", mode, i)
	}
	if mode, i := followSkip(0, 2, false, -1); mode != "back" || i != -1 {
		t.Fatalf("prev from first related = %s %d", mode, i)
	}
	if mode, i := followSkip(1, 2, false, 1); mode != "stop" || i != -1 {
		t.Fatalf("next past related = %s %d", mode, i)
	}
}

func TestAdvancesPastEnd(t *testing.T) {
	a := NewActor(nil)
	a.mu.Lock()
	a.queue = []string{"a", "b", "c"}
	a.index = 2
	if !a.advancesPastEndLocked(1) {
		t.Fatal("last track should end the queue")
	}
	a.index = 1
	if a.advancesPastEndLocked(1) {
		t.Fatal("middle track should advance inside the queue")
	}
	a.shuffle = true
	a.shuffleOrd = []int{2, 0, 1}
	a.index = 1
	if !a.advancesPastEndLocked(1) {
		t.Fatal("last shuffle slot should end the queue")
	}
	a.index = 0
	if a.advancesPastEndLocked(1) {
		t.Fatal("middle shuffle slot should stay in the queue")
	}
	a.mu.Unlock()
}
