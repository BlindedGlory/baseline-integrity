package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type config struct {
	rootDir      string
	retention    time.Duration
	dryRun       bool
	prunePlayers bool
}

type riskState struct {
	PlayerID   string    `json:"PlayerID"`
	TotalRisk  float64   `json:"TotalRisk"`
	LastUpdate time.Time `json:"LastUpdate"`
}

func main() {
	cfg := parseFlags()

	logger := log.New(os.Stdout, "bi-retention: ", log.Ldate|log.Ltime|log.LUTC|log.Lmsgprefix)

	if cfg.rootDir == "" {
		fmt.Fprintln(os.Stderr, "error: could not determine .baselineintegrity root.")
		fmt.Fprintln(os.Stderr, "Run from the repo root (where ./.baselineintegrity exists), or pass: --root /path/to/.baselineintegrity")
		os.Exit(2)
	}

	if cfg.retention <= 0 {
		fmt.Fprintln(os.Stderr, "error: --retention must be > 0 (e.g. 1440h for 60 days)")
		os.Exit(2)
	}

	now := time.Now().UTC()
	cutoff := now.Add(-cfg.retention)

	logger.Printf("starting (root=%s retention=%s cutoff=%s dry_run=%v prune_players=%v)",
		cfg.rootDir, cfg.retention, cutoff.Format(time.RFC3339Nano), cfg.dryRun, cfg.prunePlayers)

	var deletedFiles int
	var deletedDirs int

	// Telemetry: delete match_*.jsonl by embedded line timestamps (robust vs mtime changes).
	{
		dir := filepath.Join(cfg.rootDir, "telemetry")
		nf, nd, err := deleteOldTelemetryByLineTime(logger, dir, cutoff, cfg.dryRun)
		if err != nil {
			logger.Printf("telemetry prune error: %v", err)
		}
		deletedFiles += nf
		deletedDirs += nd
	}

	// Risk ledger: date-sharded directories risk/ledger/YYYY-MM-DD/ledger.jsonl
	{
		dir := filepath.Join(cfg.rootDir, "risk", "ledger")
		nf, nd, err := deleteOldLedgerDirs(logger, dir, cutoff, cfg.dryRun)
		if err != nil {
			logger.Printf("ledger prune error: %v", err)
		}
		deletedFiles += nf
		deletedDirs += nd
	}

	// Risk applied: delete *.ok markers by file ModTime
	{
		dir := filepath.Join(cfg.rootDir, "risk", "applied")
		nf, nd, err := deleteOldFilesByModTime(logger, dir, cutoff, cfg.dryRun, func(_ string, d fs.DirEntry) bool {
			return !d.IsDir() && strings.HasSuffix(d.Name(), ".ok")
		})
		if err != nil {
			logger.Printf("applied prune error: %v", err)
		}
		deletedFiles += nf
		deletedDirs += nd
	}

	// Outbox: prune done/failed by JSON CreatedAt (restore-safe). Never touch pending/processing.
	{
		for _, sub := range []string{"done", "failed"} {
			dir := filepath.Join(cfg.rootDir, "outbox", sub)

			nf, nd, err := deleteOldOutboxEventsByCreatedAt(logger, dir, cutoff, cfg.dryRun)
			if err != nil {
				logger.Printf("outbox/%s prune error: %v", sub, err)
			}
			deletedFiles += nf
			deletedDirs += nd

			// Also prune failed sidecars (*.err.txt) using the paired JSON's CreatedAt (skip if missing).
			if sub == "failed" {
				nf2, nd2, err2 := deleteOldOutboxFailedSidecarsByCreatedAt(logger, dir, cutoff, cfg.dryRun)
				if err2 != nil {
					logger.Printf("outbox/failed sidecar prune error: %v", err2)
				}
				deletedFiles += nf2
				deletedDirs += nd2
			}
		}
	}

	// Risk players: optional prune if LastUpdate older than cutoff
	if cfg.prunePlayers {
		dir := filepath.Join(cfg.rootDir, "risk", "players")
		nf, nd, err := deleteOldPlayersByLastUpdate(logger, dir, cutoff, cfg.dryRun)
		if err != nil {
			logger.Printf("players prune error: %v", err)
		}
		deletedFiles += nf
		deletedDirs += nd
	}

	logger.Printf("done (deleted_files=%d deleted_dirs=%d dry_run=%v)", deletedFiles, deletedDirs, cfg.dryRun)
}

