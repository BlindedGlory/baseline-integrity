package risk

import (
	"errors"
	"time"

	baselineintegrityv1 "github.com/BlindedGlory/baseline-integrity/gen/go/baselineintegrity/v1"
)

var (
	ErrNilPlayerAggs      = errors.New("nil player aggregates")
	ErrSchemaMismatch     = errors.New("telemetry schema mismatch")
	ErrMissingPlayerID    = errors.New("missing player_id")
	ErrMissingSessionRef  = errors.New("missing session ref")
	ErrUnsupportedPctlKey = errors.New("unsupported percentile key")
)

// MapAggregatesToMatchRisk converts one player's aggregates into a per-match risk contribution.
// This does NOT make enforcement decisions. It produces a bounded signal used by longitudinal scoring.
func MapAggregatesToMatchRisk(
	p *baselineintegrityv1.PlayerAggregates,
	cfg MappingConfig,
	at time.Time,
) (MatchRisk, error) {

	if p == nil {
		return MatchRisk{}, ErrNilPlayerAggs
	}

	ref := p.GetRef()
	if ref == nil {
		return MatchRisk{}, ErrMissingSessionRef
	}
	playerID := ref.GetPlayerId()
	if playerID == "" {
		return MatchRisk{}, ErrMissingPlayerID
	}

	// Schema guardrail: reject unexpected schema to prevent mixing meanings.
	if cfg.ExpectedSchemaID != "" && p.GetTelemetrySchemaId() != cfg.ExpectedSchemaID {
		return MatchRisk{}, ErrSchemaMismatch
	}

	perCap := cfg.PerSignalCap
	if perCap <= 0 {
		perCap = 1.0
	}

	total := 0.0

	// Counter rules
	for name, rule := range cfg.Counters {
		raw := float64(GetCounter(p, name))
		norm := rule.Normalization
		if norm > 0 {
			raw = raw / norm
		}

		// Treat "0 is normal" and "higher is more suspicious".
		// Convert to a contribution using a soft curve.
		contrib := SoftScore(raw*rule.Weight, perCap)
		total += contrib
	}

	// Quantile rules
	for name, rule := range cfg.Quantiles {
		q := GetQuantiles(p, name)
		if q == nil {
			continue
		}

		v, err := pickPercentile(q, rule.Pctl)
		if err != nil {
			return MatchRisk{}, err
		}

		// Higher-than-baseline is treated as suspicious by default (game-specific tuning).
		// If a signal is inverted (lower is suspicious), represent it by negating Mean/Weight
		// in server config rather than adding special-case code.
		z := ZScore(v, rule.Mean, rule.Std)

		contrib := SoftScore(z*rule.Weight, perCap)
		total += contrib
	}

	return MatchRisk{
		PlayerID: playerID,
		Value:    total,
		At:       at,
	}, nil
}

func pickPercentile(q *baselineintegrityv1.Quantiles, pctl string) (float64, error) {
	switch pctl {
	case "p50":
		return q.GetP50(), nil
	case "p75":
		return q.GetP75(), nil
	case "p90":
		return q.GetP90(), nil
	case "p95":
		return q.GetP95(), nil
	case "p99":
		return q.GetP99(), nil
	default:
		return 0, ErrUnsupportedPctlKey
	}
}
