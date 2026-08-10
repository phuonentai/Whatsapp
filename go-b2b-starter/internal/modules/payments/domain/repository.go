package domain

import (
	"context"
	"time"
)

// PaymentRepository is the local system-of-record access for client payments.
type PaymentRepository interface {
	Create(ctx context.Context, p *ClientPayment) (*ClientPayment, error)
	GetByPreferenceID(ctx context.Context, preferenceID string) (*ClientPayment, error)
	GetByPaymentID(ctx context.Context, paymentID string) (*ClientPayment, error)
	// AttachPaymentID links a provider payment id to a pending row (guarded:
	// only 'pending' rows accept the link). Returns ErrPaymentTerminal when
	// the row is already terminal.
	AttachPaymentID(ctx context.Context, id int64, mpPaymentID string) (*ClientPayment, error)
	// Transition applies a guarded status change: only 'pending' rows move;
	// terminal states are never mutated. Returns ErrPaymentTerminal when the
	// row is already terminal (idempotent no-op for the caller).
	Transition(ctx context.Context, id int64, status PaymentStatus, mpPaymentID string, paidAt *time.Time) (*ClientPayment, error)
}
