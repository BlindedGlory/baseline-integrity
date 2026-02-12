package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	bioutbox "github.com/BlindedGlory/baseline-integrity/server/internal/baselineintegrity/outbox"
	"github.com/BlindedGlory/baseline-integrity/server/internal/baselineintegrity/risk"
	"github.com/BlindedGlory/baseline-integrity/server/internal/baselineintegrity/risk/ledger"
	"github.com/BlindedGlory/baseline-integrity/server/internal/baselineintegrity/telemetry/store"
	"github.com/BlindedGlory/baseline-integrity/server/internal/baselineintegrity/worker"
)

type config struct {
	rootDir string

	outboxDir         string
	telemetryDir      string
	riskDir           string
	mappingConfigPath string

	decayFactor float64
	riskCap     float64

	finalizeGrace time.Duration

	pollInterval time.Duration
	once         bool
}

func main() {
	cfg := parseFlags()

	logger := log.New(os.Stdout, "risk-worker: ", log.Ldate|log.Ltime|log.LUTC|log.Lmsgprefix)

	if cfg.outboxDir == "" || cfg.telemetryDir == "" || cfg.riskDir == "" || cfg.mappingConfigPath == "" {
		fmt.Fprintln(os.Stderr, "error: missing required paths.")
		fmt.Fprintln(os.Stderr, "Provide --root <.baselineintegrity> OR set:")
		fmt.Fprintln(os.Stderr, "  --outbox-dir, --telemetry-dir, --risk-dir, --mapping-config")
		os.Exit(2)
	}

	if cfg.decayFactor <= 0 || cfg.decayFactor > 1 {
		fmt.Fprintln(os.Stderr, "error: --decay-factor must be in (0,1]")
		os.Exit(2)
	}
	if cfg.riskCap <= 0 {
		fmt.Fprintln(os.Stderr, "error: --risk-cap must be > 0")
		os.Exit(2)
	}
	if cfg.finalizeGrace < 0 {
		fmt.Fprintln(os.Stderr, "error: --finalize-grace must be >= 0")
		os.Exit(2)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	logger.Printf("starting (once=%v poll=%s grace=%s outbox=%s telemetry=%s risk=%s mapping=%s)",
		cfg.once, cfg.pollInterval, cfg.finalizeGrace, cfg.outboxDir, cfg.telemetryDir, cfg.riskDir, cfg.mappingConfigPath)

	if err := run(ctx, logger, cfg); err != nil {
		logger.Printf("fatal: %v", err)
		os.Exit(1)
	}

	logger.Printf("exiting cleanly")
}

func parseFlags() config {
	var cfg config

	flag.StringVar(&cfg.rootDir, "root", "", "Path to .baselineintegrity root directory (optional; derives outbox/telemetry/risk + default mapping)")

	flag.StringVar(&cfg.outboxDir, "outbox-dir", "", "Path to outbox root directory")
	flag.StringVar(&cfg.telemetryDir, "telemetry-dir", "", "Path to telemetry storage directory")
	flag.StringVar(&cfg.riskDir, "risk-dir", "", "Path to risk storage directory")
	flag.StringVar(&cfg.mappingConfigPath, "mapping-config", "", "Path to risk mapping config JSON")

	flag.Float64Var(&cfg.decayFactor, "decay-factor", 0.999, "Risk decay factor per hour (0-1]")
	flag.Float64Var(&cfg.riskCap, "risk-cap", 10.0, "Maximum longitudinal risk cap")

	flag.DurationVar(&cfg.finalizeGrace, "finalize-grace", 0, "Grace window after finalize CreatedAt before processing (e.g. 30s)")

	flag.DurationVar(&cfg.pollInterval, "poll-interval", 1*time.Second, "Poll interval")
	flag.BoolVar(&cfg.once, "once", false, "Process available events once then exit")

	flag.Parse()

	cfg.rootDir = defaultRootIfPresent(cfg.rootDir)

	// If --root (or auto-root) is present, derive defaults unless explicitly overridden.
	if cfg.rootDir != "" {
		if cfg.outboxDir == "" {
			cfg.outboxDir = filepath.Join(cfg.rootDir, "outbox")
		}
		if cfg.telemetryDir == "" {
			cfg.telemetryDir = filepath.Join(cfg.rootDir, "telemetry")
		}
		if cfg.riskDir == "" {
			cfg.riskDir = filepath.Join(cfg.rootDir, "risk")
		}
		if cfg.mappingConfigPath == "" {
			dev := filepath.Join(cfg.rootDir, "risk_mapping.dev.json")
			example := filepath.Join(cfg.rootDir, "risk_mapping.example.json")

			if _, err := os.Stat(dev); err == nil {
				cfg.mappingConfigPath = dev
			} else {
				cfg.mappingConfigPath = example
			}
		}
	}

	return cfg
}
func defaultRootIfPresent(root string) string {
	if root != "" {
		return root
	}

	cwd, err := os.Getwd()
	if err != nil {
		return ""
	}

	candidate := filepath.Join(cwd, ".baselineintegrity")
	if _, err := os.Stat(candidate); err == nil {
		return candidate
	}

	return ""
}

func run(ctx context.Context, logger *log.Logger, cfg config) error {
	w, err := worker.New(logger, worker.Config{
		OutboxDir:    cfg.outboxDir,
		PollInterval: cfg.pollInterval,
		Once:         cfg.once,
	})
	if err != nil {
		return err
	}

	mapCfg, err := risk.LoadMappingConfigFromFile(cfg.mappingConfigPath)
	if err != nil {
		return err
	}
	if mapCfg.ExpectedSchemaID == "" {
		mapCfg.ExpectedSchemaID = "baselineintegrity.telemetry.v1"
	}

	rs := &risk.FileStore{Dir: filepath.Join(cfg.riskDir, "players")}
	if err := os.MkdirAll(rs.Dir, 0o700); err != nil {
		return err
	}

	applied := &risk.AppliedStore{Dir: filepath.Join(cfg.riskDir, "applied")}

	scoreCfg := risk.Config{
		DecayFactor: cfg.decayFactor,
		RiskCap:     cfg.riskCap,
	}

	return w.Run(ctx, func(ctx context.Context, claimed worker.Claimed) error {
		ev := claimed.Event

		if ev.Type != bioutbox.EventMatchFinalized {
			logger.Printf(
				"SKIP event=%s match=%s reason=unsupported_type id=%s",
				ev.Type,
				ev.MatchID,
				ev.ID,
			)

			return nil
		}

		// Grace window: requeue until finalize timestamp + grace passes.
		if cfg.finalizeGrace > 0 {
			retryAt := ev.CreatedAt.Add(cfg.finalizeGrace)
			now := time.Now().UTC()
			if now.Before(retryAt) {
				logger.Printf(
					"RETRY event=match_finalized match=%s reason=grace_window now=%s retry_at=%s",
					ev.MatchID,
					now.Format(time.RFC3339Nano),
					retryAt.UTC().Format(time.RFC3339Nano),
				)
				return worker.ErrRetryLater
			}
		}
		already, err := applied.IsApplied(ev.MatchID)

		if err != nil {
			return err
		}
		if already {
			logger.Printf(
				"SKIP event=match_finalized match=%s reason=already_applied",
				ev.MatchID,
			)
			return nil
		}

		logger.Printf("processing match_finalized: id=%s match=%s created_at=%s",
			ev.ID, ev.MatchID, ev.CreatedAt.UTC().Format(time.RFC3339Nano),
		)

		lines, err := store.LoadMatchRequestsJSONL(cfg.telemetryDir, ev.MatchID)
		if err != nil {
			telePath := filepath.Join(cfg.telemetryDir, fmt.Sprintf("match_%s.jsonl", ev.MatchID))
			return fmt.Errorf(
				"FAIL event=match_finalized match=%s reason=telemetry_missing expected=%s: %w",
				ev.MatchID,
				telePath,
				err,
			)

		}
		if len(lines) == 0 {
			logger.Printf("loaded telemetry match=%s lines=0", ev.MatchID)
			return nil
		}

		players := make(map[string]float64)
		var lastAt time.Time
		scored := 0

		for _, line := range lines {
			req := line.Req
			at := line.At
			lastAt = at

			for _, p := range req.GetPlayers() {
				mr, err := risk.MapAggregatesToMatchRisk(p, mapCfg, at)
				if err != nil {
					return err
				}

				// record per-match risk for ledger
				players[mr.PlayerID] = mr.Value

				prev, err := rs.Load(mr.PlayerID)
				if err != nil {
					return err
				}

				if prev.PlayerID == "" {
					prev.PlayerID = mr.PlayerID
				}
				if prev.LastUpdate.IsZero() {
					prev.LastUpdate = mr.At
				}

				next := risk.ApplyMatchRisk(prev, mr, scoreCfg, mr.At)
				if err := rs.Save(next); err != nil {
					return err
				}

				logger.Printf(
					"match=%s player=%s match_risk=%.6f total_risk: %.6f -> %.6f at=%s",
					ev.MatchID,
					mr.PlayerID,
					mr.Value,
					prev.TotalRisk,
					next.TotalRisk,
					mr.At.UTC().Format(time.RFC3339Nano),
				)

				scored++
			}
		}

		if lastAt.IsZero() {
			lastAt = time.Now().UTC()
		}

		if err := ledger.AppendMatchLine(cfg.riskDir, ledger.MatchLine{
			MatchID: ev.MatchID,
			At:      lastAt,
			Players: players,
		}); err != nil {
			return err
		}

		if err := applied.MarkApplied(ev.MatchID); err != nil {
			return err
		}

		logger.Printf(
			"SUCCESS event=match_finalized match=%s players=%d",
			ev.MatchID,
			scored,
		)
		return nil
	})
}
