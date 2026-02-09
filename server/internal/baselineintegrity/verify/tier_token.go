package verify

import (
	"bytes"
	"errors"
	"time"

	baselineintegrityv1 "github.com/BlindedGlory/baseline-integrity/gen/go/baselineintegrity/v1"
	bicrypto "github.com/BlindedGlory/baseline-integrity/server/internal/baselineintegrity/crypto"
	"google.golang.org/protobuf/proto"
)

var (
	ErrMissingToken     = errors.New("missing token")
	ErrMissingSignature = errors.New("missing signature")
	ErrUnknownKeyID     = errors.New("unknown key_id")
	ErrBadPayload       = errors.New("bad payload")
	ErrNotCanonical     = errors.New("payload not canonical")
	ErrWrapperMismatch  = errors.New("wrapper mismatch")
	ErrExpired          = errors.New("expired")
	ErrBadSignature     = errors.New("bad signature")
)

// PublicKeySet maps key_id -> ed25519 public key (32 bytes).
type PublicKeySet map[string][]byte

// VerifyTierTokenOffline verifies a TierToken without network calls.
// It enforces canonical payloads, wrapper matching, expiry, and signature validity.
func VerifyTierTokenOffline(
	tok *baselineintegrityv1.TierToken,
	keys PublicKeySet,
	now time.Time,
) error {
	if tok == nil {
		return ErrMissingToken
	}
	if tok.Signature == nil ||
		len(tok.Signature.Payload) == 0 ||
		len(tok.Signature.Signature) == 0 {
		return ErrMissingSignature
	}

	pub := keys[tok.Signature.KeyId]
	if len(pub) == 0 {
		return ErrUnknownKeyID
	}

	// Unmarshal the signed payload.
	var signed baselineintegrityv1.TierToken
	if err := proto.Unmarshal(tok.Signature.Payload, &signed); err != nil {
		return ErrBadPayload
	}

	// Payload must be signature-free.
	if signed.Signature != nil {
		return ErrNotCanonical
	}

	// Canonical encoding check.
	canonical, err := proto.Marshal(&signed)
	if err != nil || !bytes.Equal(canonical, tok.Signature.Payload) {
		return ErrNotCanonical
	}

	// Wrapper fields must exactly match signed payload fields.
	if !proto.Equal(tok.Ref, signed.Ref) ||
		tok.Tier != signed.Tier ||
		!bytes.Equal(tok.NonceHash, signed.NonceHash) ||
		!proto.Equal(tok.IssuedAt, signed.IssuedAt) ||
		!proto.Equal(tok.ExpiresAt, signed.ExpiresAt) {
		return ErrWrapperMismatch
	}

	// Expiry check.
	if tok.ExpiresAt == nil || now.After(tok.ExpiresAt.AsTime()) {
		return ErrExpired
	}

	// Signature verification.
	if !bicrypto.Verify(pub, tok.Signature.Payload, tok.Signature.Signature) {
		return ErrBadSignature
	}

	return nil
}
