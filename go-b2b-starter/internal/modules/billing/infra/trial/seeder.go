// Package trial provides the billing infrastructure implementation of the
// organizations domain TrialSeeder port. It seeds idempotent local trial
// subscription + quota rows for newly bootstrapped organizations.
package trial

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	postgres "github.com/moasq/go-b2b-starter/internal/db/postgres/sqlc/gen"
	organizationsDomain "github.com/moasq/go-b2b-starter/internal/modules/organizations/domain"
	logger "github.com/moasq/go-b2b-starter/internal/platform/logger/domain"
)

// Config gates and parameterizes trial seeding.
type Config struct {
	Enabled bool
	Days    int
}

// Seeder implements organizationsDomain.TrialSeeder.
type Seeder struct {
	store  postgres.Store
	config Config
	log    logger.Logger
}

// NewSeeder builds the TrialSeeder implementation.
func NewSeeder(store postgres.Store, config Config, log logger.Logger) organizationsDomain.TrialSeeder {
	return &Seeder{store: store, config: config, log: log}
}

func (s *Seeder) SeedTrial(ctx context.Context, organizationID int32, trialEnd time.Time) error {
	if !s.config.Enabled {
		s.log.Info("trial seeding disabled, skipping", map[string]any{
			"organization_id": organizationID,
		})
		return nil
	}

	// InsertLocalTrial: ON CONFLICT DO NOTHING — never overwrites a provider
	// row; bootstrap retry is a harmless no-op.
	_, err := s.store.InsertLocalTrial(ctx, postgres.InsertLocalTrialParams{
		OrganizationID:   organizationID,
		CurrentPeriodEnd: pgtype.Timestamp{Time: trialEnd, Valid: true},
	})
	if err != nil {
		return fmt.Errorf("insert local trial: %w", err)
	}

	// InsertLocalTrialQuota: zero invoice_count, no quota.
	_, err = s.store.InsertLocalTrialQuota(ctx, postgres.InsertLocalTrialQuotaParams{
		OrganizationID: organizationID,
		PeriodEnd:      pgtype.Timestamp{Time: trialEnd, Valid: true},
	})
	if err != nil {
		return fmt.Errorf("insert local trial quota: %w", err)
	}

	s.log.Info("local trial seeded", map[string]any{
		"organization_id": organizationID,
		"trial_end":       trialEnd.Format(time.RFC3339),
	})
	return nil
}
