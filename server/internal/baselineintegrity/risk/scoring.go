package risk

import (
	"math"
	"time"
)

// Config defines scoring parameters.
// Values are intentionally opaque and server-controlled.
type Config struct {
	DecayFactor float64
	RiskCap     float64
}

// ApplyMatchRisk updates longitudinal risk with decay.
func ApplyMatchRisk(
	prev RiskState,
	match MatchRisk,
	cfg Config,
	now time.Time,
) RiskState {

	elapsed := now.Sub(prev.LastUpdate).Hours()
	if elapsed < 0 {
		elapsed = 0
	}

	decay := math.Pow(cfg.DecayFactor, elapsed)

	decayed := prev.TotalRisk * decay
	next := decayed + match.Value

	if next < 0 {
		next = 0
	}
	if next > cfg.RiskCap {
		next = cfg.RiskCap
	}

	return RiskState{
		PlayerID:   prev.PlayerID,
		TotalRisk:  next,
		LastUpdate: now,
	}
}
