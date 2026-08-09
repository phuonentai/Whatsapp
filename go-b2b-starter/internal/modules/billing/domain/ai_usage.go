package domain

import (
	"context"
	"errors"
	"time"
)

// ErrAiUsageNotFound is returned when no AI usage row exists for a period.
var ErrAiUsageNotFound = errors.New("ai usage not found")

// AiUsage is the running token/credit totals for an organization's billing period.
type AiUsage struct {
	OrganizationID  int32
	PeriodStart     time.Time
	PeriodEnd       time.Time
	TokensInput     int64
	TokensOutput    int64
	TokensEmbedding int64
	CreditsUsed     int32
}

// AiUsageEvent is an immutable, auditable consumption record.
type AiUsageEvent struct {
	OrganizationID  int32
	Feature         string
	Model           string
	TokensInput     int64
	TokensOutput    int64
	TokensEmbedding int64
	CreditsConsumed int32
	RequestID       string
}

// AiUsageStatus is the read-only view of an organization's current-period AI
// usage, including the credit allowance and remaining credits.
type AiUsageStatus struct {
	OrganizationID  int32
	PeriodStart     time.Time
	PeriodEnd       time.Time
	TokensInput     int64
	TokensOutput    int64
	TokensEmbedding int64
	CreditsUsed     int32
	CreditsMax      int32
	CreditsRemaining int32
}

// AiUsageRepository provides database operations for the AI usage ledger.
type AiUsageRepository interface {
	// RecordUsage appends the event and increments the period totals.
	// Returns (true, nil) when recorded; (false, nil) when the request_id
	// was already recorded (idempotent duplicate).
	RecordUsage(ctx context.Context, event *AiUsageEvent, periodStart, periodEnd time.Time) (bool, error)

	// GetAiUsageByOrgAndPeriod returns the period totals for an organization.
	GetAiUsageByOrgAndPeriod(ctx context.Context, organizationID int32, periodStart time.Time) (*AiUsage, error)

	// UpdateAiCreditsMax sets the period AI credit allowance without touching
	// invoice counters.
	UpdateAiCreditsMax(ctx context.Context, organizationID int32, creditsMax int32, periodStart, periodEnd time.Time) error
}
