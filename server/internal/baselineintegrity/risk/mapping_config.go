package risk

// MappingConfig defines how telemetry maps into per-match risk.
// All values are server-controlled. This file contains structure only;
// values should live in a JSON config file in deployment.
type MappingConfig struct {
	// Expected telemetry schema guardrail.
	ExpectedSchemaID string `json:"expected_schema_id"`

	// Controls overall contribution shape.
	PerSignalCap float64 `json:"per_signal_cap"`

	// Counter mappings: (counter per match) -> risk.
	Counters map[string]CounterRule `json:"counters"`

	// Quantile mappings: (p50/p90/p99) -> risk.
	Quantiles map[string]QuantileRule `json:"quantiles"`
}

type CounterRule struct {
	Weight float64 `json:"weight"`

	// Optional normalization: divide by this to interpret "per unit".
	// Example: per-minute normalization in the game server before sending is preferred,
	// but this allows mapping to remain stable across match lengths.
	Normalization float64 `json:"normalization"`
}

type QuantileRule struct {
	Weight float64 `json:"weight"`

	// Statistical baseline (mean/std) for a chosen percentile.
	// These are *not enforcement thresholds*; they are population parameters.
	// Keep these server-private and tuned per game/mode.
	Pctl string  `json:"pctl"` // "p50", "p90", "p95", "p99"
	Mean float64 `json:"mean"`
	Std  float64 `json:"std"`
}
