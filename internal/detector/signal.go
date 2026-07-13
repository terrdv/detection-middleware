package detector

import (
	"net/http"

	"detection-middleware/internal/store"
)

// Signal inspects a request (and the client's persisted state) and returns a suspicion contribution in the range [0, 1]
type Signal interface {
	Name() string
	Score(r *http.Request, state *store.ClientState) float64
}
