// Package repositories implements the invoicing domain repository over SQLC.
package repositories

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/moasq/go-b2b-starter/internal/db/helpers"
	sqlc "github.com/moasq/go-b2b-starter/internal/db/postgres/sqlc/gen"
	"github.com/moasq/go-b2b-starter/internal/modules/invoicing/domain"
)

type invoiceRepository struct{ store sqlc.Store }

func NewInvoiceRepository(store sqlc.Store) domain.InvoiceRepository {
	return &invoiceRepository{store: store}
}

func (r *invoiceRepository) GetByDeal(ctx context.Context, orgID, dealID int32) (*domain.Invoice, error) {
	row, err := r.store.GetInvoiceByDeal(ctx, sqlc.GetInvoiceByDealParams{
		OrganizationID: orgID, DealID: dealID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrInvoiceNotFound
		}
		return nil, fmt.Errorf("failed to get invoice by deal: %w", err)
	}
	return mapInvoice(&row), nil
}

func (r *invoiceRepository) GetByExternalID(ctx context.Context, externalID string) (*domain.Invoice, error) {
	row, err := r.store.GetInvoiceByExternalIDAny(ctx, helpers.ToPgText(externalID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrInvoiceNotFound
		}
		return nil, fmt.Errorf("failed to get invoice by external id: %w", err)
	}
	return mapInvoice(&row), nil
}

func (r *invoiceRepository) Insert(ctx context.Context, inv *domain.Invoice) (*domain.Invoice, error) {
	row, err := r.store.InsertInvoice(ctx, sqlc.InsertInvoiceParams{
		OrganizationID: inv.OrganizationID,
		DealID:         inv.DealID,
		ExternalID:     helpers.ToPgText(inv.ExternalID),
		Cufe:           helpers.ToPgText(inv.Cufe),
		Status:         string(inv.Status),
		PdfUrl:         helpers.ToPgText(inv.PdfURL),
		Amount:         helpers.ToPgNumeric(inv.Amount),
		Currency:       inv.Currency,
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" { // unique_violation
			return nil, domain.ErrInvoiceExists
		}
		return nil, fmt.Errorf("failed to insert invoice: %w", err)
	}
	return mapInvoice(&row), nil
}

func (r *invoiceRepository) UpdateStatus(ctx context.Context, id int64, status domain.InvoiceStatus, cufe, pdfURL string) (*domain.Invoice, error) {
	row, err := r.store.UpdateInvoiceStatusByID(ctx, sqlc.UpdateInvoiceStatusByIDParams{
		ID:     id,
		Status: string(status),
		Cufe:   helpers.ToPgText(cufe),
		PdfUrl: helpers.ToPgText(pdfURL),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrInvoiceNotFound
		}
		return nil, fmt.Errorf("failed to update invoice status: %w", err)
	}
	return mapInvoice(&row), nil
}

func (r *invoiceRepository) MarkNotified(ctx context.Context, id int64, status domain.InvoiceStatus) (*domain.Invoice, error) {
	row, err := r.store.UpdateInvoiceNotifiedStatus(ctx, sqlc.UpdateInvoiceNotifiedStatusParams{
		ID: id, NotifiedStatus: helpers.ToPgText(string(status)),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrInvoiceNotFound
		}
		return nil, fmt.Errorf("failed to mark invoice notified: %w", err)
	}
	return mapInvoice(&row), nil
}

func (r *invoiceRepository) ListByStatus(ctx context.Context, status domain.InvoiceStatus, limit int32) ([]*domain.Invoice, error) {
	rows, err := r.store.ListInvoicesByStatus(ctx, sqlc.ListInvoicesByStatusParams{
		Status: string(status), Limit: limit,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list invoices by status: %w", err)
	}
	invoices := make([]*domain.Invoice, len(rows))
	for i := range rows {
		invoices[i] = mapInvoice(&rows[i])
	}
	return invoices, nil
}

func mapInvoice(row *sqlc.InvoicingInvoice) *domain.Invoice {
	inv := &domain.Invoice{
		ID:             row.ID,
		OrganizationID: row.OrganizationID,
		DealID:         row.DealID,
		ExternalID:     helpers.FromPgText(row.ExternalID),
		Cufe:           helpers.FromPgText(row.Cufe),
		Status:         domain.InvoiceStatus(row.Status),
		PdfURL:         helpers.FromPgText(row.PdfUrl),
		Amount:         helpers.FromPgNumeric(row.Amount),
		Currency:       row.Currency,
		CreatedAt:      row.CreatedAt.Time,
		UpdatedAt:      row.UpdatedAt.Time,
	}
	if row.NotifiedStatus.Valid {
		inv.NotifiedStatus = domain.InvoiceStatus(row.NotifiedStatus.String)
	}
	return inv
}
