package store

import (
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// TestConcurrentRecordNoRace hammers a single client key from many goroutines
// and reads each returned snapshot, which is exactly the pattern that raced
// before Record started copying under the lock. Run with -race to verify:
//
//	go test -race -run TestConcurrentRecordNoRace ./internal/store/
//
// Without the snapshot, ranging over the returned slice while another goroutine
// appends to the same backing array trips the race detector.
func TestConcurrentRecordNoRace(t *testing.T) {
	s := New(time.Minute)

	const goroutines = 50
	const perGoroutine = 2000

	var wg sync.WaitGroup
	for range goroutines {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range perGoroutine {
				// Shared key: every goroutine contends on the same ClientState.
				st := s.Record("shared", time.Now())
				// Read the snapshot outside any lock — this is the caller's job
				// (the signals do exactly this).
				var sum int64
				for _, ts := range st.RequestTimes {
					sum += ts.UnixNano()
				}
				_ = sum
			}
		}()
	}
	wg.Wait()
}

// BenchmarkRecordSharedKey is the worst case: every parallel goroutine records
// under one key, so they contend on the mutex AND each snapshot copies a full
// (maxSamples-capped) slice. This is the number lock striping should improve.
//
//	go test -bench BenchmarkRecordSharedKey -benchmem ./internal/store/
func BenchmarkRecordSharedKey(b *testing.B) {
	s := New(time.Minute)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			s.Record("shared", time.Now())
		}
	})
}

// BenchmarkRecordSharedKey1000 piles ~1000 goroutines onto one key (GOMAXPROCS
// goroutines × SetParallelism). It doesn't raise throughput over the 8-core
// shared-key run — the single mutex serializes them — it just deepens the wait
// queue. Compare its ns/op to BenchmarkRecordSharedKey-8 to see contention, not
// speedup.
func BenchmarkRecordSharedKey1000(b *testing.B) {
	s := New(time.Minute)
	b.SetParallelism(125) // 125 × 8 cores ≈ 1000 goroutines
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			s.Record("shared", time.Now())
		}
	})
}

// BenchmarkRecordDistinctKeys isolates map/mutex contention from copy cost:
// each goroutine uses its own key, so slices stay short but every write still
// serializes on the single global mutex.
func BenchmarkRecordDistinctKeys(b *testing.B) {
	s := New(time.Minute)
	var counter int64
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		// Each parallel worker (goroutine) gets a unique key.
		key := "client-" + strconv.FormatInt(atomic.AddInt64(&counter, 1), 10)
		for pb.Next() {
			s.Record(key, time.Now())
		}
	})
}
