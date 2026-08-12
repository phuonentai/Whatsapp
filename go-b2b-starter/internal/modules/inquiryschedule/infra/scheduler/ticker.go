// Package scheduler implements the two in-process loops of the
// inquiry-scheduling module: the schedule claim ticker (30–60s) and the
// follow-up sweep (≤15 min), both mirroring the durable outbox dispatcher
// pattern (context-cancellable, errors logged, never crash the loop).
package scheduler

import (
	"context"
	"time"

	"github.com/moasq/go-b2b-starter/internal/modules/inquiryschedule/domain"
	logger "github.com/moasq/go-b2b-starter/internal/platform/logger/domain"
)

const (
	// DefaultClaimInterval is the schedule claim cadence (30–60s per spec).
	DefaultClaimInterval = 45 * time.Second
	// ClaimBatch is the max schedules claimed per tick.
	ClaimBatch = 50
)

// ScheduleTicker claims due schedules and enqueues inquiry_run.scheduled
// outbox events. Concurrent tickers/replicas are safe: the repository claim
// uses FOR UPDATE SKIP LOCKED and advances next_run_at in the same
// transaction as the enqueue.
type ScheduleTicker struct {
	repo     domain.ScheduleRepository
	clock    domain.Clock
	interval time.Duration
	batch    int32
	log      logger.Logger
}

// NewScheduleTicker builds the claim ticker.
func NewScheduleTicker(repo domain.ScheduleRepository, clock domain.Clock, log logger.Logger) *ScheduleTicker {
	return &ScheduleTicker{
		repo:     repo,
		clock:    clock,
		interval: DefaultClaimInterval,
		batch:    ClaimBatch,
		log:      log,
	}
}

// WithInterval overrides the claim cadence (tests).
func (t *ScheduleTicker) WithInterval(d time.Duration) *ScheduleTicker {
	t.interval = d
	return t
}

// Run loops until ctx is cancelled.
func (t *ScheduleTicker) Run(ctx context.Context) {
	ticker := time.NewTicker(t.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			t.log.Info("schedule claim ticker stopped", nil)
			return
		case <-ticker.C:
			if err := t.TickOnce(ctx); err != nil {
				t.log.Error("schedule claim tick failed", map[string]any{"error": err.Error()})
			}
		}
	}
}

// TickOnce runs one claim cycle; exposed for tests and manual invocation.
func (t *ScheduleTicker) TickOnce(ctx context.Context) error {
	claimed, err := t.repo.ClaimAndAdvanceAndEnqueue(ctx, t.batch)
	if err != nil {
		return err
	}
	if len(claimed) > 0 {
		t.log.Info("scheduled inquiry runs enqueued", map[string]any{"count": len(claimed)})
	}
	return nil
}
