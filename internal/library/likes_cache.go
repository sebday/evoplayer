package library

import (
	"encoding/json"
	"os"
	"sync"
)

var likesCaches sync.Map

type likesCache struct {
	mu     sync.RWMutex
	file   string
	set    map[string]struct{}
	loaded bool
}

func likesForEnv(env Env) *likesCache {
	if env.LikesFile == "" {
		return &likesCache{set: map[string]struct{}{}, loaded: true}
	}
	if v, ok := likesCaches.Load(env.LikesFile); ok {
		return v.(*likesCache)
	}
	c := &likesCache{file: env.LikesFile, set: map[string]struct{}{}}
	actual, _ := likesCaches.LoadOrStore(env.LikesFile, c)
	return actual.(*likesCache)
}

func (c *likesCache) loadLocked() {
	if c.loaded {
		return
	}
	c.loaded = true
	raw, err := os.ReadFile(c.file)
	if err != nil {
		return
	}
	var likes map[string]any
	if json.Unmarshal(raw, &likes) != nil {
		return
	}
	for p := range likes {
		c.set[p] = struct{}{}
	}
}

func (c *likesCache) preload() {
	c.mu.Lock()
	c.loadLocked()
	c.mu.Unlock()
}

func (c *likesCache) has(path string) bool {
	c.mu.Lock()
	c.loadLocked()
	_, ok := c.set[path]
	c.mu.Unlock()
	return ok
}

// InvalidateLikesCache drops the in-memory likes map for a file path.
func InvalidateLikesCache(file string) {
	if file == "" {
		return
	}
	if v, ok := likesCaches.Load(file); ok {
		c := v.(*likesCache)
		c.mu.Lock()
		c.loaded = false
		c.set = map[string]struct{}{}
		c.mu.Unlock()
	}
}
