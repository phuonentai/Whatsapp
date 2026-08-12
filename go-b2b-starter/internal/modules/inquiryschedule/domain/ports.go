package domain

import (
	"context"
	"encoding/json"
	"time"
)

// CatalogReader validates referenced products/suppliers against the sibling
// supplier-inquiries capability data. The sibling change owns the procurement
// tables; this change consumes them through this port only, never raw SQL.
// Implemented by an infra adapter once the sibling's tables exist.
type CatalogReader interface {
	// ProductMembership reports, for each requested product id, whether it
	// belongs to orgID. Unknown or foreign products are false.
	ProductMembership(ctx context.Context, orgID int32, productIDs []int32) (map[int32]bool, error)

	// SupplierMembership is the same membership check for suppliers.
	SupplierMembership(ctx context.Context, orgID int32, supplierIDs []int32) (map[int32]bool, error)
}

// ScheduleRepository persists org-scoped schedules and their product/supplier
// join rows. All lookups are tenant-scoped (organization_id).
type ScheduleRepository interface {
	// Create persists the schedule and its join rows; next_run_at must be
	// precomputed by the caller.
	Create(ctx context.Context, orgID int32, s *Schedule) (*Schedule, error)
	Get(ctx context.Context, orgID, id int32) (*Schedule, error)
	// GetForUpdate locks the schedule row FOR UPDATE (handler-side dedupe).
	GetForUpdate(ctx context.Context, orgID, id int32) (*Schedule, error)
	List(ctx context.Context, orgID int32) ([]*Schedule, error)
	// Update persists schedule fields and replaces join rows; next_run_at is
	// recomputed by the caller.
	Update(ctx context.Context, orgID int32, s *Schedule) (*Schedule, error)
	Delete(ctx context.Context, orgID, id int32) error
	Pause(ctx context.Context, orgID, id int32) (*Schedule, error)
	// Resume sets is_active and recomputes next_run_at (strictly after now).
	Resume(ctx context.Context, orgID, id int32, nextRunAt time.Time) (*Schedule, error)
	// ClaimDue returns due schedules (is_active AND next_run_at <= now),
	// claimed atomically with FOR UPDATE SKIP LOCKED.
	ClaimDue(ctx context.Context, limit int32) ([]*Schedule, error)
	// ClaimAndAdvanceAndEnqueue claims up to limit due schedules and, for
	// each, persists the next next_run_at and enqueues the
	// inquiry_run.scheduled outbox event — one transaction (locks held
	// until commit; crash-before-commit re-fires on the next tick).
	ClaimAndAdvanceAndEnqueue(ctx context.Context, limit int32) ([]*Schedule, error)
	// MarkFiredOccurrence records last_run_at / last_run_occurrence_at.
	MarkFiredOccurrence(ctx context.Context, orgID, id int32, occurrence time.Time) (*Schedule, error)
	// ListWithStatus returns schedules with the status of their last run
	// (LastRunStatus nil/empty => never run).
	ListWithStatus(ctx context.Context, orgID int32) ([]*ScheduleStatus, error)
	// RecentRuns returns the most recent runs created from the schedule.
	RecentRuns(ctx context.Context, orgID, id int32, limit int32) ([]*ScheduledRun, error)
	// CountOverdueRecipients returns the count of 'sent' recipients of the
	// schedule's active runs whose followup_count reached the cap (the
	// overdue badge surfaced on the detail).
	CountOverdueRecipients(ctx context.Context, orgID, id int32) (int32, error)
}

// ScheduleStatus pairs a schedule with the status of its last run.
type ScheduleStatus struct {
	Schedule        *Schedule
	LastRunID       int32
	LastRunStatus   string // "" => never_run
	LastRunAt       *time.Time
	HasLastRun      bool
}

// ScheduledRun is a read-only run surface created from a schedule.
type ScheduledRun struct {
	ID             int32
	OrganizationID int32
	Status         string
	Source         string
	ScheduleRef    *int64
	Nota           *string
	CreatedAt      time.Time
}

// FollowUpSettingsRepository persists the org-level follow-up policy row.
type FollowUpSettingsRepository interface {
	// GetByOrg returns the row or ErrFollowUpSettingsNotFound when absent
	// (callers fall back to DefaultFollowUpSettings).
	GetByOrg(ctx context.Context, orgID int32) (*FollowUpSettings, error)
	// Upsert inserts or updates the single row per organization.
	Upsert(ctx context.Context, settings *FollowUpSettings) (*FollowUpSettings, error)
}

// Clock is the domain time source (test-friendly).
type Clock interface {
	Now() time.Time
}

// KillSwitchReader is the tenant-scoped agent kill-switch seam
// (agent.agent_settings.kill_switch; absent row => false).
type KillSwitchReader interface {
	IsKillSwitchEnabled(ctx context.Context, orgID int32) (bool, error)
}

// OrgTimezoneReader resolves the organization's IANA timezone
// (agent.agent_settings.timezone, default America/Bogota).
type OrgTimezoneReader interface {
	Timezone(ctx context.Context, orgID int32) (string, error)
}

// AuditEvent is one append-only procurement audit entry written through the
// AuditLogWriter port (procurement.audit_log).
type AuditEvent struct {
	OrganizationID int32
	EntityType     string
	EntityID       *int32
	Action         string
	Decision       string // allow | skip
	Reason         *string
	MemberID       *string
	Metadata       map[string]any
}

// AuditLogWriter appends immutable rows to the procurement audit ledger.
type AuditLogWriter interface {
	Record(ctx context.Context, event AuditEvent) error
}

