package detector

import (
	"net/http/httptest"
	"testing"

	"detection-middleware/internal/config"
	"detection-middleware/internal/store"
)

func TestHoneypotScore(t *testing.T) {
	sig := honeypotSignal{path: "/wp-admin"}

	hit := httptest.NewRequest("GET", "/wp-admin", nil)
	if got := sig.Score(hit, nil); got != 1 {
		t.Errorf("trap path: got %v, want 1", got)
	}

	miss := httptest.NewRequest("GET", "/", nil)
	if got := sig.Score(miss, nil); got != 0 {
		t.Errorf("normal path: got %v, want 0", got)
	}
}

// A honeypot hit alone should exceed the reject threshold under Default config.
func TestEvaluateHoneypotRejects(t *testing.T) {
	cfg := config.Default()
	d := New(cfg, store.New(cfg.RateLimitWindow))

	res := d.Evaluate(httptest.NewRequest("GET", cfg.HoneypotPath, nil))
	if res.Action != config.ActionReject {
		t.Errorf("honeypot hit: got action %v, want Reject (score=%v)", res.Action, res.Score)
	}
}

// A plain request from a normal browser should be allowed.
func TestEvaluateNormalRequestAllowed(t *testing.T) {
	cfg := config.Default()
	d := New(cfg, store.New(cfg.RateLimitWindow))

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36")
	req.Header.Set("Accept", "text/html")
	req.Header.Set("Accept-Language", "en-US")
	req.Header.Set("Accept-Encoding", "gzip, deflate, br")

	res := d.Evaluate(req)
	if res.Action != config.ActionAllow {
		t.Errorf("normal request: got action %v, want Allow (score=%v, contributions=%v)",
			res.Action, res.Score, res.Contributions)
	}
}