func parseFlags() config {
	var cfg config
	var days int

	flag.StringVar(&cfg.rootDir, "root", "", "Path to .baselineintegrity root directory")
	flag.IntVar(&days, "days", 60, "Retention window in days")
	flag.BoolVar(&cfg.dryRun, "dry-run", true, "If true, only log what would be deleted")
	flag.BoolVar(&cfg.prunePlayers, "prune-players", false, "If true, delete risk/players/*.json where LastUpdate < cutoff")

	flag.Parse()

	cfg.retention = time.Duration(days) * 24 * time.Hour
	return cfg
}

func deleteOldFilesByModTime(
	logger *log.Logger,
	dir string,
	cutoff time.Time,
	dryRun bool,
	keep func(path string, d fs.DirEntry) bool,
) (deletedFiles int, deletedDirs int, err error) {

	// If directory doesn't exist, nothing to do.
	if _, statErr := os.Stat(dir); statErr != nil {
		return 0, 0, nil
	}

	walkErr := filepath.WalkDir(dir, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		if keep != nil && !keep(path, d) {
			return nil
		}

		info, err := d.Info()
		if err != nil {
			return err
		}
		if info.ModTime().UTC().Before(cutoff) {
			if dryRun {
				logger.Printf("DRY delete file: %s (mtime=%s)", path, info.ModTime().UTC().Format(time.RFC3339Nano))
				return nil
			}
			if err := os.Remove(path); err != nil {
				return err
			}
			logger.Printf("deleted file: %s", path)
			deletedFiles++
		}
		return nil
	})
	if walkErr != nil {
		return deletedFiles, deletedDirs, walkErr
	}

	// Best-effort: remove empty directories under dir (bottom-up).
	nd, err := removeEmptyDirs(logger, dir, dryRun)
	if err != nil {
		return deletedFiles, deletedDirs, err
	}
	deletedDirs += nd

	return deletedFiles, deletedDirs, nil
}

func deleteOldLedgerDirs(logger *log.Logger, ledgerRoot string, cutoff time.Time, dryRun bool) (deletedFiles int, deletedDirs int, err error) {
	if _, statErr := os.Stat(ledgerRoot); statErr != nil {
		return 0, 0, nil
	}

	entries, err := os.ReadDir(ledgerRoot)
	if err != nil {
		return 0, 0, err
	}

	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		day := e.Name()
		t, err := time.Parse("2006-01-02", day)
		if err != nil {
			continue // ignore non-date dirs
		}

		// Treat "YYYY-MM-DD" as midnight UTC of that day. If the whole day ends before cutoff, prune it.
		dayEnd := t.Add(24 * time.Hour).UTC()
		if !dayEnd.Before(cutoff) {
			continue
		}

		full := filepath.Join(ledgerRoot, day)
		if dryRun {
			logger.Printf("DRY delete dir: %s (day=%s)", full, day)
			continue
		}

		if err := os.RemoveAll(full); err != nil {
			return deletedFiles, deletedDirs, err
		}
		logger.Printf("deleted dir: %s", full)
		deletedDirs++
	}

	return deletedFiles, deletedDirs, nil
}

func deleteOldPlayersByLastUpdate(logger *log.Logger, playersDir string, cutoff time.Time, dryRun bool) (deletedFiles int, deletedDirs int, err error) {
	if _, statErr := os.Stat(playersDir); statErr != nil {
		return 0, 0, nil
	}

	entries, err := os.ReadDir(playersDir)
	if err != nil {
		return 0, 0, err
	}

	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		path := filepath.Join(playersDir, e.Name())

		b, err := os.ReadFile(path)
		if err != nil {
			return deletedFiles, deletedDirs, err
		}

		var s riskState
		if err := json.Unmarshal(b, &s); err != nil {
			// If it can't parse, don't delete it automatically.
			logger.Printf("skip unparseable player state: %s (err=%v)", path, err)
			continue
		}

		lu := s.LastUpdate.UTC()
		if lu.Before(cutoff) {
			if dryRun {
				logger.Printf("DRY delete player: %s (last_update=%s)", path, lu.Format(time.RFC3339Nano))
				continue
			}
			if err := os.Remove(path); err != nil {
				return deletedFiles, deletedDirs, err
			}
			logger.Printf("deleted player: %s", path)
			deletedFiles++
		}
	}

	nd, err := removeEmptyDirs(logger, playersDir, dryRun)
	if err != nil {
		return deletedFiles, deletedDirs, err
	}
	deletedDirs += nd

	return deletedFiles, deletedDirs, nil
}

