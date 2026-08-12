package postgres

import (
	"context"
	"encoding/json"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/moasq/go-b2b-starter/internal/db/helpers"
	sqlc "github.com/moasq/go-b2b-starter/internal/db/postgres/sqlc/gen"
	"github.com/moasq/go-b2b-starter/internal/modules/inquiryschedule/domain"
)

// auditRecord appends one immutable row to procurement.audit_log.
func auditRecord(ctx context.Context, s sqlc.Store, orgID int32, entityType string, entityID *int32, action string, reason *string, metadata map[string]any) error {
	meta, err := json.Marshal(metadata)
	if err != nil {
		return err
	}
	if metadata == nil {
		meta = []byte(`{}`)
	}
	decision := "allow"
	if reason != nil {
		decision = "skip"
	}
	_, err = s.InsertProcurementAudit(ctx, sqlc.InsertProcurementAuditParams{
		OrganizationID: orgID,
		EntityType:     entityType,
		EntityID:       helpers.ToPgInt4Ptr(entityID),
		Action:         action,
		Decision:       decision,
		Reason:         helpers.ToPgTextPtr(reason),
		Metadata:       meta,
	})
	return err
}

// ------------- CatalogReader -------------

type catalogReader struct{ store sqlc.Store }

// NewCatalogReader builds the org-scope validation port over the sibling's
// products/suppliers tables.
func NewCatalogReader(store sqlc.Store) domain.CatalogReader {
	return &catalogReader{store: store}
}

func (c *catalogReader) ProductMembership(ctx context.Context, orgID int32, productIDs []int32) (map[int32]bool, error) {
	return membership(ctx, func() ([]int32, error) {
		rows, err := c.store.ListProductsByIDs(ctx, sqlc.ListProductsByIDsParams{
			OrganizationID: orgID,
			Column2:        productIDs,
		})
		if err != nil {
			return nil, err
		}
		ids := make([]int32, 0, len(rows))
		for _, r := range rows {
			ids = append(ids, r.ID)
		}
		return ids, nil
	}, productIDs)
}

func (c *catalogReader) SupplierMembership(ctx context.Context, orgID int32, supplierIDs []int32) (map[int32]bool, error) {
	return membership(ctx, func() ([]int32, error) {
		rows, err := c.store.ListSuppliersWithDisplay(ctx, sqlc.ListSuppliersWithDisplayParams{
			OrganizationID: orgID,
			Column2:        supplierIDs,
		})
		if err != nil {
			return nil, err
		}
		ids := make([]int32, 0, len(rows))
		for _, r := range rows {
			ids = append(ids, r.ID)
		}
		return ids, nil
	}, supplierIDs)
}

func membership(ctx context.Context, list func() ([]int32, error), requested []int32) (map[int32]bool, error) {
	found := make(map[int32]bool, len(requested))
	ids, err := list()
	if err != nil {
		return nil, err
	}
	for _, id := range ids {
		found[id] = true
	}
	return found, nil
}

// ------------- KillSwitchReader / OrgTimezoneReader -------------

type governanceReader struct{ store sqlc.Store }

// NewGovernanceReader builds the kill-switch and org-timezone seams over
// agent.agent_settings.
func NewGovernanceReader(store sqlc.Store) *governanceReader {
	return &governanceReader{store: store}
}

func (g *governanceReader) IsKillSwitchEnabled(ctx context.Context, orgID int32) (bool, error) {
	return g.store.GetAgentKillSwitch(ctx, orgID)
}

func (g *governanceReader) Timezone(ctx context.Context, orgID int32) (string, error) {
	return g.store.GetOrgTimezone(ctx, orgID)
}

// ------------- RecipientStateReader / NudgeIncrementer / FollowUpEnqueuer -------------

type recipientReader struct {
	store sqlc.Store
	clock domain.Clock
}

// NewRecipientReader builds the follow-up candidate and nudge-guard ports.
func NewRecipientReader(store sqlc.Store, clock domain.Clock) *recipientReader {
	return &recipientReader{store: store, clock: clock}
}

