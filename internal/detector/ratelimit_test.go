package detector

import (
	"net/http/httptest"
	"testing"
	"time"

	"detection-middleware/internal/store"
)

// stateWithRequests builds a ClientState whose RequestTimes are n timestamps
// spaced 1s apart, ending now.
func stateWithRequests(n int) *store.ClientState {
	now := time.Now()
	times := make([]time.Time, n)
	for i := 0; i < n; i++ {
		times[i] = now.Add(-time.Duration(n-1-i) * time.Second)
	}
	return &store.ClientState{RequestTimes: times}
}

func TestRateLimitScore(t *testing.T) {
	sig := rateLimitSignal{window: time.Minute, limit: 10}
	req := httptest.NewRequest("GET", "/", nil)

	cases := []struct {
		name  string
		count int
		want  float64
	}{
		{"under limit", 5, 0},
		{"at limit", 10, 0},
		{"one over ramps", 11, 0.1},
		{"halfway up ramp", 15, 0.5},
		{"at twice limit saturates", 20, 1},
		{"far over stays capped", 100, 1},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := sig.Score(req, stateWithRequests(tc.count))
			if diff := got - tc.want; diff > 1e-9 || diff < -1e-9 {
				t.Errorf("count=%d: got %v, want %v", tc.count, got, tc.want)
			}
		})
	}
}

// Requests older than the window must not count toward the limit.
func TestRateLimitScoreIgnoresOldRequests(t *testing.T) {
	sig := rateLimitSignal{window: time.Minute, limit: 10}
	req := httptest.NewRequest("GET", "/", nil)

	now := time.Now()
	var times []time.Time
	for i := 0; i < 50; i++ { // 50 requests, all ~1 hour ago
		times = append(times, now.Add(-time.Hour).Add(time.Duration(i)*time.Second))
	}
	state := &store.ClientState{RequestTimes: times}

	if got := sig.Score(req, state); got != 0 {
		t.Errorf("stale requests should score 0, got %v", got)
	}
}

func TestRateLimitScoreGuards(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	state := stateWithRequests(50)

	if got := (rateLimitSignal{window: time.Minute, limit: 0}).Score(req, state); got != 0 {
		t.Errorf("zero limit should score 0, got %v", got)
	}
	if got := (rateLimitSignal{window: 0, limit: 10}).Score(req, state); got != 0 {
		t.Errorf("zero window should score 0, got %v", got)
	}
}
