package perf

import (
	"sync"
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

func Snapshot() SnapshotData {
	count := global.RequestCount.Load()
	total := global.RequestTotalNs.Load()
	avg := time.Duration(0)
	if count > 0 {
		avg = time.Duration(int64(total / count))
	}
	return SnapshotData{
		RequestCount:  count,
		AvgRequest:    avg,
		CacheHits:     global.CacheHits.Load(),
		CacheMisses:   global.CacheMisses.Load(),
		IPCQueueDepth: global.IPCQueueDepth.Load(),
	}
}

type SnapshotData struct {
	RequestCount  uint64
	AvgRequest    time.Duration
	CacheHits     uint64
	CacheMisses   uint64
	IPCQueueDepth int64
}

var resetMu sync.Mutex

func Reset() {
	resetMu.Lock()
	defer resetMu.Unlock()
	global = Metrics{}
}
