package store

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	baselineintegrityv1 "github.com/BlindedGlory/baseline-integrity/gen/go/baselineintegrity/v1"
	"google.golang.org/protobuf/encoding/protojson"
)

type MatchRequestLine struct {
	At  time.Time
	Req *baselineintegrityv1.SubmitMatchAggregatesRequest
}

func LoadMatchRequestsJSONL(telemetryDir string, matchID string) ([]MatchRequestLine, error) {
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

	var out []MatchRequestLine

	s := bufio.NewScanner(f)
	buf := make([]byte, 0, 1024*1024)
	s.Buffer(buf, 8*1024*1024)

	unmarshal := protojson.UnmarshalOptions{
		DiscardUnknown: false,
	}

	for s.Scan() {
		line := strings.TrimSpace(s.Text())
		if line == "" {
			continue
		}

		// Format: "<RFC3339Nano> <JSON>"
		i := strings.IndexByte(line, ' ')
		if i <= 0 || i >= len(line)-1 {
			return nil, fmt.Errorf("invalid telemetry line (missing timestamp prefix)")
		}

		ts := line[:i]
		j := line[i+1:]

		at, err := time.Parse(time.RFC3339Nano, ts)
		if err != nil {
			return nil, fmt.Errorf("invalid telemetry timestamp %q: %w", ts, err)
		}

		req := &baselineintegrityv1.SubmitMatchAggregatesRequest{}
		if err := unmarshal.Unmarshal([]byte(j), req); err != nil {
			return nil, fmt.Errorf("invalid telemetry json: %w", err)
		}

		out = append(out, MatchRequestLine{At: at, Req: req})
	}
	if err := s.Err(); err != nil {
		return nil, err
	}

	return out, nil
}
