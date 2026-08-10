package domain

import (
	"context"
	"time"
)

// NumerationMode tells whether the provider auto-assigns consecutive numbers
// ("auto") or requires the platform to supply the next number ("manual").
type NumerationMode string

const (
	NumerationAuto   NumerationMode = "auto"
	NumerationManual NumerationMode = "manual"
)

// NumerationInfo is the live numeration read from the provider. With Siigo
// (verified by spike: no numeration resource exposed; invoices number
// automatically), auto mode carries empty resolution fields.
type NumerationInfo struct {
	Mode         NumerationMode
	ResolutionID string
	Prefix       string
	NextNumber   string
}

// NumerationSnapshot is the human-confirmed numeration stored per org.
type NumerationSnapshot struct {
	OrganizationID int32
	Mode           NumerationMode
	ResolutionID   string
	Prefix         string
	NextNumber     string
	ConfirmedAt    *time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// NumerationReader is implemented by the provider adapter.
type NumerationReader interface {
	GetNumeration(ctx context.Context, orgID int32) (NumerationInfo, error)
}

// NumerationRepository is the local access surface for numeration snapshots.
type NumerationRepository interface {
	Get(ctx context.Context, orgID int32) (*NumerationSnapshot, error)
	UpsertConfirmed(ctx context.Context, snapshot *NumerationSnapshot) (*NumerationSnapshot, error)
}

// CustomerRecord is one provider customer as seen by the import flow.
type CustomerRecord struct {
	ExternalID         string
	Name               string
	Identification     string
	IdentificationType string
	Email              string
	Phone              string
}

// CustomerReader is implemented by the provider adapter.
type CustomerReader interface {
	ListCustomers(ctx context.Context, orgID int32, page int32) ([]CustomerRecord, error)
}

// ImportRunKind classifies import executions.
type ImportRunKind string

const (
	ImportRunConfirm ImportRunKind = "confirm"
	ImportRunDelta   ImportRunKind = "delta"
)

// ImportRun records one executed import (confirm or delta; previews never
// write anything, including runs).
type ImportRun struct {
	ID             int64
	OrganizationID int32
	Kind           ImportRunKind
	Counts         map[string]int32
	Error          string
	PulledAt       time.Time
}

// ImportRunRepository records and lists import executions.
type ImportRunRepository interface {
	Record(ctx context.Context, run *ImportRun) (*ImportRun, error)
	ListByOrg(ctx context.Context, orgID int32, limit int32) ([]*ImportRun, error)
}
