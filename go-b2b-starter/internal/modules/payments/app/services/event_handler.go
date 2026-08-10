package services

import "context"

// PaymentEventHandler is the seam consumed by the billing webhook service to
// dispatch MercadoPago payment events to the client-payments module. Billing
// imports only this interface — the payments module never imports billing.
type PaymentEventHandler interface {
	HandlePaymentEvent(ctx context.Context, eventType, paymentID string) error
}
