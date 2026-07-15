package detector

import (
	"net/http/httptest"
	"testing"
	"time"

	"detection-middleware/internal/store"
)

// stateWithIntervals builds a ClientState from the given inter-request gaps.
func stateWithIntervals(gaps ...time.Duration) *store.ClientState {
	t := time.Now()
	times := []time.Time{t}
	for _, g := range gaps {
		t = t.Add(g)
		times = append(times, t)
	}
	return &store.ClientState{RequestTimes: times}
}

func TestTimingScore(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	sig := timingSignal{}

	t.Run("too few requests scores 0", func(t *testing.T) {
		if got := sig.Score(req, stateWithIntervals()); got != 0 {
			t.Errorf("got %v, want 0", got)
		}
		if got := sig.Score(req, &store.ClientState{RequestTimes: []time.Time{time.Now()}}); got != 0 {
			t.Errorf("single request: got %v, want 0", got)
		}
	})

	t.Run("perfectly regular intervals are maximally suspicious", func(t *testing.T) {
		s := stateWithIntervals(time.Second, time.Second, time.Second, time.Second)
		if got := sig.Score(req, s); got != 1 {
			t.Errorf("got %v, want 1", got)
		}
	})

	t.Run("highly irregular intervals are not suspicious", func(t *testing.T) {
		s := stateWithIntervals(100*time.Millisecond, 5*time.Second, 200*time.Millisecond, 8*time.Second)
		if got := sig.Score(req, s); got != 0 {
			t.Errorf("got %v, want 0", got)
		}
	})
}
