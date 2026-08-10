package services

import (
	"context"

	"github.com/moasq/go-b2b-starter/internal/modules/campaigns/domain"
)

// SegmentService manages org-scoped saved segments and previews.
type SegmentService interface {
	Create(ctx context.Context, orgID int32, nombre string, spec []domain.Filter, createdBy string) (*domain.Segment, error)
	Update(ctx context.Context, orgID, id int32, nombre string, spec []domain.Filter) (*domain.Segment, error)
	Delete(ctx context.Context, orgID, id int32) error
	Get(ctx context.Context, orgID, id int32) (*domain.Segment, error)
	List(ctx context.Context, orgID int32) ([]*domain.Segment, error)
	// Preview evaluates an arbitrary (candidate or saved) spec with hard
	// gates and returns the counts. Persists nothing.
	Preview(ctx context.Context, orgID int32, spec []domain.Filter) (*domain.EvalResult, error)
}

// CampaignService manages campaign drafts, launches, and recipient snapshots.
type CampaignService interface {
	Create(ctx context.Context, orgID int32, nombre string, segmentID int32, createdBy string) (*domain.Campaign, error)
	Get(ctx context.Context, orgID, id int32) (*domain.Campaign, error)
	List(ctx context.Context, orgID int32) ([]*domain.Campaign, error)
	// Launch evaluates the campaign's segment, snapshots the audience
	// idempotently, and transitions draft -> ready (409 on relaunch).
	Launch(ctx context.Context, orgID, id int32, createdBy string) (*domain.Campaign, error)
	ListRecipients(ctx context.Context, orgID, campaignID int32, limit, offset int32) ([]*domain.CampaignRecipient, error)
}

// AudienceBuilderService converts natural language into a validated candidate
// filter spec with preview. Metered, PII-safe, nothing persisted.
type AudienceBuilderService interface {
	Build(ctx context.Context, orgID int32, naturalLanguage string) (*domain.AudienceBuildResult, error)
}
