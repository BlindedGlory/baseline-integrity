package risk

import (
	"os"
	"path/filepath"
)

type AppliedStore struct {
	Dir string
}

func (as *AppliedStore) path(matchID string) string {
	return filepath.Join(as.Dir, matchID+".ok")
}

func (as *AppliedStore) IsApplied(matchID string) (bool, error) {
	_, err := os.Stat(as.path(matchID))
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func (as *AppliedStore) MarkApplied(matchID string) error {
	if err := os.MkdirAll(as.Dir, 0o700); err != nil {
		return err
	}
	// Atomic-ish: create/truncate marker file.
	return os.WriteFile(as.path(matchID), []byte("ok\n"), 0o600)
}
