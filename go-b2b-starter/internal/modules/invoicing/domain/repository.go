package domain

import "context"

// InvoiceRepository is the local system-of-record access for invoices.
type InvoiceRepository interface {
	GetByDeal(ctx context.Context, orgID, dealID int32) (*Invoice, error)
	GetByExternalID(ctx context.Context, externalID string) (*Invoice, error)
	Insert(ctx context.Context, inv *Invoice) (*Invoice, error)
	UpdateStatus(ctx context.Context, id int64, status InvoiceStatus, cufe, pdfURL string) (*Invoice, error)
	MarkNotified(ctx context.Context, id int64, status InvoiceStatus) (*Invoice, error)
	ListByStatus(ctx context.Context, status InvoiceStatus, limit int32) ([]*Invoice, error)
}
