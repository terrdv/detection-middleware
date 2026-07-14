// Package store keeps per-client state that must persist across requests,
// such as request counts and timestamps used for rate limiting and timing
// analysis. The MVP uses an in-memory map guarded by a mutex, keyed by IP.
//
package store

import (
	"sync"
	"time"
)

// ClientState holds the rolling state tracked for a single client.
type ClientState struct {
	RequestTimes []time.Time
}


type Store struct {
	mu      sync.Mutex
	clients map[string]*ClientState
}

func New() *Store {
	return &Store{clients: make(map[string]*ClientState)}
}

// Get returns the state for key, creating it if absent.
// Concurrent safe bc if another calls the store it blocks and waits until lock is released
func (s *Store) Get(key string) *ClientState {
	s.mu.Lock()
	defer s.mu.Unlock()
	value, ok := s.clients[key]
	if !ok {
		value = &ClientState{}
		s.clients[key] = value
	}
	return value
}

func (s *Store) Record(key string, t time.Time) *ClientState {
	s.mu.Lock()
	defer s.mu.Unlock()
	st := s.clients[key]
	if st == nil {
		st = &ClientState{}
		s.clients[key] = st
	}
	st.RequestTimes = append(st.RequestTimes, t)

	//todo trim entries
	return st
}