package risk

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

type FileStore struct {
	Dir string
}

func (fs *FileStore) path(playerID string) string {
	return filepath.Join(fs.Dir, playerID+".json")
}

func (fs *FileStore) Load(playerID string) (RiskState, error) {
	p := fs.path(playerID)

	b, err := os.ReadFile(p)
	if err != nil {
		return RiskState{
			PlayerID:   playerID,
			TotalRisk:  0,
			LastUpdate: time.Now(),
		}, nil
	}

	var s RiskState
	if err := json.Unmarshal(b, &s); err != nil {
		return s, err
	}
	return s, nil
}

func (fs *FileStore) Save(state RiskState) error {
	b, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(fs.path(state.PlayerID), b, 0600)
}
