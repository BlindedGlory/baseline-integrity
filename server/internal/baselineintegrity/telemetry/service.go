package telemetry

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	baselineintegrityv1 "github.com/BlindedGlory/baseline-integrity/gen/go/baselineintegrity/v1"
	bicrypto "github.com/BlindedGlory/baseline-integrity/server/internal/baselineintegrity/crypto"
	bioutbox "github.com/BlindedGlory/baseline-integrity/server/internal/baselineintegrity/outbox"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

const expectedTelemetrySchemaID = "baselineintegrity.telemetry.v1"

type Server struct {
	baselineintegrityv1.UnimplementedTelemetryServiceServer

	sinkDir     string
	requireSig  bool
	allowedKeys map[string][]byte // key_id -> ed25519 pubkey (32 bytes)
}

func NewServer() (*Server, error) {
	dir := os.Getenv("BASELINEINTEGRITY_TELEMETRY_DIR")
	if dir == "" {
		// Dev default.
		dir = "./.baselineintegrity/telemetry"
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, fmt.Errorf("mkdir telemetry sink: %w", err)
	}

	requireSig := os.Getenv("BASELINEINTEGRITY_REQUIRE_TELEMETRY_SERVER_SIG") == "1"
	allowed, err := parseAllowedServerKeys(os.Getenv("BASELINEINTEGRITY_TELEMETRY_SERVER_PUBKEYS"))
	if err != nil {
		return nil, fmt.Errorf("parse telemetry server pubkeys: %w", err)
	}

	return &Server{
		sinkDir:     dir,
		requireSig:  requireSig,
		allowedKeys: allowed,
	}, nil
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

	// Optional signature enforcement (deployment decides auth).
	if s.requireSig {
		if req.ServerSignature == nil || len(req.ServerSignature.Payload) == 0 || len(req.ServerSignature.Signature) == 0 {
			return &baselineintegrityv1.SubmitMatchAggregatesResponse{Accepted: false, Reason: "missing_server_signature"}, nil
		}
		pub := s.allowedKeys[req.ServerSignature.KeyId]
		if len(pub) == 0 {
			return &baselineintegrityv1.SubmitMatchAggregatesResponse{Accepted: false, Reason: "unknown_server_key_id"}, nil
		}

		// Canonical payload = request without server_signature.
		unsigned := proto.Clone(req).(*baselineintegrityv1.SubmitMatchAggregatesRequest)
		unsigned.ServerSignature = nil

		canonical, err := proto.Marshal(unsigned)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "marshal unsigned telemetry: %v", err)
		}
		if !bytes.Equal(canonical, req.ServerSignature.Payload) {
			return &baselineintegrityv1.SubmitMatchAggregatesResponse{Accepted: false, Reason: "server_signature_payload_mismatch"}, nil
		}
		if !bicrypto.Verify(pub, req.ServerSignature.Payload, req.ServerSignature.Signature) {
			return &baselineintegrityv1.SubmitMatchAggregatesResponse{Accepted: false, Reason: "bad_server_signature"}, nil
		}
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
	safeMatch := sanitize(req.GetMatchId())
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

	prefix := []byte(time.Now().UTC().Format(time.RFC3339Nano) + " ")
	if _, err := f.Write(prefix); err != nil {
		return nil, status.Errorf(codes.Internal, "write telemetry sink: %v", err)
	}
	if _, err := f.Write(b); err != nil {
		return nil, status.Errorf(codes.Internal, "write telemetry sink: %v", err)
	}

	// Match-finalized push (filesystem outbox).
	// Today: SubmitMatchAggregates is typically called once at match end.
	// Later: when chunked telemetry exists, set BASELINEINTEGRITY_OUTBOX_ON_FINALIZE_ONLY=1
	// and enqueue only when the request represents a true finalize moment.
	outboxDir := os.Getenv("BASELINEINTEGRITY_OUTBOX_DIR")
	if outboxDir == "" {
		outboxDir = "./.baselineintegrity/outbox"
	}

	enqueueFinalizeOnly := os.Getenv("BASELINEINTEGRITY_OUTBOX_ON_FINALIZE_ONLY") == "1"
	isFinalized := !enqueueFinalizeOnly // default: treat every submit as final for now

	// Placeholder finalize signal until the proto grows a real "finalized" flag/RPC.
	// When you add a proto field, set: isFinalized = req.GetFinalized()
	if isFinalized {
		ob := bioutbox.FSOutbox{Dir: outboxDir}

	instance := os.Getenv("BASELINEINTEGRITY_SERVER_INSTANCE_ID")
	if instance == "" {
	instance = "dev"
	}

	ev := bioutbox.Event{
	    ID:        bioutbox.NewEventID(instance, req.GetMatchId()),
	    Type:      bioutbox.EventMatchFinalized,
	    MatchID:   req.GetMatchId(),
	    CreatedAt: time.Now().UTC(), // finalize moment (grace window uses this)
	}

		if err := ob.Enqueue(ev); err != nil {
			// Do not fail ingestion; telemetry is already persisted.
			log.Printf("outbox enqueue failed: %v", err)
		}
	}

	return &baselineintegrityv1.SubmitMatchAggregatesResponse{Accepted: true, Reason: "ok"}, nil
}
func parseAllowedServerKeys(s string) (map[string][]byte, error) {
	m := map[string][]byte{}
	s = strings.TrimSpace(s)
	if s == "" {
		return m, nil
	}
	parts := strings.Split(s, ",")
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		kv := strings.SplitN(p, ":", 2)
		if len(kv) != 2 {
			return nil, fmt.Errorf("bad key entry %q (want keyId:base64pub)", p)
		}
		keyID := strings.TrimSpace(kv[0])
		b64 := strings.TrimSpace(kv[1])

		pub, err := base64.StdEncoding.DecodeString(b64)
		if err != nil {
			return nil, fmt.Errorf("decode pubkey for %q: %w", keyID, err)
		}
		if len(pub) != 32 {
			return nil, fmt.Errorf("pubkey for %q wrong length: got %d want 32", keyID, len(pub))
		}
		m[keyID] = pub
	}
	return m, nil
}

var nonSafe = regexp.MustCompile(`[^a-zA-Z0-9._-]+`)

func sanitize(s string) string {
	if s == "" {
		return "empty"
	}
	return nonSafe.ReplaceAllString(s, "_")
}
