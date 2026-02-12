package risk

import "time"

// RiskState represents longitudinal risk for a player.
type RiskState struct {
	PlayerID   string
	TotalRisk  float64
	LastUpdate time.Time
}

// MatchRisk represents risk contribution from a single match.
type MatchRisk struct {
	PlayerID string
	Value    float64
	At       time.Time
}
