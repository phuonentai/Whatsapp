package repositories

import (
	"context"
	"encoding/json"

	"github.com/moasq/go-b2b-starter/internal/db/helpers"
	sqlc "github.com/moasq/go-b2b-starter/internal/db/postgres/sqlc/gen"
	"github.com/moasq/go-b2b-starter/internal/modules/procurement/domain"
)

// auditRepository records append-only procurement audit rows.
type auditRepository struct {
	store sqlc.Store
}

// NewAuditRepository builds the audit repository.
func NewAuditRepository(store sqlc.Store) domain.AuditRepository {
	return &auditRepository{store: store}
}

func (r *auditRepository) Record(ctx context.Context, entry domain.AuditEntry) error {
	metadata, err := json.Marshal(entry.Metadata)
	if err != nil {
		return err
	}
	if entry.Metadata == nil {
		metadata = []byte(`{}`)
	}
	_, err = r.store.InsertProcurementAudit(ctx, sqlc.InsertProcurementAuditParams{
		OrganizationID: entry.OrganizationID,
		EntityType:     entry.EntityType,
		EntityID:       helpers.ToPgInt4Ptr(entry.EntityID),
		Action:         entry.Action,
		Decision:       entry.Decision,
		Reason:         helpers.ToPgTextPtr(entry.Reason),
		MemberID:       helpers.ToPgTextPtr(entry.MemberID),
		Metadata:       metadata,
	})
	return err
}
