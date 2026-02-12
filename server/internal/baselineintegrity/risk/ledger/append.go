package ledger

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

type MatchLine struct {
	MatchID string             `json:"match_id"`
	At      time.Time          `json:"at"`
	Players map[string]float64 `json:"players"`
}

func AppendMatchLine(riskRoot string, line MatchLine) error {
	day := line.At.UTC().Format("2006-01-02")
	dir := filepath.Join(riskRoot, "ledger", day)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}

	path := filepath.Join(dir, "ledger.jsonl")
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()

	b, err := json.Marshal(line)
	if err != nil {
		return err
	}
	b = append(b, '\n')

	_, err = f.Write(b)
	return err
}
