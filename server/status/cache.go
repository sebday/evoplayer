package status

import (
	"sync"

	"github.com/sebday/evoplayer/server/paths"
	"github.com/sebday/evoplayer/server/perf"
	"github.com/sebday/evoplayer/server/playback"
)

type cachedMeta struct {
	Title    string
	Artist   string
	Album    string
	Genre    string
	Year     string
	Label    string
	Art      string
	Waveform string
	Liked    bool
	Duration float64
}

var metaMu sync.RWMutex
var metaByPath = map[string]cachedMeta{}

func cacheMeta(st playback.Status) {
	if st.Path == "" {
		return
	}
	metaMu.Lock()
	metaByPath[st.Path] = cachedMeta{
		Title:    st.Title,
		Artist:   st.Artist,
		Album:    st.Album,
		Genre:    st.Genre,
		Year:     st.Year,
		Label:    st.Label,
		Art:      st.Art,
		Waveform: st.Waveform,
		Liked:    st.Liked,
		Duration: st.Duration,
	}
	metaMu.Unlock()
}

func applyCached(st playback.Status, m cachedMeta) playback.Status {
	st.Title = pick(m.Title, st.Title)
	st.Artist = m.Artist
	st.Album = m.Album
	st.Genre = m.Genre
	st.Year = m.Year
	st.Label = m.Label
	st.Art = m.Art
	st.Waveform = m.Waveform
	st.Liked = m.Liked
	if m.Duration > 0 && st.Duration <= 0 {
		st.Duration = m.Duration
		st = st.WithLabels()
	}
	return st
}

// EnrichLight reuses cached metadata for the current path and only updates transport fields.
func EnrichLight(env paths.Env, st playback.Status) playback.Status {
	if st.Path == "" {
		return mergeSavedPlaylist(env, st)
	}
	metaMu.RLock()
	m, ok := metaByPath[st.Path]
	metaMu.RUnlock()
	if ok {
		perf.RecordCacheHit()
		return mergeSavedPlaylist(env, applyCached(st, m))
	}
	perf.RecordCacheMiss()
	return EnrichFull(env, st)
}

// EnrichFull loads metadata once and caches it for subsequent light enrich calls.
func EnrichFull(env paths.Env, st playback.Status) playback.Status {
	enriched := enrichOnce(env, st)
	cacheMeta(enriched)
	return enriched
}

// InvalidateMeta drops cached metadata for a track path.
func InvalidateMeta(path string) {
	if path == "" {
		return
	}
	metaMu.Lock()
	delete(metaByPath, path)
	metaMu.Unlock()
}

// InvalidateAllMeta clears the metadata cache.
func InvalidateAllMeta() {
	metaMu.Lock()
	metaByPath = map[string]cachedMeta{}
	metaMu.Unlock()
}
