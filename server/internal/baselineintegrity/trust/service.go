package trust

import (
	"context"
	"errors"
	"time"

	baselineintegrityv1 "github.com/BlindedGlory/baseline-integrity/gen/go/baselineintegrity/v1"
	bicrypto "github.com/BlindedGlory/baseline-integrity/server/internal/baselineintegrity/crypto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Server implements baselineintegrity.v1.TrustService.
type Server struct {
	baselineintegrityv1.UnimplementedTrustServiceServer
	signer *bicrypto.Signer
}

// NewServer creates a TrustService server with an in-memory signing key (v1).
// Later we will replace this with persisted + rotated keys.
func NewServer() (*Server, error) {
	signer, err := bicrypto.NewEphemeralSigner("dev-ephemeral")
	if err != nil {
		return nil, err
	}
	return &Server{signer: signer}, nil
}

func (s *Server) StartSession(ctx context.Context, req *baselineintegrityv1.StartSessionRequest) (*baselineintegrityv1.StartSessionResponse, error) {
	if s == nil || s.signer == nil {
		return nil, status.Error(codes.Internal, "trust server not initialized")
	}
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is nil")
	}

	ref := req.GetRef()
	if ref.GetSessionId() == "" || ref.GetMatchId() == "" {
		return nil, status.Error(codes.InvalidArgument, "ref.session_id and ref.match_id are required")
	}

	now := time.Now()
	exp := now.Add(10 * time.Minute)

	nonceBytes, err := bicrypto.NewNonce32()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "nonce generation failed: %v", err)
	}
	nonceHash := bicrypto.SHA256(nonceBytes)

	// Public, non-sensitive policy (no thresholds, no weights).
	policy := &baselineintegrityv1.Policy{
		MaxTierSupported:          baselineintegrityv1.TrustTier_TRUST_TIER_VERIFIED,
		TelemetrySchemaId:         "baselineintegrity.telemetry.v1",
		VerifiedRequiresCompanion: true,
	}

	// Build the OPEN tier token and sign a canonical, signature-free copy.
	openUnsigned := &baselineintegrityv1.TierToken{
		Ref:       ref,
		Tier:      baselineintegrityv1.TrustTier_TRUST_TIER_OPEN,
		NonceHash: nonceHash,
		IssuedAt:  timestamppb.New(now),
		ExpiresAt: timestamppb.New(exp),
	}

	payload, err := proto.Marshal(openUnsigned)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "token marshal failed: %v", err)
	}

	sig, err := s.signer.Sign(payload)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "token sign failed: %v", err)
	}

	if len(sig) == 0 {
		return nil, status.Error(codes.Internal, "token sign returned empty signature")
	}

	openToken := proto.Clone(openUnsigned).(*baselineintegrityv1.TierToken)
	openToken.Signature = &baselineintegrityv1.SignedEnvelope{
		KeyId:     s.signer.KeyID,
		Payload:   payload,
		Signature: sig,
		SignedAt:  timestamppb.New(now),
	}

	// v1: Verified availability is policy/platform-dependent.
	verifiedAvailable := req.GetRequestedTier() == baselineintegrityv1.TrustTier_TRUST_TIER_VERIFIED ||
		req.GetRequestedTier() == baselineintegrityv1.TrustTier_TRUST_TIER_OPEN

	if !verifiedAvailable {
		// Defensive: only OPEN/VERIFIED are valid requests in v1.
		return nil, status.Error(codes.InvalidArgument, "requested_tier must be OPEN or VERIFIED")
	}

	resp := &baselineintegrityv1.StartSessionResponse{
		Ref:               ref,
		Nonce:             &baselineintegrityv1.Nonce{Value: nonceBytes},
		Policy:            policy,
		OpenTierToken:     openToken,
		VerifiedAvailable: true,
		ExpiresAt:         timestamppb.New(exp),
	}

	// Sanity: ensure token has required fields.
	if resp.OpenTierToken.GetSignature() == nil {
		return nil, status.Error(codes.Internal, "missing token signature")
	}
	if len(resp.GetNonce().GetValue()) != 32 {
		return nil, errors.New("nonce length is not 32 bytes")
	}

	return resp, nil
}
