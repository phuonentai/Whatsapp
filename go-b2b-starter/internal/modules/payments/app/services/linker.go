package services

import (
	"context"

	invoicingServices "github.com/moasq/go-b2b-starter/internal/modules/invoicing/app/services"
)

// invoicingPaymentLinker adapts PaymentsService to the invoicing PaymentLinker
// seam so the invoice WhatsApp notification carries a real, tracked link.
// Link creation failures are returned; invoicing treats them as non-fatal.
type invoicingPaymentLinker struct {
	svc PaymentsService
}

// NewInvoicingPaymentLinker provides the real PaymentLinker implementation
// for the invoicing module (replaces the noop seam).
func NewInvoicingPaymentLinker(svc PaymentsService) invoicingServices.PaymentLinker {
	return &invoicingPaymentLinker{svc: svc}
}

func (l *invoicingPaymentLinker) PaymentLink(ctx context.Context, orgID, dealID int32, amountCOP int64) (string, error) {
	return l.svc.CreateLink(ctx, orgID, dealID, nil, amountCOP)
}
