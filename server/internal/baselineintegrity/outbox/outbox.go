package outbox

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

var ErrNoPending = errors.New("no pending events")

type EventType string

const (
	EventMatchFinalized EventType = "match_finalized"
)

type Event struct {
	ID        string    `json:"id"`
	Type      EventType `json:"type"`
	MatchID   string    `json:"match_id"`
	CreatedAt time.Time `json:"created_at"`
}

type FSOutbox struct {
	Dir string
}

func NewEventID(instance, matchID string) string {
	return fmt.Sprintf("%d_%s_%s",
		time.Now().UTC().UnixNano(),
		instance,
		matchID,
	)
}

func (o *FSOutbox) Ensure() error {
	for _, d := range []string{"pending", "processing", "done", "failed"} {
		if err := os.MkdirAll(filepath.Join(o.Dir, d), 0700); err != nil {
			return err
		}
	}
	return nil
}

func (o *FSOutbox) Enqueue(ev Event) error {
	if err := o.Ensure(); err != nil {
		return err
	}

	b, err := json.Marshal(ev)
	if err != nil {
		return err
	}

	tmp := filepath.Join(o.Dir, "pending", ev.ID+".tmp")
	dst := filepath.Join(o.Dir, "pending", ev.ID+".json")

	if err := os.WriteFile(tmp, b, 0600); err != nil {
		return err
	}

	if err := os.Rename(tmp, dst); err != nil {
		_ = os.Remove(tmp)

		// Already queued: treat as success.
		if errors.Is(err, os.ErrExist) {
			return nil
		}
		// Some filesystems may not return os.ErrExist for rename collision; fallback.
		if _, statErr := os.Stat(dst); statErr == nil {
			return nil
		}
		return err
	}

	return nil
}

// ClaimOne atomically moves a pending event into processing and returns it + its processing path.
func (o *FSOutbox) ClaimOne() (Event, string, error) {
	if err := o.Ensure(); err != nil {
		return Event{}, "", err
	}

	pendingDir := filepath.Join(o.Dir, "pending")
	entries, err := os.ReadDir(pendingDir)
	if err != nil {
		return Event{}, "", err
	}

	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if filepath.Ext(name) != ".json" {
			continue
		}

		src := filepath.Join(pendingDir, name)
		dst := filepath.Join(o.Dir, "processing", name)

		// atomic claim: if rename succeeds, we own it
		if err := os.Rename(src, dst); err != nil {
			continue
		}

		b, err := os.ReadFile(dst)
		if err != nil {
			_ = os.Rename(dst, filepath.Join(o.Dir, "failed", name))
			return Event{}, "", err
		}

		var ev Event
		if err := json.Unmarshal(b, &ev); err != nil {
			_ = os.Rename(dst, filepath.Join(o.Dir, "failed", name))
			return Event{}, "", err
		}

		return ev, dst, nil
	}

	return Event{}, "", ErrNoPending
}

func (o *FSOutbox) MarkDone(processingPath string) error {
	name := filepath.Base(processingPath)
	return os.Rename(processingPath, filepath.Join(o.Dir, "done", name))
}

func (o *FSOutbox) MarkFailed(processingPath string, why error) error {
	name := filepath.Base(processingPath)
	// keep original file; optionally write a sidecar error file
	_ = os.WriteFile(filepath.Join(o.Dir, "failed", name+".err.txt"), []byte(fmt.Sprintf("%v\n", why)), 0600)
	return os.Rename(processingPath, filepath.Join(o.Dir, "failed", name))
}
func (o *FSOutbox) Requeue(processingPath string) error {
	name := filepath.Base(processingPath)
	return os.Rename(processingPath, filepath.Join(o.Dir, "pending", name))
}
