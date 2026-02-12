package store

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type MatchAggregatesLine map[string]any

func LoadMatchAggregatesJSONL(telemetryDir string, matchID string) ([]MatchAggregatesLine, error) {
	if telemetryDir == "" {
		return nil, fmt.Errorf("telemetryDir is required")
	}
	if matchID == "" {
		return nil, fmt.Errorf("matchID is required")
	}

	path := filepath.Join(telemetryDir, fmt.Sprintf("match_%s.jsonl", matchID))
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var out []MatchAggregatesLine

	s := bufio.NewScanner(f)
	// Allow larger lines than the default 64K.
	buf := make([]byte, 0, 1024*1024)
	s.Buffer(buf, 8*1024*1024)

	for s.Scan() {
		line := s.Bytes()
		if len(line) == 0 {
			continue
		}

		var m MatchAggregatesLine
		if err := json.Unmarshal(line, &m); err != nil {
			return nil, fmt.Errorf("invalid jsonl line: %w", err)
		}
		out = append(out, m)
	}
	if err := s.Err(); err != nil {
		return nil, err
	}

	return out, nil
}
