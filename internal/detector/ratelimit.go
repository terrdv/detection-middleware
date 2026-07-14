package detector

import (
	"net/http"
	"time"

	"detection-middleware/internal/store"
)

// rateLimitSignal flags clients whose request rate over a sliding time window
// exceeds a configured limit. Unlike a hard rate limiter it returns a graded
// suspicion: 0 up to the limit, ramping to 1 as the count reaches twice the
// limit.
type rateLimitSignal struct {
	window time.Duration
	limit  int
}

func (rateLimitSignal) Name() string { return "rate_limit" }

func (s rateLimitSignal) Score(r *http.Request, state *store.ClientState) float64 {
	if s.limit <= 0 || s.window <= 0 {
		return 0
	}

	// Count requests that fall inside the sliding window ending now.
	cutoff := time.Now().Add(-s.window)
	var count int
	for _, t := range state.RequestTimes {
		if t.After(cutoff) {
			count++
		}
	}

	if count <= s.limit {
		return 0
	}

	// Ramp from 0 at the limit to 1 at twice the limit.
	over := float64(count-s.limit) / float64(s.limit)
	if over >= 1 {
		return 1
	}
	return over
}
