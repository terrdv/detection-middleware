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
	// RateLimitWindow and RateLimitLimit bound how many requests a single
	// client may make within the sliding window before the rate_limit signal
	// starts contributing suspicion.
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
		RateLimitLimit:     100,
	}
}
