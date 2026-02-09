package crypto

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

type diskKeyFileV1 struct {
	Version int    `json:"version"`
	KeyID   string `json:"key_id"`
	PrivB64 string `json:"priv_b64"`
}

type DiskKeyStore struct {
	Path string
}

// LoadOrCreateEd25519 loads an existing key file or creates a new one.
// Returns (keyID, priv, pub).
func (ks DiskKeyStore) LoadOrCreateEd25519() (string, ed25519.PrivateKey, ed25519.PublicKey, error) {
	dir := filepath.Dir(ks.Path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", nil, nil, fmt.Errorf("keystore mkdir: %w", err)
	}

	// Load if exists
	if b, err := os.ReadFile(ks.Path); err == nil {
		var f diskKeyFileV1
		if err := json.Unmarshal(b, &f); err != nil {
			return "", nil, nil, fmt.Errorf("keystore parse: %w", err)
		}
		if f.Version != 1 || f.KeyID == "" || f.PrivB64 == "" {
			return "", nil, nil, errors.New("keystore invalid fields")
		}
		raw, err := base64.RawStdEncoding.DecodeString(f.PrivB64)
		if err != nil {
			return "", nil, nil, fmt.Errorf("keystore priv decode: %w", err)
		}
		priv := ed25519.PrivateKey(raw)
		if len(priv) != ed25519.PrivateKeySize {
			return "", nil, nil, fmt.Errorf("wrong private key length: %d", len(priv))
		}
		pub := priv.Public().(ed25519.PublicKey)
		return f.KeyID, priv, pub, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", nil, nil, fmt.Errorf("keystore read: %w", err)
	}

	// Create new
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return "", nil, nil, fmt.Errorf("keystore generate: %w", err)
	}

	// KeyID: non-secret, stable per file; derived from pubkey prefix
	keyID := "dev-" + base64.RawURLEncoding.EncodeToString(pub[:12])

	f := diskKeyFileV1{
		Version: 1,
		KeyID:   keyID,
		PrivB64: base64.RawStdEncoding.EncodeToString([]byte(priv)),
	}
	out, err := json.MarshalIndent(&f, "", "  ")
	if err != nil {
		return "", nil, nil, fmt.Errorf("keystore marshal: %w", err)
	}
	out = append(out, '\n')

	// Atomic write, perms 0600
	tmp := ks.Path + ".tmp"
	if err := os.WriteFile(tmp, out, 0o600); err != nil {
		return "", nil, nil, fmt.Errorf("keystore write tmp: %w", err)
	}
	if err := os.Rename(tmp, ks.Path); err != nil {
		return "", nil, nil, fmt.Errorf("keystore rename: %w", err)
	}

	return keyID, priv, pub, nil
}
