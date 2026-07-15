// Package store keeps per-client state that must persist across requests,
// such as request counts and timestamps used for rate limiting and timing
// analysis. The MVP uses an in-memory map guarded by a mutex, keyed by IP.
//
package store

import (
	"hash/fnv"
	"sync"
	"time"
)

// ClientState holds the rolling state tracked for a single client.
type ClientState struct {
	RequestTimes []time.Time
}

type Store struct {
	shards [256]Shard
	retention time.Duration
	maxSamples int
}

type Shard struct {
	mu      sync.Mutex
	clients map[string]*ClientState
}

func New(retention time.Duration) *Store {
	s := &Store{
		retention:  retention,
		maxSamples: 128,
	}
	for i := range s.shards {
		s.shards[i].clients = make(map[string]*ClientState)
	}
	return s
}

func shardIndex(key string) uint32 {
    h := fnv.New32a()
    h.Write([]byte(key))
    return h.Sum32() & 255
}
// Get returns the state for key, creating it if absent.
// Concurrent safe bc if another calls the store it blocks and waits until lock is released
func (s *Store) Get(key string) *ClientState {
	shard := &s.shards[shardIndex(key)]
	shard.mu.Lock()
	defer shard.mu.Unlock()
	value, ok := shard.clients[key]
	if !ok {
		value = &ClientState{}
		shard.clients[key] = value
	}
	return value
}

func (s *Store) Record(key string, t time.Time) *ClientState {
	shard := &s.shards[shardIndex(key)]
	shard.mu.Lock()
	defer shard.mu.Unlock()
	st := shard.clients[key]
	if st == nil {
		st = &ClientState{}
		shard.clients[key] = st
	}
	st.RequestTimes = append(st.RequestTimes, t)

	// Drop timestamps older than the retention window. RequestTimes is appended
	// in time order, so the stale entries are always a prefix.
	if s.retention > 0 {
		cutoff := t.Add(-s.retention)
		i := 0
		for i < len(st.RequestTimes) && st.RequestTimes[i].Before(cutoff) {
			i++
		}
		if i > 0 {
			st.RequestTimes = st.RequestTimes[i:]
		}
	}

	// Backstop: cap the slice even if a burst fills the whole window.
	if s.maxSamples > 0 && len(st.RequestTimes) > s.maxSamples {
		st.RequestTimes = st.RequestTimes[len(st.RequestTimes)-s.maxSamples:]
	}

	// Return a snapshot, not the live state. The caller (signals) reads this
	// outside the lock, so it must not share the backing array with st, which a
	// later Record from another goroutine will mutate. Copy under the lock.
	snapshot := make([]time.Time, len(st.RequestTimes))
	copy(snapshot, st.RequestTimes)
	return &ClientState{RequestTimes: snapshot}
}