// DraftSupplier is the minimal supplier surface the drafting seam needs
// (NIT persona jurídica display name, contact, active state).
type DraftSupplier struct {
	ID          int32
	NIT         string
	DisplayName string
	ContactID   int32
	IsActive    bool
}

// DraftProduct is the minimal product surface for the drafting prompt.
type DraftProduct struct {
	ID   int32
	Name string
	SKU  string
	Unit string
}

// DraftFunc drafts one personalized Spanish inquiry message for a supplier
// (the metered supplier-inquiries drafting step; scheduled run creation
// reuses it through this seam). The production implementation adapts the
// sibling capability's DraftingService.
type DraftFunc func(ctx context.Context, orgID int32, supplier DraftSupplier, products []DraftProduct) (string, error)

// CreateScheduledRunInput is what the inquiry_run.scheduled handler passes to
// the supplier-inquiries run-creation service through the port.
type CreateScheduledRunInput struct {
	OrganizationID int32
	ScheduleID     int32
	OccurrenceAt   time.Time
	ProductIDs     []int32
	SupplierIDs    []int32
	Note           string
}

// ScheduledRunResult reports the created run. Escalated is true when the run
// was created as escalated (credits exhausted / drafting failed); Skipped is
// true when the occurrence was already fired (duplicate dispatch).
type ScheduledRunResult struct {
	RunID        int32
	Escalated    bool
	Skipped      bool
	DraftedCount int32
}

// InquiryRunCreator is the supplier-inquiries run-creation entrypoint for
// scheduled occurrences: creates the run (source='scheduled', schedule_ref),
// drafts one metered message per supplier, and stores one recipient per
// supplier. Credits exhaustion / drafting failure escalate the run.
type InquiryRunCreator interface {
	CreateScheduledRun(ctx context.Context, in CreateScheduledRunInput) (*ScheduledRunResult, error)
}

// FollowUpCandidate is a recipient eligible for (or being evaluated for) a
// follow-up nudge, joined with the send target and policy data.
type FollowUpCandidate struct {
	RecipientID         int32
	OrganizationID      int32
	RunID               int32
	SupplierID          int32
	ContactID           int32
	RecipientStatus     string
	SentAt              *time.Time
	FollowupCount       int32
	RunStatus           string
	ContactPhone        string
	ConsentStatus       string
	SupplierDisplayName string
	SupplierNIT         string
	DeadlineHours       int32
	MaxNudges           int32
	MessageTemplate     string
	SettingsPresent     bool // false => org row absent (defaults apply)
}

// RecipientRef identifies an active recipient + its run (reply-arrival
// exclusion set).
type RecipientRef struct {
	RecipientID int32
	RunID       int32
}

// RecipientStateReader surfaces follow-up candidates (sweep + reply-trigger)
// and the dispatch-time re-validation target.
type RecipientStateReader interface {
	// ListFollowUpCandidates returns overdue candidates org-wide (sweep).
	ListFollowUpCandidates(ctx context.Context, orgID int32, limit int32) ([]*FollowUpCandidate, error)
	// ListOverdueRecipientsForRun returns overdue candidates of one run
	// (cheap reply-arrival check).
	ListOverdueRecipientsForRun(ctx context.Context, orgID, runID int32) ([]*FollowUpCandidate, error)
	// ListOverdueRecipientsForContact returns overdue candidates of a contact
	// (reply-arrival check).
	ListOverdueRecipientsForContact(ctx context.Context, orgID, contactID int32) ([]*FollowUpCandidate, error)
	// ActiveRecipientsByPhone returns the just-answered recipient refs for a
	// phone in active runs (exclusion set for the reply-arrival check).
	ActiveRecipientsByPhone(ctx context.Context, orgID int32, phoneNumber string) ([]RecipientRef, error)
	// GetFollowUpTarget re-validates one recipient at dispatch time.
	GetFollowUpTarget(ctx context.Context, orgID, recipientID int32) (*FollowUpCandidate, error)
}

// NudgeIncrementer performs the atomic conditional nudge guard: increments
// followup_count only while below the cap. Returns false when the cap was
// reached (no increment, no double-nudge).
type NudgeIncrementer interface {
	TryIncrementFollowupCount(ctx context.Context, orgID, recipientID, maxNudges int32) (bool, error)
}

// OutboxEventInput is a durable outbox event to enqueue inside a transaction.
type OutboxEventInput struct {
	EventType string
	Payload   json.RawMessage
}

// FollowUpEnqueuer atomically enqueues an inquiry.followup_send event with the
// nudge guard and the inquiry_followup audit in ONE transaction. Returns false
// when the guard was at the cap (nothing enqueued).
type FollowUpEnqueuer interface {
	EnqueueNudge(ctx context.Context, orgID, recipientID int32, maxNudges int32, event OutboxEventInput) (bool, error)
}

// FollowUpEnabledOrgLister returns the orgs with follow-ups enabled (sweep).
type FollowUpEnabledOrgLister interface {
	ListFollowUpEnabledOrgs(ctx context.Context) ([]int32, error)
}

// FollowUpSend is the payload of one durable inquiry.followup_send event.
type FollowUpSend struct {
	RunID           int32
	OrganizationID  int32
	SupplierID      int32
	ContactID       int32
	RecipientID     int32
	ContactPhone    string
	Message         string
	NudgeIndex      int32
}

// FollowUpSender dispatches a follow-up message through the circuit-breakered,
// rate-limited WhatsApp outbound client. Returns the provider message id.
type FollowUpSender interface {
	SendFollowUp(ctx context.Context, orgID int32, send *FollowUpSend) (string, error)
}
