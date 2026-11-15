package playback

import (
	"os"
	"slices"
)

// SetFollow is the ordered list of tracks to play after the queue ends.
func (a *Actor) SetFollow(paths []string) {
	a.dispatch(func() {
		next := append([]string(nil), paths...)
		a.mu.Lock()
		if a.followIdx >= 0 && a.path != "" {
			if i := slices.Index(next, a.path); i >= 0 {
				a.followIdx = i
			}
		}
		a.follow = next
		if a.followWait >= len(next) {
			a.followWait = -1
		}
		a.mu.Unlock()
	})
}

// SetFollowReady reports whether a follow path can be played yet.
func (a *Actor) SetFollowReady(fn func(string) bool) {
	a.mu.Lock()
	a.followReady = fn
	a.mu.Unlock()
}

// SetOnFollow is called when the next follow track is not ready to play.
func (a *Actor) SetOnFollow(fn func(int, string)) {
	a.mu.Lock()
	a.onFollow = fn
	a.mu.Unlock()
}

// ContinueFollow plays a follow track once its file is ready.
func (a *Actor) ContinueFollow(index int) error {
	return a.dispatchErr(func() error {
		a.mu.Lock()
		if a.followWait != index {
			a.mu.Unlock()
			return nil
		}
		a.mu.Unlock()
		return a.playFollow(index)
	})
}

func (a *Actor) playFollow(index int) error {
	a.mu.Lock()
	if index < 0 || index >= len(a.follow) {
		a.mu.Unlock()
		a.finishFollow()
		return nil
	}
	path := a.follow[index]
	readyFn := a.followReady
	cb := a.onFollow
	a.mu.Unlock()
	if !followPathReady(path, readyFn) {
		a.mu.Lock()
		a.followIdx = index
		a.followWait = index
		a.detached = false
		a.mu.Unlock()
		if cb != nil {
			cb(index, path)
		}
		return nil
	}
	a.mu.Lock()
	if index < 0 || index >= len(a.follow) || a.follow[index] != path {
		a.mu.Unlock()
		return nil
	}
	queue, follow := takeFollow(a.queue, a.follow, index)
	a.queue = queue
	a.follow = follow
	a.index = slices.Index(queue, path)
	if a.index < 0 {
		a.index = 0
	}
	if a.shuffle && !slices.Contains(a.shuffleOrd, a.index) {
		a.shuffleOrd = append(a.shuffleOrd, a.index)
	}
	a.followIdx = -1
	a.followWait = -1
	a.detached = false
	a.continued = true
	a.bumpQueueRevLocked()
	a.mu.Unlock()
	return a.loadPath(path, 0, false)
}

// takeFollow moves follow[index] onto the end of the queue.
func takeFollow(queue, follow []string, index int) ([]string, []string) {
	path := follow[index]
	nextFollow := make([]string, 0, len(follow)-1)
	nextFollow = append(nextFollow, follow[:index]...)
	nextFollow = append(nextFollow, follow[index+1:]...)
	if slices.Contains(queue, path) {
		return queue, nextFollow
	}
	nextQueue := make([]string, len(queue)+1)
	copy(nextQueue, queue)
	nextQueue[len(queue)] = path
	return nextQueue, nextFollow
}

func (a *Actor) finishFollow() {
	a.mu.Lock()
	a.followIdx = -1
	a.followWait = -1
	a.mu.Unlock()
	a.haltPlayback()
	a.emit()
}

func followPathReady(path string, ready func(string) bool) bool {
	if path == "" {
		return false
	}
	if ready != nil {
		return ready(path)
	}
	st, err := os.Stat(path)
	return err == nil && !st.IsDir() && st.Size() > 0
}

func (a *Actor) advancesPastEndLocked(step int) bool {
	if len(a.queue) == 0 {
		return false
	}
	if a.shuffle {
		if len(a.shuffleOrd) != len(a.queue) {
			next := a.index + step
			return next < 0 || next >= len(a.queue)
		}
		pos := slices.Index(a.shuffleOrd, a.index)
		if pos < 0 {
			pos = 0
		}
		pos += step
		return pos < 0 || pos >= len(a.shuffleOrd)
	}
	next := a.index + step
	return next < 0 || next >= len(a.queue)
}

// followAdvance chooses what plays when the current track finishes.
func followAdvance(followIdx, followN int, repeat, wrap, continued bool) (string, int) {
	if followIdx >= 0 {
		if repeat {
			return "replay", followIdx
		}
		if followIdx+1 < followN {
			return "follow", followIdx + 1
		}
		return "stop", -1
	}
	if !repeat && wrap && followN > 0 {
		return "follow", 0
	}
	if !repeat && wrap && continued {
		return "stop", -1
	}
	return "queue", -1
}

// followSkip chooses what plays for next and previous.
func followSkip(followIdx, followN int, wrap, continued bool, dir int) (string, int) {
	if dir > 0 {
		if followIdx >= 0 {
			if followIdx+1 < followN {
				return "follow", followIdx + 1
			}
			return "stop", -1
		}
		if wrap && followN > 0 {
			return "follow", 0
		}
		if wrap && continued {
			return "stop", -1
		}
		return "queue", -1
	}
	if followIdx > 0 {
		return "follow", followIdx - 1
	}
	if followIdx == 0 {
		return "back", -1
	}
	return "queue", -1
}
