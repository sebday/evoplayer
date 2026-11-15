package playback

import "testing"

func TestFollowAdvance(t *testing.T) {
	if mode, i := followAdvance(-1, 3, false, true, false); mode != "follow" || i != 0 {
		t.Fatalf("end of queue = %s %d", mode, i)
	}
	if mode, _ := followAdvance(-1, 3, true, true, false); mode != "queue" {
		t.Fatalf("repeat stays on the queue, got %s", mode)
	}
	if mode, _ := followAdvance(-1, 0, false, true, false); mode != "queue" {
		t.Fatalf("empty follow wraps the queue, got %s", mode)
	}
	if mode, i := followAdvance(-1, 0, false, true, true); mode != "stop" || i != -1 {
		t.Fatalf("after the continuation = %s %d", mode, i)
	}
	if mode, i := followAdvance(0, 3, false, false, true); mode != "follow" || i != 1 {
		t.Fatalf("next related = %s %d", mode, i)
	}
	if mode, i := followAdvance(2, 3, false, false, true); mode != "stop" || i != -1 {
		t.Fatalf("last related = %s %d", mode, i)
	}
	if mode, i := followAdvance(1, 3, true, false, true); mode != "replay" || i != 1 {
		t.Fatalf("repeat related = %s %d", mode, i)
	}
}

func TestFollowSkip(t *testing.T) {
	if mode, i := followSkip(-1, 2, true, false, 1); mode != "follow" || i != 0 {
		t.Fatalf("next at end = %s %d", mode, i)
	}
	if mode, i := followSkip(-1, 0, true, true, 1); mode != "stop" || i != -1 {
		t.Fatalf("next after continuation = %s %d", mode, i)
	}
	if mode, i := followSkip(0, 2, false, true, -1); mode != "back" || i != -1 {
		t.Fatalf("prev from first related = %s %d", mode, i)
	}
	if mode, i := followSkip(1, 2, false, true, 1); mode != "stop" || i != -1 {
		t.Fatalf("next past related = %s %d", mode, i)
	}
}

func TestTakeFollowAppendsTheQueue(t *testing.T) {
	queue, follow := takeFollow([]string{"a", "b"}, []string{"r1", "r2"}, 0)
	if len(queue) != 3 || queue[2] != "r1" {
		t.Fatalf("queue = %#v", queue)
	}
	if len(follow) != 1 || follow[0] != "r2" {
		t.Fatalf("follow = %#v", follow)
	}
	again, rest := takeFollow(queue, follow, 0)
	if len(again) != 4 || again[3] != "r2" || len(rest) != 0 {
		t.Fatalf("second take queue=%#v follow=%#v", again, rest)
	}
	same, gone := takeFollow(again, []string{"r1"}, 0)
	if len(same) != len(again) || len(gone) != 0 {
		t.Fatalf("duplicate take queue=%#v follow=%#v", same, gone)
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
