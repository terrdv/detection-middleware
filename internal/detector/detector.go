package detector

import (
	"net"
	"net/http"
	"strings"
	"time"

	"detection-middleware/internal/config"
	"detection-middleware/internal/store"
)

// Detector aggregates a set of Signals into a single weighted score and maps
// that score onto a config.Action tier.
type Detector struct {
	signals []Signal
	store   *store.Store
	cfg     config.Config
}

func New(cfg config.Config, s *store.Store) *Detector {
	return &Detector{
		signals: []Signal{
			userAgentSignal{},
			headersSignal{},
			timingSignal{},
			rateLimitSignal{window: cfg.RateLimitWindow, limit: cfg.RateLimitLimit},
			honeypotSignal{path: cfg.HoneypotPath},
		},
		store: s,
		cfg:   cfg,
	}
}

type Result struct {
	Score         float64
	Action        config.Action
	Contributions map[string]float64
}

// clientKey derives the identity a request is tracked under. By default this is
// the TCP source IP with the port stripped, so every request from one client
// shares state (r.RemoteAddr is "IP:port", and the port changes per connection,
// so using it raw would give each request its own bucket). When configured to
// trust it, the first hop of X-Forwarded-For is used instead.
func (d *Detector) clientKey(r *http.Request) string {
	if d.cfg.TrustForwardedFor {
		if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
			// The list is "client, proxy1, proxy2..."; the first entry is the
			// original client.
			first, _, _ := strings.Cut(xff, ",")
			return strings.TrimSpace(first)
		}
	}

	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		// No port present (or malformed) — fall back to the raw value.
		return r.RemoteAddr
	}
	return host
}

// Evaluate scores a request across all signals and returns the decision.
func (d *Detector) Evaluate(r *http.Request) Result {
	clientState := d.store.Record(d.clientKey(r), time.Now())

	contributions := make(map[string]float64, len(d.signals))
	var score float64
	for _, s := range d.signals {
		raw := s.Score(r, clientState) // [0,1]
		w := d.cfg.Weights[s.Name()]
		contributions[s.Name()] = raw * w
		score += raw * w
	}

	var action config.Action
	if score < d.cfg.ChallengeThreshold {
		action = config.ActionAllow
	} else if score >= d.cfg.RejectThreshold {
		action = config.ActionReject
	} else {
		action = config.ActionChallenge
	}

	return Result{score, action, contributions}
}
