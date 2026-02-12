package risk

// Store abstracts persistence of risk state.
type Store interface {
	Load(playerID string) (RiskState, error)
	Save(state RiskState) error
}
