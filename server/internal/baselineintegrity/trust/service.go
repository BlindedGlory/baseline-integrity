package trust

import (
	"bytes"
	"context"
	"errors"
	"os"
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
	keyPath := os.Getenv("BASELINEINTEGRITY_SIGNING_KEY_PATH")
	if keyPath == "" {
		keyPath = "./.baselineintegrity/dev_signing_key.json"
	}

	signer, err := bicrypto.NewDiskSigner(keyPath)
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
func (s *Server) GetPublicKeys(ctx context.Context, req *baselineintegrityv1.GetPublicKeysRequest) (*baselineintegrityv1.GetPublicKeysResponse, error) {
	_ = ctx
	_ = req // purpose is informational in v1

	// Cache hint: game servers can cache keys; rotation comes later.
	cacheUntil := timestamppb.New(time.Now().Add(24 * time.Hour))

	pk := &baselineintegrityv1.PublicKey{
		KeyId:   s.signer.KeyID,
		Ed25519: s.signer.Pub, // 32 bytes
	}

	return &baselineintegrityv1.GetPublicKeysResponse{
		Keys:       []*baselineintegrityv1.PublicKey{pk},
		CacheUntil: cacheUntil,
	}, nil
}
func (s *Server) IntrospectTierToken(ctx context.Context, req *baselineintegrityv1.IntrospectTierTokenRequest) (*baselineintegrityv1.IntrospectTierTokenResponse, error) {
	_ = ctx

	if req == nil || req.Token == nil {
		return &baselineintegrityv1.IntrospectTierTokenResponse{
			Valid:  false,
			Reason: "missing_token",
		}, nil
	}

	tok := req.Token
	if tok.Signature == nil {
		return &baselineintegrityv1.IntrospectTierTokenResponse{
			Valid:  false,
			Reason: "missing_signature",
		}, nil
	}
	if len(tok.Signature.Payload) == 0 || len(tok.Signature.Signature) == 0 {
		return &baselineintegrityv1.IntrospectTierTokenResponse{
			Valid:  false,
			Reason: "missing_signature_bytes",
		}, nil
	}

	// v1: single active key. Rotation later.
	if tok.Signature.KeyId != s.signer.KeyID {
		return &baselineintegrityv1.IntrospectTierTokenResponse{
			Valid:  false,
			Reason: "unknown_key_id",
		}, nil
	}

	// Decode the signed payload (canonical unsigned TierToken).
	var signedTok baselineintegrityv1.TierToken
	if err := proto.Unmarshal(tok.Signature.Payload, &signedTok); err != nil {
		return &baselineintegrityv1.IntrospectTierTokenResponse{
			Valid:  false,
			Reason: "bad_payload",
		}, nil
	}

	// Canonical rule: payload must be signature-free.
	if signedTok.Signature != nil {
		return &baselineintegrityv1.IntrospectTierTokenResponse{
			Valid:  false,
			Reason: "payload_not_canonical",
		}, nil
	}

	// Canonical encoding check: re-marshal must match exactly what was signed.
	canonical, err := proto.Marshal(&signedTok)
	if err != nil || !bytes.Equal(canonical, tok.Signature.Payload) {
		return &baselineintegrityv1.IntrospectTierTokenResponse{
			Valid:  false,
			Reason: "payload_not_canonical",
		}, nil
	}

	// Cross-check wrapper fields match the signed payload fields.
	if !proto.Equal(tok.Ref, signedTok.Ref) {
		return &baselineintegrityv1.IntrospectTierTokenResponse{
			Valid:  false,
			Reason: "ref_mismatch",
		}, nil
	}
	if tok.Tier != signedTok.Tier {
		return &baselineintegrityv1.IntrospectTierTokenResponse{
			Valid:  false,
			Reason: "tier_mismatch",
		}, nil
	}
	if !bytes.Equal(tok.NonceHash, signedTok.NonceHash) {
		return &baselineintegrityv1.IntrospectTierTokenResponse{
			Valid:  false,
			Reason: "nonce_hash_mismatch",
		}, nil
	}
	if !proto.Equal(tok.IssuedAt, signedTok.IssuedAt) {
		return &baselineintegrityv1.IntrospectTierTokenResponse{
			Valid:  false,
			Reason: "issued_at_mismatch",
		}, nil
	}
	if !proto.Equal(tok.ExpiresAt, signedTok.ExpiresAt) {
		return &baselineintegrityv1.IntrospectTierTokenResponse{
			Valid:  false,
			Reason: "expires_at_mismatch",
		}, nil
	}

	// Expiry check (server authoritative time), after we know wrapper/payload align.
	if tok.ExpiresAt == nil {
		return &baselineintegrityv1.IntrospectTierTokenResponse{
			Valid:  false,
			Reason: "missing_expires_at",
		}, nil
	}
	if time.Now().After(tok.ExpiresAt.AsTime()) {
		return &baselineintegrityv1.IntrospectTierTokenResponse{
			Valid:     false,
			Reason:    "expired",
			Tier:      tok.Tier,
			Ref:       tok.Ref,
			ExpiresAt: tok.ExpiresAt,
		}, nil
	}

	// Verify signature over SignedEnvelope.payload
	if !bicrypto.Verify(s.signer.Pub, tok.Signature.Payload, tok.Signature.Signature) {
		return &baselineintegrityv1.IntrospectTierTokenResponse{
			Valid:  false,
			Reason: "bad_signature",
		}, nil
	}

	return &baselineintegrityv1.IntrospectTierTokenResponse{
		Valid:     true,
		Reason:    "ok",
		Tier:      tok.Tier,
		Ref:       tok.Ref,
		ExpiresAt: tok.ExpiresAt,
	}, nil
}
