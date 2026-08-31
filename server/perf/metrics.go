package perf

import (
	"sync/atomic"
	"time"
)

// Metrics collects lightweight runtime counters for regression tests and debug logs.
type Metrics struct {
	RequestCount   atomic.Uint64
	RequestTotalNs atomic.Uint64
	CacheHits      atomic.Uint64
	CacheMisses    atomic.Uint64
	IPCQueueDepth  atomic.Int64
}

var global Metrics

func RecordRequest(d time.Duration) {
	global.RequestCount.Add(1)
	global.RequestTotalNs.Add(uint64(d.Nanoseconds()))
}

func RecordCacheHit()  { global.CacheHits.Add(1) }
func RecordCacheMiss() { global.CacheMisses.Add(1) }

func IncIPCQueue() { global.IPCQueueDepth.Add(1) }
func DecIPCQueue() { global.IPCQueueDepth.Add(-1) }
