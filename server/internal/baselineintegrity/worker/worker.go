package worker

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/BlindedGlory/baseline-integrity/server/internal/baselineintegrity/outbox"
)

// ErrRetryLater tells the worker to requeue the claimed event back to pending,
// so it can be processed again later (grace window / eventual consistency).
var ErrRetryLater = errors.New("retry later")

type Config struct {
	OutboxDir    string
	PollInterval time.Duration
	Once         bool
}

type Claimed struct {
	Event          outbox.Event
	ProcessingPath string
}

type Worker struct {
	ob     *outbox.FSOutbox
	cfg    Config
	logger *log.Logger
}

func New(logger *log.Logger, cfg Config) (*Worker, error) {
	if logger == nil {
		return nil, errors.New("logger is required")
	}
	if cfg.OutboxDir == "" {
		return nil, errors.New("OutboxDir is required")
	}
	if cfg.PollInterval <= 0 {
		cfg.PollInterval = 1 * time.Second
	}

	ob := &outbox.FSOutbox{Dir: cfg.OutboxDir}
	if err := ob.Ensure(); err != nil {
		return nil, err
	}

	return &Worker{
		ob:     ob,
		cfg:    cfg,
		logger: logger,
	}, nil
}

func (w *Worker) Run(ctx context.Context, handler func(context.Context, Claimed) error) error {
	if handler == nil {
		return errors.New("handler is required")
	}

	if w.cfg.Once {
		return w.runOnce(ctx, handler)
	}

	t := time.NewTicker(w.cfg.PollInterval)
	defer t.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		ev, processingPath, err := w.ob.ClaimOne()
		if err != nil {
			if errors.Is(err, outbox.ErrNoPending) {
				// normal idle
			} else {
				w.logger.Printf("claim error: %v", err)
			}
		} else if processingPath != "" {
			claimed := Claimed{Event: ev, ProcessingPath: processingPath}

			if err := handler(ctx, claimed); err != nil {
				if errors.Is(err, ErrRetryLater) {
					_ = w.ob.Requeue(claimed.ProcessingPath)
				} else {
					_ = w.ob.MarkFailed(claimed.ProcessingPath, err)
				}
			} else {
				_ = w.ob.MarkDone(claimed.ProcessingPath)
			}
		}

		select {
		case <-ctx.Done():
			return nil
		case <-t.C:
		}
	}
}

func (w *Worker) runOnce(ctx context.Context, handler func(context.Context, Claimed) error) error {
	ev, processingPath, err := w.ob.ClaimOne()
	if err != nil {
		if errors.Is(err, outbox.ErrNoPending) {
			w.logger.Printf("no work (once mode)")
			return nil
		}
		w.logger.Printf("claim error (once mode): %v", err)
		return nil
	}
	if processingPath == "" {
		w.logger.Printf("no work (once mode)")
		return nil
	}

	claimed := Claimed{Event: ev, ProcessingPath: processingPath}

	if err := handler(ctx, claimed); err != nil {
		if errors.Is(err, ErrRetryLater) {
			_ = w.ob.Requeue(claimed.ProcessingPath)
		} else {
			_ = w.ob.MarkFailed(claimed.ProcessingPath, err)
		}
	} else {
		_ = w.ob.MarkDone(claimed.ProcessingPath)
	}

	return nil
}
