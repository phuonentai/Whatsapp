package scheduler

import (
	"context"
	"time"

	"github.com/moasq/go-b2b-starter/internal/modules/inquiryschedule/app/services"
	"github.com/moasq/go-b2b-starter/internal/modules/inquiryschedule/domain"
	logger "github.com/moasq/go-b2b-starter/internal/platform/logger/domain"
)

const (
	// DefaultSweepInterval is the follow-up sweep cadence (≤15 min per spec).
	DefaultSweepInterval = 15 * time.Minute
	// SweepBatch is the max candidates processed per org per sweep.
	SweepBatch = 100
)

// FollowUpSweeper periodically scans orgs with follow-ups enabled and
// enqueues candidate nudges through the same idempotent path as the
// reply-arrival trigger (the atomic nudge-count increment is the guard).
type FollowUpSweeper struct {
	orgs     domain.FollowUpEnabledOrgLister
	service  *services.FollowUpService
	interval time.Duration
	batch    int32
	log      logger.Logger
}

// NewFollowUpSweeper builds the sweep loop.
func NewFollowUpSweeper(
	orgs domain.FollowUpEnabledOrgLister,
	service *services.FollowUpService,
	log logger.Logger,
) *FollowUpSweeper {
	return &FollowUpSweeper{
		orgs:     orgs,
		service:  service,
		interval: DefaultSweepInterval,
		batch:    SweepBatch,
		log:      log,
	}
}

// WithInterval overrides the sweep cadence (tests).
func (s *FollowUpSweeper) WithInterval(d time.Duration) *FollowUpSweeper {
	s.interval = d
	return s
}

// Run loops until ctx is cancelled.
func (s *FollowUpSweeper) Run(ctx context.Context) {
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			s.log.Info("follow-up sweep stopped", nil)
			return
		case <-ticker.C:
			if err := s.SweepOnce(ctx); err != nil {
				s.log.Error("follow-up sweep failed", map[string]any{"error": err.Error()})
			}
		}
	}
}

// SweepOnce scans all orgs with follow-ups enabled; exposed for tests.
func (s *FollowUpSweeper) SweepOnce(ctx context.Context) error {
	orgs, err := s.orgs.ListFollowUpEnabledOrgs(ctx)
	if err != nil {
		return err
	}
	for _, orgID := range orgs {
		nudged, err := s.service.SweepOrg(ctx, orgID, s.batch)
		if err != nil {
			s.log.Error("follow-up sweep failed for org", map[string]any{
				"organization_id": orgID, "error": err.Error(),
			})
			continue
		}
		if nudged > 0 {
			s.log.Info("follow-up nudges enqueued", map[string]any{"organization_id": orgID, "count": nudged})
		}
	}
	return nil
}