func removeEmptyDirs(logger *log.Logger, root string, dryRun bool) (deleted int, err error) {
	// Walk bottom-up by collecting directories then processing in reverse.
	var dirs []string
	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			dirs = append(dirs, path)
		}
		return nil
	})
	if err != nil {
		return 0, err
	}

	// Reverse so we try to remove children before parents.
	for i := len(dirs) - 1; i >= 0; i-- {
		p := dirs[i]
		if p == root {
			continue
		}
		entries, err := os.ReadDir(p)
		if err != nil {
			return deleted, err
		}
		if len(entries) != 0 {
			continue
		}
		if dryRun {
			logger.Printf("DRY delete empty dir: %s", p)
			continue
		}
		if err := os.Remove(p); err != nil {
			return deleted, err
		}
		logger.Printf("deleted empty dir: %s", p)
		deleted++
	}

	return deleted, nil
}

func deleteOldTelemetryByLineTime(logger *log.Logger, telemetryDir string, cutoff time.Time, dryRun bool) (deletedFiles int, deletedDirs int, err error) {
	if _, statErr := os.Stat(telemetryDir); statErr != nil {
		return 0, 0, nil
	}

	entries, err := os.ReadDir(telemetryDir)
	if err != nil {
		return 0, 0, err
	}

	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasPrefix(name, "match_") || !strings.HasSuffix(name, ".jsonl") {
			continue
		}
		path := filepath.Join(telemetryDir, name)

		lastTs, ok, err := telemetryFileLastTimestamp(path)
		if err != nil {
			return deletedFiles, deletedDirs, err
		}
		if !ok {
			logger.Printf("skip telemetry (no timestamp): %s", path)
			continue
		}

		if lastTs.Before(cutoff) {
			if dryRun {
				logger.Printf("DRY delete telemetry: %s (last_ts=%s)", path, lastTs.UTC().Format(time.RFC3339Nano))
				continue
			}
			if err := os.Remove(path); err != nil {
				return deletedFiles, deletedDirs, err
			}
			logger.Printf("deleted telemetry: %s", path)
			deletedFiles++
		}
	}

	nd, err := removeEmptyDirs(logger, telemetryDir, dryRun)
	if err != nil {
		return deletedFiles, deletedDirs, err
	}
	deletedDirs += nd

	return deletedFiles, deletedDirs, nil
}

