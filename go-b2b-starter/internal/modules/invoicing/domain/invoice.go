// Package domain holds the transport-free contracts for the invoicing capability.
package domain

import "time"

type InvoiceStatus string

const (
	InvoiceStatusPending InvoiceStatus = "pending"
	InvoiceStatusValid   InvoiceStatus = "valid"
	InvoiceStatusInvalid InvoiceStatus = "invalid"
	InvoiceStatusErrored InvoiceStatus = "errored"
)

func (s InvoiceStatus) IsFinal() bool {
	return s == InvoiceStatusValid || s == InvoiceStatusInvalid || s == InvoiceStatusErrored
}

// Invoice is the local system-of-record row for an electronic invoice.
// The invoicing provider (Siigo) is a rail; the local DB is the record.
type Invoice struct {
	ID              int64         `json:"id"`
	OrganizationID  int32         `json:"organization_id"`
	DealID          int32         `json:"deal_id"`
	ExternalID      string        `json:"external_id,omitempty"`
	Cufe            string        `json:"cufe,omitempty"`
	Status          InvoiceStatus `json:"status"`
	PdfURL          string        `json:"pdf_url,omitempty"`
	Amount          *float64      `json:"amount,omitempty"`
	Currency        string        `json:"currency"`
	NotifiedStatus  InvoiceStatus `json:"notified_status,omitempty"`
	CreatedAt       time.Time     `json:"created_at"`
	UpdatedAt       time.Time     `json:"updated_at"`
}

// CustomerInfo is the minimal customer projection needed to invoice a sale.
type CustomerInfo struct {
	Name             string
	Identification   string // NIT, CC, CE, TI, PP per contact/company data
	IdentificationType string // "NIT" | "CC" | ...
	Email            string
	Phone            string
}

// CustomerRef is the provider-side identifier for an upserted customer.
type CustomerRef struct {
	ExternalID string
	Name       string
}

// InvoiceRequest carries the sale details for creating an invoice at the provider.
type InvoiceRequest struct {
	OrganizationID int32
	DealID         int32
	Customer       CustomerInfo
	Amount         *float64
	Currency       string
	Description    string
}
