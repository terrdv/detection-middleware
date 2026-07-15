// Package config holds tunable settings for the middleware: signal weights,
// score thresholds for each action tier, rate-limit window, and the listen
// address.
package config

import "time"

// Action is the decision the middleware takes for a request based on its
// aggregate suspicion score.
type Action int

const (
	ActionAllow Action = iota
	ActionChallenge
	ActionReject
)

// Config groups all tunable parameters.
type Config struct {
	Addr string
	// score < ChallengeThreshold        -> ActionAllow
	// ChallengeThreshold <= score < Reject -> ActionChallenge
	// score >= RejectThreshold           -> ActionReject
	ChallengeThreshold float64
	RejectThreshold    float64
	Weights map[string]float64

	// HoneypotPath is the hidden trap endpoint; any request to it is treated
	// as high-confidence bot traffic.
	HoneypotPath string

	// TrustForwardedFor controls how a client is identified. When false, the
	// key is the TCP source IP (RemoteAddr with the port stripped). When true,
	// the first hop in the X-Forwarded-For header is used instead — required
	// when running behind a trusted proxy, and handy for load tests that want
	// to simulate many distinct clients from one machine. Leave it false unless
	// a trusted proxy sets the header: clients can forge it otherwise.
	TrustForwardedFor bool
	// RateLimitWindow and RateLimitLimit bound how many requests a single
	// client may make within the sliding window before the rate_limit signal
	// starts contributing suspicion. The signal ramps to full suspicion at
	// 2*RateLimitLimit requests, so keep 2*RateLimitLimit <= the store's
	// maxSamples cap, or the count gets clipped and rate_limit can never reach
	// 1.0.
	RateLimitWindow time.Duration
	RateLimitLimit  int
}

// Starting configuration.
func Default() Config {
	weights := map[string]float64{
		"user_agent": 0.25, // ua
		"headers":    0.20, // req headers
		"timing":     0.50, // ex 500ms exact apart
		"rate_limit": 0.50, // too many requests / sliding window
		"honeypot":   1.00, // hit a trap endpoint -> instant reject
	}
	return Config{
		Addr:               ":8080",
		ChallengeThreshold: 0.4,
		RejectThreshold:    0.7,
		Weights:            weights,
		HoneypotPath:       "/wp-admin",
		RateLimitWindow:    time.Minute,
		// 2*50 = 100 <= maxSamples (128), so rate_limit can still saturate.
		RateLimitLimit: 50,
	}
}
