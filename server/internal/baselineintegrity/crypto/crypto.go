package crypto

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
)

func NewNonce32() ([]byte, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return nil, err
	}
	return b, nil
}

func SHA256(b []byte) []byte {
	h := sha256.Sum256(b)
	return h[:]
}

func HexSHA256(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

type Signer struct {
	KeyID string
	Priv  ed25519.PrivateKey
	Pub   ed25519.PublicKey
}

func NewEphemeralSigner(keyID string) (*Signer, error) {
	// v1: ephemeral in-memory key. We'll replace with persisted/rotating keys later.
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}
	return &Signer{KeyID: keyID, Priv: priv, Pub: pub}, nil
}

// NewDiskSigner loads or creates a persisted Ed25519 signing key at keyPath.
func NewDiskSigner(keyPath string) (*Signer, error) {
	keyID, priv, pub, err := (DiskKeyStore{Path: keyPath}).LoadOrCreateEd25519()
	if err != nil {
		return nil, fmt.Errorf("load/create signing key: %w", err)
	}
	return &Signer{KeyID: keyID, Priv: priv, Pub: pub}, nil
}

func (s *Signer) Sign(payload []byte) ([]byte, error) {
	if s == nil || len(s.Priv) == 0 {
		return nil, errors.New("signer not initialized")
	}
	return ed25519.Sign(s.Priv, payload), nil
}

func Verify(pub ed25519.PublicKey, payload, sig []byte) bool {
	return ed25519.Verify(pub, payload, sig)
}
