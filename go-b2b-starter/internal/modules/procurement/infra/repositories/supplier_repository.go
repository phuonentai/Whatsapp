// Package repositories implements the procurement domain repositories over
// SQLC, using the store.Transaction seam for multi-statement atomicity
// (same pattern as the whatsapp webhook-log repository).
package repositories

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/moasq/go-b2b-starter/internal/db/helpers"
	sqlc "github.com/moasq/go-b2b-starter/internal/db/postgres/sqlc/gen"
	"github.com/moasq/go-b2b-starter/internal/modules/procurement/domain"
)

// transactioner is implemented by *sqlc.SQLStore (gen/exec.go) and lets
// repositories compose queries atomically. Non-transactional stores run the
// function directly (test fakes).
type transactioner interface {
	Transaction(ctx context.Context, fn func(sqlc.Store) error) error
}

func inTx(ctx context.Context, store sqlc.Store, fn func(sqlc.Store) error) error {
	if t, ok := store.(transactioner); ok {
		return t.Transaction(ctx, fn)
	}
	return fn(store)
}

func mapSupplier(row sqlc.ProcurementSupplier) *domain.Supplier {
	return &domain.Supplier{
		ID:             row.ID,
		OrganizationID: row.OrganizationID,
		ContactID:      row.ContactID,
		NIT:            row.Nit,
		DeliveryDays:   helpers.FromPgInt4Ptr(row.DeliveryDays),
		MinOrderAmount: helpers.FromPgNumeric(row.MinOrderAmount),
		Notes:          helpers.FromPgTextPtr(row.Notes),
		IsActive:       row.IsActive,
		CreatedAt:      row.CreatedAt.Time,
		UpdatedAt:      row.UpdatedAt.Time,
	}
}

func supplierParams(s *domain.Supplier, orgID int32) sqlc.CreateSupplierParams {
	return sqlc.CreateSupplierParams{
		OrganizationID: orgID,
		ContactID:      s.ContactID,
		Nit:            s.NIT,
		DeliveryDays:   helpers.ToPgInt4Ptr(s.DeliveryDays),
		MinOrderAmount: helpers.ToPgNumeric(s.MinOrderAmount),
		Notes:          helpers.ToPgTextPtr(s.Notes),
	}
}

type supplierRepository struct {
	store sqlc.Store
}

// NewSupplierRepository builds the supplier repository.
func NewSupplierRepository(store sqlc.Store) domain.SupplierRepository {
	return &supplierRepository{store: store}
}

