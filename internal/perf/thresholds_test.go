package perf_test

import (
	"testing"
	"time"

	"github.com/sebday/evoplayer/internal/perf"
)

const maxAvgRequest = 50 * time.Millisecond

func TestRegressionThresholds(t *testing.T) {
	perf.Reset()
	for i := 0; i < 20; i++ {
		perf.RecordRequest(2 * time.Millisecond)
	}
	perf.RecordCacheHit()
	perf.RecordCacheHit()
	perf.RecordCacheMiss()

	snap := perf.Snapshot()
	if snap.RequestCount != 20 {
		t.Fatalf("request count = %d", snap.RequestCount)
	}
	if snap.AvgRequest > maxAvgRequest {
		t.Fatalf("avg request = %v, want <= %v", snap.AvgRequest, maxAvgRequest)
	}
	if snap.CacheHits < 2 || snap.CacheMisses < 1 {
		t.Fatalf("cache hits=%d misses=%d", snap.CacheHits, snap.CacheMisses)
	}
}
