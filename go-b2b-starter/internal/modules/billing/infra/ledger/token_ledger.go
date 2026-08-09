package ledger

import (
	"context"
	"fmt"
	"time"

	"github.com/moasq/go-b2b-starter/internal/modules/billing/domain"
	"github.com/moasq/go-b2b-starter/internal/modules/billing/infra/credits"
	llmdomain "github.com/moasq/go-b2b-starter/internal/platform/llm/domain"
	logger "github.com/moasq/go-b2b-starter/internal/platform/logger/domain"
)

// aiTokensConsumedMeterSlug is the billing-provider meter fed with AI
// consumption. Created in the Polar dashboard (ops step); the MercadoPago
// adapter logs a no-op for meter events.
const aiTokensConsumedMeterSlug = "ai.tokens.consumed"

// tokenLedger implements llmdomain.TokenLedger on top of the billing module's
// AiUsageRepository. It resolves the current billing period, converts tokens
// to credits, records idempotently per request_id, and ingests a best-effort
// meter event to the billing provider.
type tokenLedger struct {
	aiRepo      domain.AiUsageRepository
	subRepo     domain.SubscriptionRepository
	orgAdapter  domain.OrganizationAdapter
	billing     domain.BillingProvider
	log         logger.Logger
}

// NewTokenLedger creates a TokenLedger backed by the local AI usage ledger.
func NewTokenLedger(
	aiRepo domain.AiUsageRepository,
	subRepo domain.SubscriptionRepository,
	orgAdapter domain.OrganizationAdapter,
	billing domain.BillingProvider,
	log logger.Logger,
) llmdomain.TokenLedger {
	return &tokenLedger{aiRepo: aiRepo, subRepo: subRepo, orgAdapter: orgAdapter, billing: billing, log: log}
}

func (l *tokenLedger) RecordUsage(ctx context.Context, event llmdomain.UsageEvent) error {
	periodStart, periodEnd := l.currentPeriod(ctx, event.OrganizationID)
	creditsUsed := credits.CreditsFor(event.Model, event.TokensInput, event.TokensOutput, event.TokensEmbedding)

	recorded, err := l.aiRepo.RecordUsage(ctx, &domain.AiUsageEvent{
		OrganizationID:  event.OrganizationID,
		Feature:         event.Feature,
		Model:           event.Model,
		TokensInput:     event.TokensInput,
		TokensOutput:    event.TokensOutput,
		TokensEmbedding: event.TokensEmbedding,
		CreditsConsumed: creditsUsed,
		RequestID:       event.RequestID,
	}, periodStart, periodEnd)
	if err != nil {
		return fmt.Errorf("failed to record ai usage: %w", err)
	}
	if !recorded {
		l.log.Info("duplicate ai usage event ignored", map[string]any{
			"organization_id": event.OrganizationID,
			"request_id":      event.RequestID,
		})
		return nil
	}

	l.log.Debug("recorded ai usage", map[string]any{
		"organization_id": event.OrganizationID,
		"feature":         event.Feature,
		"model":           event.Model,
		"credits":         creditsUsed,
	})

	// Ingest meter event to the billing provider (best-effort, async).
	go l.ingestMeterEventToProvider(event.OrganizationID, creditsUsed)

	return nil
}

// ingestMeterEventToProvider feeds the consumed credits to the billing
// provider's usage meter. Best-effort: failures are logged and never affect
// the AI response or the local ledger.
func (l *tokenLedger) ingestMeterEventToProvider(organizationID int32, creditsUsed int32) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	externalID, err := l.orgAdapter.GetStytchOrgID(ctx, organizationID)
	if err != nil {
		l.log.Error("failed to get external customer ID for ai meter event", map[string]any{
			"organization_id": organizationID,
			"error":           err.Error(),
		})
		return
	}

	if err := l.billing.IngestMeterEvent(ctx, externalID, aiTokensConsumedMeterSlug, creditsUsed); err != nil {
		// The MercadoPago adapter logs a no-op for meter events; anything else
		// is a connectivity/provider issue and must not block usage recording.
		l.log.Error("failed to ingest ai meter event", map[string]any{
			"organization_id": organizationID,
			"external_id":     externalID,
			"meter_slug":      aiTokensConsumedMeterSlug,
			"amount":          creditsUsed,
			"error":           err.Error(),
		})
		return
	}

	l.log.Info("ingested ai meter event", map[string]any{
		"organization_id": organizationID,
		"external_id":     externalID,
		"meter_slug":      aiTokensConsumedMeterSlug,
		"amount":          creditsUsed,
	})
}

// currentPeriod resolves the org's active billing period: quota_tracking
// first (authoritative for period bounds), then the subscription period,
// falling back to a rolling 30-day window.
func (l *tokenLedger) currentPeriod(ctx context.Context, organizationID int32) (time.Time, time.Time) {
	if quota, err := l.subRepo.GetQuotaByOrgID(ctx, organizationID); err == nil && !quota.PeriodStart.IsZero() {
		return quota.PeriodStart, quota.PeriodEnd
	}
	if sub, err := l.subRepo.GetSubscriptionByOrgID(ctx, organizationID); err == nil {
		return sub.CurrentPeriodStart, sub.CurrentPeriodEnd
	}
	now := time.Now()
	return now, now.AddDate(0, 1, 0)
}
