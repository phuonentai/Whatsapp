// Package domain holds the transport-free contracts for the client-payments capability.
package domain

import (
	"errors"
	"time"
)

// PaymentStatus tracks the lifecycle of a client payment.
type PaymentStatus string

const (
	PaymentStatusPending PaymentStatus = "pending"
	PaymentStatusPaid    PaymentStatus = "paid"
	PaymentStatusFailed  PaymentStatus = "failed"
	PaymentStatusExpired PaymentStatus = "expired"
)

// IsTerminal reports whether the status is immutable.
func (s PaymentStatus) IsTerminal() bool {
	return s != PaymentStatusPending
}

// ClientPayment is the local system-of-record row for a one-shot, customer
// payment made to the SME. Only MercadoPago identifiers are stored — never
// tokens, card data, or wallet credentials.
type ClientPayment struct {
	ID              int64         `json:"id"`
	OrganizationID  int32         `json:"organization_id"`
	DealID          int32         `json:"deal_id"`
	InvoiceID       *int32        `json:"invoice_id,omitempty"`
	AmountCOP       int64         `json:"amount_cop"`
	CommissionCOP   int64         `json:"commission_cop"`
	Currency        string        `json:"currency"`
	Status          PaymentStatus `json:"status"`
	MPPreferenceID  string        `json:"mp_preference_id,omitempty"`
	MPPaymentID     string        `json:"mp_payment_id,omitempty"`
	PaidAt          *time.Time    `json:"paid_at,omitempty"`
	CreatedAt       time.Time     `json:"created_at"`
}

var (
	ErrPaymentNotFound = errors.New("client payment not found")
	ErrPaymentTerminal = errors.New("client payment is in a terminal state")
)
