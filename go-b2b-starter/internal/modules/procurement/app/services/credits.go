package services

import (
	"context"
	"time"

	billingServices "github.com/moasq/go-b2b-starter/internal/modules/billing/app/services"
)

// creditsExhausted reports whether the org's AI credits are exhausted
// (CreditsMax > 0 and no remaining). An unset allowance never blocks.
// Fail-open: ledger errors do not block AI paths (mirrors the cognitive
// credit guard).
func creditsExhausted(ctx context.Context, billing billingServices.BillingService, orgID int32) bool {
	if billing == nil {
		return false
	}
	status, err := billing.GetAiUsageStatus(ctx, orgID)
	if err != nil {
		return false
	}
	return status.CreditsMax > 0 && status.CreditsRemaining <= 0
}

func strPtr(s string) *string { return &s }

// nowTime is a test-friendly time helper.
func nowTime() time.Time { return time.Now() }