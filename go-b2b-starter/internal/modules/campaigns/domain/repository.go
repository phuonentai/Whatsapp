package domain

import "context"

// SegmentRepository persists org-scoped segments.
type SegmentRepository interface {
	Create(ctx context.Context, organizationID int32, nombre string, spec []Filter, createdBy string) (*Segment, error)
	Update(ctx context.Context, organizationID, id int32, nombre string, spec []Filter) (*Segment, error)
	Delete(ctx context.Context, organizationID, id int32) error
	Get(ctx context.Context, organizationID, id int32) (*Segment, error)
	List(ctx context.Context, organizationID int32) ([]*Segment, error)
}

// CampaignRepository persists campaigns and the recipient snapshot.
type CampaignRepository interface {
	Create(ctx context.Context, organizationID int32, nombre string, segmentID int32, createdBy string) (*Campaign, error)
	Get(ctx context.Context, organizationID, id int32) (*Campaign, error)
	List(ctx context.Context, organizationID int32) ([]*Campaign, error)
	// Launch transitions draft -> ready guarded. Returns ErrCampaignNotDraft
	// when the campaign is not in draft state.
	Launch(ctx context.Context, organizationID, id int32, recipientCount int32) (*Campaign, error)
	// SnapshotRecipients inserts recipient rows idempotently and returns the
	// number of rows actually inserted.
	SnapshotRecipients(ctx context.Context, campaignID int32, contactIDs []int32) (int64, error)
	ListRecipients(ctx context.Context, campaignID int32, limit, offset int32) ([]*CampaignRecipient, error)
}

// SegmentEvaluator evaluates a filter spec against the organization's
// contacts, always appending the mandatory hard gates:
//   - consent_status = 'granted'
//   - valid E.164 phone number
type SegmentEvaluator interface {
	// Count returns total matches (post gates) and gate exclusions.
	Count(ctx context.Context, organizationID int32, spec []Filter) (*EvalResult, error)
	// ContactIDs returns the matched contact ids (post gates).
	ContactIDs(ctx context.Context, organizationID int32, spec []Filter) ([]int32, error)
}

// AudienceBuilder converts a natural-language audience description into a
// validated candidate filter spec plus preview. Implementations must use the
// metered LLM client with the organization id in context and must not send
// contact PII to the provider.
type AudienceBuilder interface {
	Build(ctx context.Context, organizationID int32, naturalLanguage string) (*AudienceBuildResult, error)
}
