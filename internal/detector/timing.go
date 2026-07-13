package detector

import (
	"math"
	"net/http"

	"detection-middleware/internal/store"
)

// humanCV is the coefficient of variation at (or above) which request timing
// looks irregular enough to be human. Below it, timing is suspiciously
// regular. Tunable.
const humanCV = 0.5

// timingSignal flags unnaturally regular request intervals
type timingSignal struct{}

func (timingSignal) Name() string { return "timing" }

func (timingSignal) Score(r *http.Request, state *store.ClientState) float64 {
	timings := state.RequestTimes
	n := len(timings)
	if n < 2 {
		return 0
	}

	intervals := make([]float64, n-1)
	var sum float64
	for i := 1; i < n; i++ {
		intervals[i-1] = timings[i].Sub(timings[i-1]).Seconds()
		sum += intervals[i-1]
	}
	mean := sum / float64(len(intervals))

	var variance float64
	for _, iv := range intervals {
		d := iv - mean // deviation from mean
		variance += d * d
	}
	variance /= float64(len(intervals))

	// Coefficient of variation: spread relative to the mean
	if mean <= 0 {
		return 0
	}
	cv := math.Sqrt(variance) / mean

	if cv >= humanCV {
		return 0
	}
	return 1 - cv/humanCV
}
