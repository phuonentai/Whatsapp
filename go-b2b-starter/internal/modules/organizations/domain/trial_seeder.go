// Package domain defines the organizations module domain contracts.
// The TrialSeeder interface is a narrow port injected via DI so the
// organizations module can trigger trial seeding without importing billing
// repositories or infrastructure — governance-compliant module boundary.
package domain

import (
	"context"
	"time"
)

// TrialSeeder seeds a local trial subscription for a newly bootstrapped
// organization. The implementation lives in the billing infrastructure layer
// and is injected via dig; the organizations module never imports billing
// packages directly.
//
// Seeding is idempotent (ON CONFLICT DO NOTHING) — a bootstrap retry
// that runs SeedTrial again will not duplicate or overwrite rows.
// If a provider-backed subscription already exists the trial insert
// is a no-op.
type TrialSeeder interface {
	SeedTrial(ctx context.Context, organizationID int32, trialEnd time.Time) error
}