func buildCandidate(
	recipientID, organizationID, runID, supplierID, contactID int32,
	recipientStatus, runStatus string,
	sentAt pgtype.Timestamptz,
	followupCount int32,
	contactPhone, consentStatus, displayName, nit string,
	deadlineHours, maxNudges int32,
	template string,
) *domain.FollowUpCandidate {
	return &domain.FollowUpCandidate{
		RecipientID:         recipientID,
		OrganizationID:      organizationID,
		RunID:               runID,
		SupplierID:          supplierID,
		ContactID:           contactID,
		RecipientStatus:     recipientStatus,
		SentAt:              helpers.FromPgTimestamptzPtr(sentAt),
		FollowupCount:       followupCount,
		RunStatus:           runStatus,
		ContactPhone:        contactPhone,
		ConsentStatus:       consentStatus,
		SupplierDisplayName: displayName,
		SupplierNIT:         nit,
		DeadlineHours:       deadlineHours,
		MaxNudges:           maxNudges,
		MessageTemplate:     template,
		SettingsPresent:     true,
	}
}

func mapCandidate(row sqlc.ListFollowUpCandidatesRow) *domain.FollowUpCandidate {
	return buildCandidate(
		row.RecipientID, row.OrganizationID, row.RunID, row.SupplierID, row.ContactID,
		row.RecipientStatus, row.RunStatus, row.SentAt, row.FollowupCount,
		row.ContactPhone.String, row.ConsentStatus, row.SupplierDisplayName.String, row.SupplierNit,
		row.DeadlineHours, row.MaxNudges, row.MessageTemplate,
	)
}

func mapRunCandidate(row sqlc.ListOverdueRecipientsForRunRow) *domain.FollowUpCandidate {
	return buildCandidate(
		row.RecipientID, row.OrganizationID, row.RunID, row.SupplierID, row.ContactID,
		row.RecipientStatus, row.RunStatus, row.SentAt, row.FollowupCount,
		row.ContactPhone.String, row.ConsentStatus, row.SupplierDisplayName.String, row.SupplierNit,
		row.DeadlineHours, row.MaxNudges, row.MessageTemplate,
	)
}

func mapContactCandidate(row sqlc.ListOverdueRecipientsForContactRow) *domain.FollowUpCandidate {
	return buildCandidate(
		row.RecipientID, row.OrganizationID, row.RunID, row.SupplierID, row.ContactID,
		row.RecipientStatus, row.RunStatus, row.SentAt, row.FollowupCount,
		row.ContactPhone.String, row.ConsentStatus, row.SupplierDisplayName.String, row.SupplierNit,
		row.DeadlineHours, row.MaxNudges, row.MessageTemplate,
	)
}

func mapTarget(row sqlc.GetFollowUpTargetRow) *domain.FollowUpCandidate {
	return &domain.FollowUpCandidate{
		RecipientID:         row.RecipientID,
		OrganizationID:      row.OrganizationID,
		RunID:               row.RunID,
		SupplierID:          row.SupplierID,
		ContactID:           row.ContactID,
		RecipientStatus:     row.RecipientStatus,
		SentAt:              helpers.FromPgTimestamptzPtr(row.SentAt),
		FollowupCount:       row.FollowupCount,
		RunStatus:           row.RunStatus,
		ContactPhone:        row.ContactPhone.String,
		ConsentStatus:       row.ConsentStatus,
		SupplierDisplayName: row.SupplierDisplayName.String,
		SupplierNIT:         row.SupplierNit,
		DeadlineHours:       int32(row.DeadlineHours.Int32),
		MaxNudges:           int32(row.MaxNudges.Int32),
		MessageTemplate:     row.MessageTemplate.String,
		SettingsPresent:     row.DeadlineHours.Valid,
	}
}

func (r *recipientReader) ListFollowUpCandidates(ctx context.Context, orgID int32, limit int32) ([]*domain.FollowUpCandidate, error) {
	rows, err := r.store.ListFollowUpCandidates(ctx, sqlc.ListFollowUpCandidatesParams{
		OrganizationID: orgID,
		Limit:          limit,
	})
	if err != nil {
		return nil, err
	}
	out := make([]*domain.FollowUpCandidate, 0, len(rows))
	for i := range rows {
		out = append(out, mapCandidate(rows[i]))
	}
	return out, nil
}

func (r *recipientReader) ListOverdueRecipientsForRun(ctx context.Context, orgID, runID int32) ([]*domain.FollowUpCandidate, error) {
	rows, err := r.store.ListOverdueRecipientsForRun(ctx, sqlc.ListOverdueRecipientsForRunParams{
		RunID:          runID,
		OrganizationID: orgID,
	})
	if err != nil {
		return nil, err
	}
	out := make([]*domain.FollowUpCandidate, 0, len(rows))
	for i := range rows {
		out = append(out, mapRunCandidate(rows[i]))
	}
	return out, nil
}

