package warm

import (
	"sync"

	"github.com/sebday/evoplayer/server/library"
	"github.com/sebday/evoplayer/server/paths"
)

// DefaultWorkers is the parallel warm worker count for batch IPC and CLI.
const DefaultWorkers = 8

func BatchThumbs(env paths.Env, trackPaths []string, workers int) ([]Result, error) {
	if workers <= 0 {
		workers = DefaultWorkers
	}
	libEnv := library.EnvFrom(env)
	type job struct {
		idx  int
		path string
	}
	jobs := make(chan job)
	results := make([]Result, len(trackPaths))
	var wg sync.WaitGroup
	worker := func() {
		defer wg.Done()
		for j := range jobs {
			row, err := library.Meta(libEnv, j.path, "")
			if err != nil {
				results[j.idx] = Result{Path: j.path}
				continue
			}
			res := Result{
				Path:  j.path,
				Art:   row.Art,
				Thumb: row.Thumb,
			}
			if res.Art != "" && (res.Thumb == "" || !assetExists(res.Thumb)) {
				if thumb, err := library.EnsureThumb(libEnv, res.Art); err == nil {
					res.Thumb = thumb
				}
			}
			results[j.idx] = res
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

func BatchTracks(env paths.Env, trackPaths []string, workers int) ([]Result, error) {
	if workers <= 0 {
		workers = DefaultWorkers
	}
	type job struct {
		idx  int
		path string
	}
	jobs := make(chan job)
	results := make([]Result, len(trackPaths))
	var wg sync.WaitGroup
	worker := func() {
		defer wg.Done()
		for j := range jobs {
			res, err := TrackAssets(env, j.path)
			if err != nil {
				results[j.idx] = Result{Path: j.path}
				continue
			}
			results[j.idx] = res
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
