package detector

import (
	"net/http"

	"detection-middleware/internal/store"
)

// headersSignal flags requests missing headers real browsers typically send
// (Accept-Language, Accept-Encoding, Referer, etc.), which scripts often omit.
type headersSignal struct{}

func (headersSignal) Name() string { return "headers" }

func (headersSignal) Score(r *http.Request, _ *store.ClientState) float64 {
	return 0
}
