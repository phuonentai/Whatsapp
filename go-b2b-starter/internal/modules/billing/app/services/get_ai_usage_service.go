package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/moasq/go-b2b-starter/internal/modules/billing/domain"
)

// GetAiUsageStatus returns the read-only AI usage state for the organization's
// current billing period: token totals, credits used, allowance, and remaining.
// When no allowance is configured, CreditsMax is 0 (no blocking applies).
func (s *billingService) GetAiUsageStatus(ctx context.Context, organizationID int32) (*domain.AiUsageStatus, error) {
	periodStart, periodEnd := s.currentAiPeriod(ctx, organizationID)

	creditsMax := int32(0)
	quota, err := s.repo.GetQuotaByOrgID(ctx, organizationID)
	if err == nil {
		creditsMax = quota.AiCreditsMax
	}

	aiUsage, err := s.aiRepo.GetAiUsageByOrgAndPeriod(ctx, organizationID, periodStart)
	if err != nil {
		if errors.Is(err, domain.ErrAiUsageNotFound) {
			return &domain.AiUsageStatus{
				OrganizationID:   organizationID,
				PeriodStart:      periodStart,
				PeriodEnd:        periodEnd,
				CreditsMax:       creditsMax,
				CreditsRemaining: creditsMax,
			}, nil
		}
		return nil, fmt.Errorf("failed to get ai usage status: %w", err)
	}

	remaining := creditsMax - aiUsage.CreditsUsed
	if remaining < 0 {
		remaining = 0
	}

	return &domain.AiUsageStatus{
		OrganizationID:   organizationID,
		PeriodStart:      aiUsage.PeriodStart,
		PeriodEnd:        aiUsage.PeriodEnd,
		TokensInput:      aiUsage.TokensInput,
		TokensOutput:     aiUsage.TokensOutput,
		TokensEmbedding:  aiUsage.TokensEmbedding,
		CreditsUsed:      aiUsage.CreditsUsed,
		CreditsMax:       creditsMax,
		CreditsRemaining: remaining,
	}, nil
}

// currentAiPeriod resolves the org's active billing period: quota_tracking
// first, then the subscription period, falling back to a rolling 30-day window.
func (s *billingService) currentAiPeriod(ctx context.Context, organizationID int32) (time.Time, time.Time) {
	if quota, err := s.repo.GetQuotaByOrgID(ctx, organizationID); err == nil && !quota.PeriodStart.IsZero() {
		return quota.PeriodStart, quota.PeriodEnd
	}
	if sub, err := s.repo.GetSubscriptionByOrgID(ctx, organizationID); err == nil {
		return sub.CurrentPeriodStart, sub.CurrentPeriodEnd
	}
	now := time.Now()
	return now, now.AddDate(0, 1, 0)
}
