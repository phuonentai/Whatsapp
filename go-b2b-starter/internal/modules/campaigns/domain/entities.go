package domain

import "time"

// Filter is one clause of a segment filter spec. Value is a JSON-decoded
// scalar (string/number) or array (tag_ids). Semantics: all filters AND-ed.
type Filter struct {
	Field string `json:"field"`
	Op    string `json:"op"`
	Value any    `json:"value"`
}

// Segment is a saved, org-scoped audience definition.
type Segment struct {
	ID             int32
	OrganizationID int32
	Nombre         string
	FilterSpec     []Filter
	CreatedBy      string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// CampaignStatus models the campaign lifecycle.
type CampaignStatus string

const (
	CampaignDraft CampaignStatus = "draft"
	CampaignReady CampaignStatus = "ready"
)

// Campaign is a draft or launched campaign. v1 covers draft -> ready
// (audience captured); the scheduler consumes recipients later.
type Campaign struct {
	ID             int32
	OrganizationID int32
	Nombre         string
	SegmentID      int32
	Status         CampaignStatus
	RecipientCount int32
	LaunchedAt     *time.Time
	CreatedBy      string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// RecipientStatus is the per-recipient send lifecycle.
type RecipientStatus string

const (
	RecipientPending RecipientStatus = "pending"
	RecipientSent    RecipientStatus = "sent"
	RecipientFailed  RecipientStatus = "failed"
	RecipientSkipped RecipientStatus = "skipped"
)

// CampaignRecipient is one row of the audience snapshot.
type CampaignRecipient struct {
	ID               int32
	CampaignID       int32
	ContactID        int32
	Status           RecipientStatus
	WhatsappMessageID string
	Error            string
	PhoneNumber      string
	DisplayName      string
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// EvalResult is the segment evaluation summary (post hard gates).
type EvalResult struct {
	Total           int64
	ExcludedByGates int64
}

// AudienceBuildResult is the AI builder output: a validated candidate spec
// plus its live preview. Nothing is persisted by the builder.
type AudienceBuildResult struct {
	FilterSpec []Filter
	Preview    *EvalResult
}
