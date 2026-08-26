package perf_test

import (
	"testing"
	"time"

	"github.com/sebday/evoplayer/internal/perf"
)

func TestMetricsRecord(t *testing.T) {
	perf.Reset()
	perf.RecordRequest(10 * time.Millisecond)
	perf.RecordCacheHit()
	snap := perf.Snapshot()
	if snap.RequestCount != 1 {
		t.Fatalf("count = %d", snap.RequestCount)
	}
	if snap.CacheHits != 1 {
		t.Fatalf("hits = %d", snap.CacheHits)
	}
}
