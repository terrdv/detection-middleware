// Package middleware adapts the detector into the idiomatic net/http
// middleware pattern: a function that wraps an http.Handler and returns a new
// one which scores each request before deciding whether to pass it through,
// challenge it, or reject it.
package middleware

import (
	"net/http"

	"detection-middleware/internal/config"
	"detection-middleware/internal/detector"
)

func New(d *detector.Detector, cfg config.Config) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {

		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			result := d.Evaluate(r)
			if result.Action == config.ActionAllow {
				next.ServeHTTP(w, r)
			} else if result.Action == config.ActionChallenge {
				http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
			} else if result.Action == config.ActionReject {
				http.Error(w, "Forbidden", http.StatusForbidden)
			}

		})
	}
}
