package detector

import (
	"net/http"
	"time"

	"detection-middleware/internal/store"
)

type rateLimitSignal struct {
	window time.Duration
	limit  int
}

func (rateLimitSignal) Name() string { return "rate_limit" }

func (s rateLimitSignal) Score(r *http.Request, state *store.ClientState) float64 {

}