func telemetryFileLastTimestamp(path string) (time.Time, bool, error) {
	f, err := os.Open(path)
	if err != nil {
		return time.Time{}, false, err
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	var last time.Time
	found := false

	for sc.Scan() {
		line := sc.Bytes()
		space := bytes.IndexByte(line, ' ')
		if space <= 0 {
			continue
		}
		tsStr := string(line[:space])
		ts, err := time.Parse(time.RFC3339Nano, tsStr)
		if err != nil {
			continue
		}
		last = ts.UTC()
		found = true
	}
	if err := sc.Err(); err != nil {
		return time.Time{}, false, err
	}
	return last, found, nil
}

// outboxEvent defines the minimal JSON shape required for
// restore-safe retention of outbox events.
//
// Outbox pruning is based exclusively on the embedded
// "created_at" timestamp inside the JSON file.
//
// This prevents accidental deletion due to modified file mtimes
// (e.g., after backup/restore or filesystem copy operations).
//
// Files missing or containing an invalid "created_at" field
// are intentionally skipped (fail-safe).
type outboxEvent struct {
	ID        string    `json:"id"`
	Type      string    `json:"type"`
	MatchID   string    `json:"match_id"`
	CreatedAt time.Time `json:"created_at"`
}

func deleteOldOutboxEventsByCreatedAt(logger *log.Logger, dir string, cutoff time.Time, dryRun bool) (deletedFiles int, deletedDirs int, err error) {
	if _, statErr := os.Stat(dir); statErr != nil {
		return 0, 0, nil
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0, 0, err
	}

	var scanned int
	var skipped int

	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}

		scanned++

		path := filepath.Join(dir, e.Name())

		b, err := os.ReadFile(path)
		if err != nil {
			return deletedFiles, deletedDirs, err
		}

		var ev outboxEvent
		if err := json.Unmarshal(b, &ev); err != nil {
			logger.Printf("skip outbox (unparseable): %s (err=%v)", path, err)
			skipped++
			continue
		}

		if ev.CreatedAt.IsZero() {
			logger.Printf(
				"skip outbox (missing or invalid created_at; expected JSON field \"created_at\" as RFC3339/RFC3339Nano timestamp): %s",
				path,
			)
			skipped++
			continue
		}

		created := ev.CreatedAt.UTC()
		if created.Before(cutoff) {

			if dryRun {
				logger.Printf("DRY delete outbox: %s (created_at=%s cutoff=%s)", path,
					ev.CreatedAt.UTC().Format(time.RFC3339Nano),
					cutoff.UTC().Format(time.RFC3339Nano),
				)
				// IMPORTANT: don't increment deletedFiles in dry-run
				continue
			}

			if err := os.Remove(path); err != nil {
				return deletedFiles, deletedDirs, err
			}
			logger.Printf("deleted outbox: %s", path)
			deletedFiles++
		}

	}

	logger.Printf("outbox summary (%s): scanned=%d deleted=%d skipped=%d",
		filepath.Base(dir), scanned, deletedFiles, skipped)

	nd, err := removeEmptyDirs(logger, dir, dryRun)
	if err != nil {
		return deletedFiles, deletedDirs, err
	}
	deletedDirs += nd

	return deletedFiles, deletedDirs, nil
}

func deleteOldOutboxFailedSidecarsByCreatedAt(logger *log.Logger, dir string, cutoff time.Time, dryRun bool) (deletedFiles int, deletedDirs int, err error) {
	if _, statErr := os.Stat(dir); statErr != nil {
		return 0, 0, nil
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0, 0, err
	}

	var scanned int
	var skipped int

	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".err.txt") {
			continue
		}

		scanned++

		sidecarPath := filepath.Join(dir, e.Name())
		pairedJSONName := strings.TrimSuffix(e.Name(), ".err.txt")

		if !strings.HasSuffix(pairedJSONName, ".json") {
			logger.Printf("skip outbox sidecar (unexpected name): %s", sidecarPath)
			skipped++
			continue
		}

		pairedJSONPath := filepath.Join(dir, pairedJSONName)

		if _, statErr := os.Stat(pairedJSONPath); statErr != nil {
			logger.Printf("skip outbox sidecar (missing paired json): %s", sidecarPath)
			skipped++
			continue
		}

		b, err := os.ReadFile(pairedJSONPath)
		if err != nil {
			return deletedFiles, deletedDirs, err
		}

		var ev outboxEvent
		if err := json.Unmarshal(b, &ev); err != nil {
			logger.Printf("skip outbox sidecar (unparseable paired json): %s", sidecarPath)
			skipped++
			continue
		}

		if ev.CreatedAt.IsZero() {
			logger.Printf("skip outbox sidecar (missing created_at): %s", sidecarPath)
			skipped++
			continue
		}

		if ev.CreatedAt.UTC().Before(cutoff) {
			if dryRun {
				logger.Printf("DRY delete outbox sidecar: %s", sidecarPath)
			} else {
				if err := os.Remove(sidecarPath); err != nil {
					return deletedFiles, deletedDirs, err
				}
				logger.Printf("deleted outbox sidecar: %s", sidecarPath)
			}
			deletedFiles++
		}
	}

	logger.Printf("outbox sidecar summary (%s): scanned=%d deleted=%d skipped=%d",
		filepath.Base(dir), scanned, deletedFiles, skipped)

	nd, err := removeEmptyDirs(logger, dir, dryRun)
	if err != nil {
		return deletedFiles, deletedDirs, err
	}
	deletedDirs += nd

	return deletedFiles, deletedDirs, nil
}
