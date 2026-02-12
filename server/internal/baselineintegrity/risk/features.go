package risk

import (
	"math"

	baselineintegrityv1 "github.com/BlindedGlory/baseline-integrity/gen/go/baselineintegrity/v1"
)

// GetCounter returns a counter value by name (0 if missing).
func GetCounter(p *baselineintegrityv1.PlayerAggregates, name string) uint64 {
	if p == nil {
		return 0
	}
	for _, c := range p.Counters {
		if c.GetName() == name {
			return c.GetValue()
		}
	}
	return 0
}

// GetQuantiles returns a quantiles struct by name (nil if missing).
func GetQuantiles(p *baselineintegrityv1.PlayerAggregates, name string) *baselineintegrityv1.Quantiles {
	if p == nil {
		return nil
	}
	for _, q := range p.Quantiles {
		if q.GetName() == name {
			return q
		}
	}
	return nil
}

// SoftScore turns an unbounded "how bad is this" number into a bounded contribution.
// This avoids crisp thresholds and makes tuning safer.
//
// input: x where 0 is normal, positive means suspicious.
// output: [0, cap]
func SoftScore(x float64, cap float64) float64 {
	if cap <= 0 {
		return 0
	}
	if x <= 0 {
		return 0
	}

	// Saturating curve: cap * (1 - exp(-x))
	return cap * (1.0 - math.Exp(-x))
}

// ZScore computes (v - mean) / std, with guardrails.
func ZScore(v, mean, std float64) float64 {
	if std <= 0 {
		return 0
	}
	return (v - mean) / std
}
