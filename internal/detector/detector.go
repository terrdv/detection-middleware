package detector

import (
	"net/http"

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
	//todo
	return &Detector{store: s, cfg: cfg}
}

type Result struct {
	Score  float64
	Action config.Action
	Contributions map[string]float64
}

// Evaluate scores a request across all signals and returns the decision.
func (d *Detector) Evaluate(r *http.Request) Result {
	//todo
	return Result{}
}
