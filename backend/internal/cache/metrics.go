package cache

import (
	"sync/atomic"
	"time"
)

type CacheMetrics struct {
	Hits   int64 `json:"hits"`
	Misses int64 `json:"misses"`
	Errors int64 `json:"errors"`

	Sets      int64 `json:"sets"`
	Deletes   int64 `json:"deletes"`
	StartTime int64 `json:"start_time"`
}

func NewCacheMetrics() *CacheMetrics {
	return &CacheMetrics{
		StartTime: time.Now().Unix(),
	}
}

func (m *CacheMetrics) RecordHit() {
	atomic.AddInt64(&m.Hits, 1)
}

func (m *CacheMetrics) RecordMiss() {
	atomic.AddInt64(&m.Misses, 1)
}

func (m *CacheMetrics) RecordError() {
	atomic.AddInt64(&m.Errors, 1)
}

func (m *CacheMetrics) RecordSet() {
	atomic.AddInt64(&m.Sets, 1)
}

func (m *CacheMetrics) RecordDelete() {
	atomic.AddInt64(&m.Deletes, 1)
}

func (m *CacheMetrics) GetStats() CacheMetrics {
	return CacheMetrics{
		Hits:      atomic.LoadInt64(&m.Hits),
		Misses:    atomic.LoadInt64(&m.Misses),
		Errors:    atomic.LoadInt64(&m.Errors),
		Sets:      atomic.LoadInt64(&m.Sets),
		Deletes:   atomic.LoadInt64(&m.Deletes),
		StartTime: m.StartTime,
	}
}

func (m *CacheMetrics) HitRate() float64 {
	hits := atomic.LoadInt64(&m.Hits)
	misses := atomic.LoadInt64(&m.Misses)
	total := hits + misses

	if total == 0 {
		return 0.0
	}

	return float64(hits) / float64(total) * 100.0
}

func (m *CacheMetrics) Reset() {
	atomic.StoreInt64(&m.Hits, 0)
	atomic.StoreInt64(&m.Misses, 0)
	atomic.StoreInt64(&m.Errors, 0)
	atomic.StoreInt64(&m.Sets, 0)
	atomic.StoreInt64(&m.Deletes, 0)
	m.StartTime = time.Now().Unix()
}
