package domain

import (
	"context"
	"errors"
)

var (
	ErrInvoiceNotFound = errors.New("invoice not found")
	ErrInvoiceExists   = errors.New("invoice already exists for this deal")
	ErrProvider        = errors.New("invoicing provider error")
)

// InvoicingProvider is the provider-agnostic rail for electronic invoices.
// Implemented by adapters in infra (Siigo first, Alegra later). Domain MUST NOT
// import provider SDKs or transport packages.
type InvoicingProvider interface {
	CreateInvoice(ctx context.Context, orgID int32, req *InvoiceRequest) (*Invoice, error)
	GetInvoiceStatus(ctx context.Context, orgID int32, externalID string) (*Invoice, error)
	UpsertCustomer(ctx context.Context, orgID int32, customer CustomerInfo) (*CustomerRef, error)
}
