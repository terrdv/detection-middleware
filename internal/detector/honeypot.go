package detector

import (
	"net/http"

	"detection-middleware/internal/store"
)

// honeypotSignal returns a high-confidence score when a request targets a hidden trap endpoint
type honeypotSignal struct {
	path string // trap endpoint
}

func (honeypotSignal) Name() string { return "honeypot" }

func (s honeypotSignal) Score(r *http.Request, _ *store.ClientState) float64 {
	if s.path == r.URL.Path {
		return 1
	}
	return 0
}
