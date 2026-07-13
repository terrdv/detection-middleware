package detector

import (
	"net/http"

	"detection-middleware/internal/store"
)

// expectedHeaders are headers a normal browser sends on a top-level request
// but scripting tools often omit. Referer is intentionally excluded: browsers
// leave it off on the first/direct navigation, so its absence is not by itself
// suspicious.
var expectedHeaders = []string{
	"Accept",
	"Accept-Language",
	"Accept-Encoding",
}

// headersSignal flags requests missing headers real browsers typically send
// (Accept-Language, Accept-Encoding, etc.), which scripts often omit.
type headersSignal struct{}

func (headersSignal) Name() string { return "headers" }

func (headersSignal) Score(r *http.Request, _ *store.ClientState) float64 {
	missing := 0
	for _, h := range expectedHeaders {
		if r.Header.Get(h) == "" {
			missing++
		}
	}

	// Graded: the more expected headers are absent, the more botlike.
	return float64(missing) / float64(len(expectedHeaders))
}
