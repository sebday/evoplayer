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
	a.followIdx = index
	a.detached = false
	a.followWait = -1
	a.mu.Unlock()
	if !followPathReady(path, readyFn) {
		a.mu.Lock()
		a.followWait = index
		a.mu.Unlock()
		if cb != nil {
			cb(index, path)
		}
		return nil
	}
	return a.loadPath(path, 0, false)
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
func followAdvance(followIdx, followN int, repeat, wrap bool) (string, int) {
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
	return "queue", -1
}

// followSkip chooses what plays for next and previous.
func followSkip(followIdx, followN int, wrap bool, dir int) (string, int) {
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
