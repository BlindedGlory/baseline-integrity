package telemetry

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"time"

	baselineintegrityv1 "github.com/BlindedGlory/baseline-integrity/gen/go/baselineintegrity/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
)

const expectedTelemetrySchemaID = "baselineintegrity.telemetry.v1"

type Server struct {
	baselineintegrityv1.UnimplementedTelemetryServiceServer
	sinkDir string
}

func NewServer() (*Server, error) {
	dir := os.Getenv("BASELINEINTEGRITY_TELEMETRY_DIR")
	if dir == "" {
		// Keep dev data under the same local folder you already use for trust keys.
		dir = "./.baselineintegrity/telemetry"
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, fmt.Errorf("mkdir telemetry sink: %w", err)
	}
	return &Server{sinkDir: dir}, nil
}

func (s *Server) SubmitMatchAggregates(ctx context.Context, req *baselineintegrityv1.SubmitMatchAggregatesRequest) (*baselineintegrityv1.SubmitMatchAggregatesResponse, error) {
	_ = ctx

	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "missing request")
	}
	if req.MatchId == "" {
		return &baselineintegrityv1.SubmitMatchAggregatesResponse{Accepted: false, Reason: "missing_match_id"}, nil
	}
	if req.GameBuildId == "" {
		return &baselineintegrityv1.SubmitMatchAggregatesResponse{Accepted: false, Reason: "missing_game_build_id"}, nil
	}
	if len(req.Players) == 0 {
		return &baselineintegrityv1.SubmitMatchAggregatesResponse{Accepted: false, Reason: "missing_players"}, nil
	}

	// v1: enforce schema guardrail per player.
	for i, p := range req.Players {
		if p == nil {
			return &baselineintegrityv1.SubmitMatchAggregatesResponse{Accepted: false, Reason: "nil_player"}, nil
		}
		if p.Ref == nil {
			return &baselineintegrityv1.SubmitMatchAggregatesResponse{Accepted: false, Reason: fmt.Sprintf("player_%d_missing_ref", i)}, nil
		}
		if p.Ref.MatchId != "" && p.Ref.MatchId != req.MatchId {
			return &baselineintegrityv1.SubmitMatchAggregatesResponse{Accepted: false, Reason: "ref_match_id_mismatch"}, nil
		}
		if p.TelemetrySchemaId != expectedTelemetrySchemaID {
			return &baselineintegrityv1.SubmitMatchAggregatesResponse{Accepted: false, Reason: "schema_id_mismatch"}, nil
		}
		// Optional basic shape checks (kept light in v1)
		for _, h := range p.Histograms {
			if h == nil {
				continue
			}
			if h.BucketCount != 0 && int(h.BucketCount) != len(h.Buckets) {
				return &baselineintegrityv1.SubmitMatchAggregatesResponse{Accepted: false, Reason: "histogram_bucket_count_mismatch"}, nil
			}
		}
	}

	// Dev-safe disk sink: append JSON line per request.
	safeMatch := sanitize(req.MatchId)
	path := filepath.Join(s.sinkDir, "match_"+safeMatch+".jsonl")

	b, err := protojson.MarshalOptions{
		UseProtoNames:   true,
		EmitUnpopulated: false,
	}.Marshal(req)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "marshal telemetry: %v", err)
	}
	b = append(b, '\n')

	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "open telemetry sink: %v", err)
	}
	defer f.Close()

	// Tiny timestamp prefix makes tail/grep easier, still JSONL-friendly.
	prefix := []byte(time.Now().UTC().Format(time.RFC3339Nano) + " ")
	if _, err := f.Write(prefix); err != nil {
		return nil, status.Errorf(codes.Internal, "write telemetry sink: %v", err)
	}
	if _, err := f.Write(b); err != nil {
		return nil, status.Errorf(codes.Internal, "write telemetry sink: %v", err)
	}

	return &baselineintegrityv1.SubmitMatchAggregatesResponse{Accepted: true, Reason: "ok"}, nil
}

var nonSafe = regexp.MustCompile(`[^a-zA-Z0-9._-]+`)

func sanitize(s string) string {
	if s == "" {
		return "empty"
	}
	return nonSafe.ReplaceAllString(s, "_")
}
