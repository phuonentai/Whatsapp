package repositories

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/moasq/go-b2b-starter/internal/db/helpers"
	sqlc "github.com/moasq/go-b2b-starter/internal/db/postgres/sqlc/gen"
	"github.com/moasq/go-b2b-starter/internal/modules/procurement/domain"
)

// contactLookup implements domain.ContactLookup over the sqlc store
// (dispatch-time guards: consent + last-message window, D14).
type contactLookup struct {
	store sqlc.Store
}

// NewContactLookup builds the tenant-scoped contact lookup adapter.
func NewContactLookup(store sqlc.Store) domain.ContactLookup {
	return &contactLookup{store: store}
}

func (c *contactLookup) ContactByID(ctx context.Context, orgID, contactID int32) (*domain.ContactRef, error) {
	row, err := c.store.GetContactByID(ctx, sqlc.GetContactByIDParams{ID: contactID, OrganizationID: orgID})
	if isNoRows(err) {
		return nil, domain.ErrContactNotFound
	}
	if err != nil {
		return nil, err
	}
	return &domain.ContactRef{
		ID:            row.ID,
		PhoneNumber:   row.PhoneNumber.String,
		ConsentStatus: row.ConsentStatus,
		LastMessageAt: pgTimestampPtr(row.LastMessageAt),
	}, nil
}

func pgTimestampPtr(t pgtype.Timestamp) *time.Time {
	return helpers.FromPgTimestampPtr(t)
}