func (r *recipientReader) ListOverdueRecipientsForContact(ctx context.Context, orgID, contactID int32) ([]*domain.FollowUpCandidate, error) {
	rows, err := r.store.ListOverdueRecipientsForContact(ctx, sqlc.ListOverdueRecipientsForContactParams{
		ContactID:      contactID,
		OrganizationID: orgID,
	})
	if err != nil {
		return nil, err
	}
	out := make([]*domain.FollowUpCandidate, 0, len(rows))
	for i := range rows {
		out = append(out, mapContactCandidate(rows[i]))
	}
	return out, nil
}

func (r *recipientReader) ActiveRecipientsByPhone(ctx context.Context, orgID int32, phoneNumber string) ([]domain.RecipientRef, error) {
	rows, err := r.store.ListActiveRecipientsByPhone(ctx, sqlc.ListActiveRecipientsByPhoneParams{
		OrganizationID: orgID,
		PhoneNumber:    helpers.ToPgText(phoneNumber),
	})
	if err != nil {
		return nil, err
	}
	out := make([]domain.RecipientRef, 0, len(rows))
	for _, row := range rows {
		out = append(out, domain.RecipientRef{RecipientID: row.ID, RunID: row.RunID})
	}
	return out, nil
}

func (r *recipientReader) GetFollowUpTarget(ctx context.Context, orgID, recipientID int32) (*domain.FollowUpCandidate, error) {
	row, err := r.store.GetFollowUpTarget(ctx, sqlc.GetFollowUpTargetParams{
		ID:             recipientID,
		OrganizationID: orgID,
	})
	if isNoRows(err) {
		return nil, domain.ErrRecipientNotFound
	}
	if err != nil {
		return nil, err
	}
	return mapTarget(row), nil
}

func (r *recipientReader) TryIncrementFollowupCount(ctx context.Context, orgID, recipientID, maxNudges int32) (bool, error) {
	_, err := r.store.IncrementFollowupCount(ctx, sqlc.IncrementFollowupCountParams{
		ID:             recipientID,
		OrganizationID: orgID,
		FollowupCount:  maxNudges,
	})
	if isNoRows(err) {
		return false, nil // guard at the cap: no increment
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// EnqueueNudge atomically applies the nudge guard, enqueues the
// inquiry.followup_send outbox event, and audits inquiry_followup in ONE
// transaction. Returns false when the guard was at the cap (no event).
func (r *recipientReader) EnqueueNudge(ctx context.Context, orgID, recipientID int32, maxNudges int32, event domain.OutboxEventInput) (bool, error) {
	enqueued := false
	err := inTx(ctx, r.store, func(tx sqlc.Store) error {
		row, err := tx.IncrementFollowupCount(ctx, sqlc.IncrementFollowupCountParams{
			ID:             recipientID,
			OrganizationID: orgID,
			FollowupCount:  maxNudges,
		})
		if isNoRows(err) {
			return nil // cap reached: no increment, no event (double-nudge guard)
		}
		if err != nil {
			return err
		}
		if _, err := tx.InsertOutboxEvent(ctx, sqlc.InsertOutboxEventParams{
			EventType:      event.EventType,
			Payload:        event.Payload,
			OrganizationID: helpers.ToPgInt4Ptr(&orgID),
		}); err != nil {
			return err
		}
		if err := auditRecord(ctx, tx, orgID, "inquiry_recipient", &recipientID, "inquiry_followup",
			nil, map[string]any{
				"recipient_id": recipientID,
				"run_id":       row.RunID,
				"nudge_index":  row.FollowupCount,
			}); err != nil {
			return err
		}
		enqueued = true
		return nil
	})
	if err != nil {
		return false, err
	}
	return enqueued, nil
}

// ------------- FollowUpEnabledOrgLister -------------

func (r *recipientReader) ListFollowUpEnabledOrgs(ctx context.Context) ([]int32, error) {
	return r.store.ListFollowUpEnabledOrgs(ctx)
}

// ------------- AuditLogWriter -------------

type auditWriter struct{ store sqlc.Store }

// NewAuditWriter builds the append-only audit port.
func NewAuditWriter(store sqlc.Store) domain.AuditLogWriter {
	return &auditWriter{store: store}
}

func (a *auditWriter) Record(ctx context.Context, event domain.AuditEvent) error {
	return auditRecord(ctx, a.store, event.OrganizationID, event.EntityType, event.EntityID,
		event.Action, event.Reason, event.Metadata)
}
