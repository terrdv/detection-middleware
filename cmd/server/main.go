// Command server runs a demo HTTP server wrapped with the bot-detection
// middleware. It exists to exercise the middleware end to end and to be a
// target for load testing.
package main

import (
	"log"
	"net/http"
	"os"

	"detection-middleware/internal/config"
	"detection-middleware/internal/detector"
	"detection-middleware/internal/middleware"
	"detection-middleware/internal/store"
)

func main() {
	cfg := config.Default()

	// For load testing: let the client self-identify via X-Forwarded-For so a
	// single machine can simulate many distinct clients. Off by default because
	// the header is spoofable — only enable it behind a trusted proxy or for
	// local benchmarking.
	if os.Getenv("TRUST_XFF") == "1" {
		cfg.TrustForwardedFor = true
		log.Print("TRUST_XFF=1: trusting X-Forwarded-For for client identity")
	}

	s := store.New(cfg.RateLimitWindow)
	d := detector.New(cfg, s)
	mw := middleware.New(d, cfg)

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok\n"))
	})

	log.Printf("listening on %s", cfg.Addr)
	if err := http.ListenAndServe(cfg.Addr, mw(mux)); err != nil {
		log.Fatal(err)
	}
}
