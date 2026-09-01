package warm

import (
	"sync"
	"sync/atomic"

	"github.com/sebday/evoplayer/server/library"
	"github.com/sebday/evoplayer/server/paths"
)

// DefaultWorkers is the parallel warm worker count for batch IPC and CLI.
const DefaultWorkers = 8

// BatchProgress reports warm-all progress.
type BatchProgress struct {
	Done   int
	Total  int
	Folder string
}

// BatchProgressFunc is called as batch warm advances.
type BatchProgressFunc func(BatchProgress)

// BatchTracks warms album art and thumbs for each path.
func BatchTracks(env paths.Env, trackPaths []string, workers int, onProgress BatchProgressFunc) ([]Result, error) {
	if workers <= 0 {
		workers = DefaultWorkers
	}
	if workers > len(trackPaths) {
		workers = len(trackPaths)
	}
	if workers == 0 {
		return nil, nil
	}
	type job struct {
		idx  int
		path string
	}
	jobs := make(chan job)
	results := make([]Result, len(trackPaths))
	var done atomic.Int64
	var wg sync.WaitGroup
	worker := func() {
		defer wg.Done()
		for j := range jobs {
			res, err := TrackAssets(env, j.path)
			if err != nil {
				results[j.idx] = Result{Path: j.path}
			} else {
				results[j.idx] = res
			}
			cur := int(done.Add(1))
			if onProgress != nil {
				onProgress(BatchProgress{
					Done:   cur,
					Total:  len(trackPaths),
					Folder: library.CacheFolderLabel(library.EnvFrom(env), j.path),
				})
			}
		}
	}
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go worker()
	}
	for i, path := range trackPaths {
		if path == "" {
			continue
		}
		jobs <- job{idx: i, path: path}
	}
	close(jobs)
	wg.Wait()
	return results, nil
}