// Create atomically creates the CRM contact (NIT + consent granted, D11),
// the supplier row, and the supplier_created + consent_grant audits. A
// duplicate (organization_id, nit) maps to domain.ErrSupplierAlreadyExists
// and rolls the whole transaction back (no orphan contact).
func (r *supplierRepository) Create(ctx context.Context, orgID int32, supplier *domain.Supplier, contact domain.ContactInput, memberID string) (*domain.Supplier, error) {
	var created *domain.Supplier
	err := inTx(ctx, r.store, func(s sqlc.Store) error {
		contactRow, err := s.CreateSupplierContact(ctx, sqlc.CreateSupplierContactParams{
			OrganizationID: orgID,
			PhoneNumber:    helpers.ToPgText(contact.PhoneNumber),
			DisplayName:    helpers.ToPgText(contact.DisplayName),
			NumeroDocumento: helpers.ToPgText(contact.NIT),
		})
		if err != nil {
			return err
		}

		supplier.ContactID = contactRow.ID
		row, err := s.CreateSupplier(ctx, supplierParams(supplier, orgID))
		if err != nil {
			return err
		}
		mapped := mapSupplier(row)
		created = mapped

		if _, err := s.InsertProcurementAudit(ctx, sqlc.InsertProcurementAuditParams{
			OrganizationID: orgID,
			EntityType:     "supplier",
			EntityID:       helpers.ToPgInt4Ptr(&mapped.ID),
			Action:         "supplier_created",
			Decision:       "allow",
			Reason:         pgtype.Text{},
			MemberID:       helpers.ToPgTextPtr(strPtr(memberID)),
			Metadata:       []byte(`{}`),
		}); err != nil {
			return err
		}
		if _, err := s.InsertProcurementAudit(ctx, sqlc.InsertProcurementAuditParams{
			OrganizationID: orgID,
			EntityType:     "contact",
			EntityID:       helpers.ToPgInt4Ptr(&contactRow.ID),
			Action:         "consent_grant",
			Decision:       "allow",
			Reason:         pgtype.Text{},
			MemberID:       helpers.ToPgTextPtr(strPtr(memberID)),
			Metadata:       []byte(`{"basis":"org_declared","tipo_documento":"NIT"}`),
		}); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.SQLState() == "23505" && pgErr.ConstraintName == "suppliers_org_nit_unique" {
			return nil, domain.ErrSupplierAlreadyExists
		}
		return nil, fmt.Errorf("create supplier: %w", err)
	}
	return created, nil
}

func (r *supplierRepository) GetByID(ctx context.Context, orgID, id int32) (*domain.Supplier, error) {
	row, err := r.store.GetSupplier(ctx, sqlc.GetSupplierParams{ID: id, OrganizationID: orgID})
	if err != nil {
		if isNoRows(err) {
			return nil, domain.ErrSupplierNotFound
		}
		return nil, err
	}
	return mapSupplier(row), nil
}

func (r *supplierRepository) GetByContactID(ctx context.Context, orgID, contactID int32) (*domain.Supplier, error) {
	row, err := r.store.GetSupplierByContact(ctx, sqlc.GetSupplierByContactParams{ContactID: contactID, OrganizationID: orgID})
	if err != nil {
		if isNoRows(err) {
			return nil, domain.ErrSupplierNotFound
		}
		return nil, err
	}
	return mapSupplier(row), nil
}

func (r *supplierRepository) List(ctx context.Context, orgID int32, limit, offset int32) ([]*domain.Supplier, error) {
	rows, err := r.store.ListSuppliersWithContact(ctx, sqlc.ListSuppliersWithContactParams{
		OrganizationID: orgID,
		Limit:          limit,
		Offset:         offset,
	})
	if err != nil {
		return nil, err
	}
	out := make([]*domain.Supplier, 0, len(rows))
	for i := range rows {
		base := sqlc.ProcurementSupplier{
			ID:             rows[i].ID,
			OrganizationID: rows[i].OrganizationID,
			ContactID:      rows[i].ContactID,
			Nit:            rows[i].Nit,
			DeliveryDays:   rows[i].DeliveryDays,
			MinOrderAmount: rows[i].MinOrderAmount,
			Notes:          rows[i].Notes,
			IsActive:       rows[i].IsActive,
			CreatedAt:      rows[i].CreatedAt,
			UpdatedAt:      rows[i].UpdatedAt,
		}
		sup := mapSupplier(base)
		sup.DisplayName = rows[i].DisplayName
		sup.PhoneNumber = rows[i].ContactPhone.String
		out = append(out, sup)
	}
	return out, nil
}

func (r *supplierRepository) ListActive(ctx context.Context, orgID int32) ([]*domain.Supplier, error) {
	rows, err := r.store.ListActiveSuppliersByOrganization(ctx, orgID)
	if err != nil {
		return nil, err
	}
	out := make([]*domain.Supplier, 0, len(rows))
	for i := range rows {
		out = append(out, mapSupplier(rows[i]))
	}
	return out, nil
}

func (r *supplierRepository) ListByIDs(ctx context.Context, orgID int32, ids []int32) ([]*domain.Supplier, error) {
	if len(ids) == 0 {
		return []*domain.Supplier{}, nil
	}
	rows, err := r.store.ListSuppliersByIDs(ctx, sqlc.ListSuppliersByIDsParams{
		OrganizationID: orgID,
		Column2:        ids,
	})
	if err != nil {
		return nil, err
	}
	out := make([]*domain.Supplier, 0, len(rows))
	for i := range rows {
		out = append(out, mapSupplier(rows[i]))
	}
	return out, nil
}

func (r *supplierRepository) Update(ctx context.Context, orgID int32, supplier *domain.Supplier) (*domain.Supplier, error) {
	row, err := r.store.UpdateSupplier(ctx, sqlc.UpdateSupplierParams{
		ID:             supplier.ID,
		OrganizationID: orgID,
		DeliveryDays:   helpers.ToPgInt4Ptr(supplier.DeliveryDays),
		MinOrderAmount: helpers.ToPgNumeric(supplier.MinOrderAmount),
		Notes:          helpers.ToPgTextPtr(supplier.Notes),
		IsActive:       supplier.IsActive,
	})
	if err != nil {
		if isNoRows(err) {
			return nil, domain.ErrSupplierNotFound
		}
		return nil, err
	}
	return mapSupplier(row), nil
}

func strPtr(s string) *string { return &s }
