// Package repositories implements the payments domain repository over SQLC.
package repositories

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/moasq/go-b2b-starter/internal/db/helpers"
	sqlc "github.com/moasq/go-b2b-starter/internal/db/postgres/sqlc/gen"
	"github.com/moasq/go-b2b-starter/internal/modules/payments/domain"
)

type paymentRepository struct{ store sqlc.Store }

func NewPaymentRepository(store sqlc.Store) domain.PaymentRepository {
	return &paymentRepository{store: store}
}

func (r *paymentRepository) Create(ctx context.Context, p *domain.ClientPayment) (*domain.ClientPayment, error) {
	row, err := r.store.CreateClientPayment(ctx, sqlc.CreateClientPaymentParams{
		OrganizationID: p.OrganizationID,
		DealID:         p.DealID,
		InvoiceID:      helpers.ToPgInt4Ptr(p.InvoiceID),
		AmountCop:      p.AmountCOP,
		CommissionCop:  p.CommissionCOP,
		MpPreferenceID: helpers.ToPgText(p.MPPreferenceID),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to insert client payment: %w", err)
	}
	return mapPayment(&row), nil
}

func (r *paymentRepository) GetByPreferenceID(ctx context.Context, preferenceID string) (*domain.ClientPayment, error) {
	row, err := r.store.GetClientPaymentByPreferenceID(ctx, helpers.ToPgText(preferenceID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrPaymentNotFound
		}
		return nil, fmt.Errorf("failed to get client payment by preference: %w", err)
	}
	return mapPayment(&row), nil
}

func (r *paymentRepository) GetByPaymentID(ctx context.Context, paymentID string) (*domain.ClientPayment, error) {
	row, err := r.store.GetClientPaymentByPaymentID(ctx, helpers.ToPgText(paymentID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrPaymentNotFound
		}
		return nil, fmt.Errorf("failed to get client payment by payment id: %w", err)
	}
	return mapPayment(&row), nil
}

func (r *paymentRepository) AttachPaymentID(ctx context.Context, id int64, mpPaymentID string) (*domain.ClientPayment, error) {
	row, err := r.store.AttachPaymentID(ctx, sqlc.AttachPaymentIDParams{ID: id, MpPaymentID: helpers.ToPgText(mpPaymentID)})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrPaymentTerminal
		}
		return nil, fmt.Errorf("failed to attach payment id: %w", err)
	}
	return mapPayment(&row), nil
}

func (r *paymentRepository) Transition(ctx context.Context, id int64, status domain.PaymentStatus, mpPaymentID string, paidAt *time.Time) (*domain.ClientPayment, error) {
	row, err := r.store.UpdatePaymentStatus(ctx, sqlc.UpdatePaymentStatusParams{
		ID:           id,
		Status:       string(status),
		MpPaymentID:  helpers.ToPgText(mpPaymentID),
		PaidAt:       helpers.ToPgTimestamptzPtr(paidAt),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrPaymentTerminal
		}
		return nil, fmt.Errorf("failed to transition client payment: %w", err)
	}
	return mapPayment(&row), nil
}

func mapPayment(row *sqlc.PaymentsClientPayment) *domain.ClientPayment {
	p := &domain.ClientPayment{
		ID:             row.ID,
		OrganizationID: row.OrganizationID,
		DealID:         row.DealID,
		AmountCOP:      row.AmountCop,
		CommissionCOP:  row.CommissionCop,
		Currency:       row.Currency,
		Status:         domain.PaymentStatus(row.Status),
		MPPreferenceID: row.MpPreferenceID.String,
		MPPaymentID:    row.MpPaymentID.String,
		CreatedAt:      row.CreatedAt.Time,
	}
	if row.InvoiceID.Valid {
		v := row.InvoiceID.Int32
		p.InvoiceID = &v
	}
	if row.PaidAt.Valid {
		t := row.PaidAt.Time
		p.PaidAt = &t
	}
	return p
}
