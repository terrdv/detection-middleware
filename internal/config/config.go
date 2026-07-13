// Package config holds tunable settings for the middleware: signal weights,
// score thresholds for each action tier, rate-limit window, and the listen
// address.
package config

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
}

// Starting configuration.
func Default() Config {
	weights := map[string]float64{
		"user_agent": 0.25, // ua
		"headers":    0.20, // req headers
		"timing":     0.50, // ex 500ms exact apart
	}
	return Config{":8080", 0.4, 0.7, weights}
}
