package domain

import (
	"context"
	"errors"
	"time"
)

var (
	// ErrConnectionNotFound is returned when no connection row exists.
	ErrConnectionNotFound = errors.New("connection not found")
	// ErrInvalidTransition is returned when a state-machine transition is
	// not allowed from the current state.
	ErrInvalidTransition = errors.New("invalid connection state transition")
	// ErrNitMismatch is returned when the provider company NIT does not
	// match the organization's NIT.
	ErrNitMismatch = errors.New("nit mismatch between provider company and organization")
	// ErrInvalidCredentials is returned when provider credential validation
	// fails during connect.
	ErrInvalidCredentials = errors.New("invalid provider credentials")
	// ErrCredentialResolution is returned when stored credentials cannot be
	// resolved or decrypted.
	ErrCredentialResolution = errors.New("failed to resolve provider credentials")
)

// ConnectionStatus is the onboarding state of an organization's invoicing
// connection. Transitions are guarded by the application service; the DB
// CHECK constraint only bounds the allowed values.
type ConnectionStatus string

const (
	ConnStatusNone             ConnectionStatus = "none"
	ConnStatusAwaitingSetup    ConnectionStatus = "awaiting_setup"
	ConnStatusConnected        ConnectionStatus = "connected"
	ConnStatusNumeracionOK     ConnectionStatus = "numeracion_ok"
	ConnStatusSandboxOK        ConnectionStatus = "sandbox_ok"
	ConnStatusLive             ConnectionStatus = "live"
	ConnStatusPaused           ConnectionStatus = "paused"
	ConnStatusInvoicingDisabled ConnectionStatus = "invoicing_disabled"
)

// IsLive reports whether deal-stage invoicing may run for this status.
func (s ConnectionStatus) IsLive() bool { return s == ConnStatusLive }

// OrgConnection is the per-organization invoicing connection row. Credential
// columns hold ciphertext only (AES-256-GCM); they are never exposed.
type OrgConnection struct {
	OrganizationID   int32
	Provider         string
	Status           ConnectionStatus
	ClientIDEnc      []byte
	ClientSecretEnc  []byte
	Nit              string
	SiigoCompanyName string
	LastError        string
	PausedAt         *time.Time
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// HasCredentials reports whether encrypted credentials are present.
func (c *OrgConnection) HasCredentials() bool {
	return len(c.ClientIDEnc) > 0 && len(c.ClientSecretEnc) > 0
}

// ConnectionRepository is the local access surface for org connections.
type ConnectionRepository interface {
	Get(ctx context.Context, orgID int32) (*OrgConnection, error)
	Upsert(ctx context.Context, conn *OrgConnection) (*OrgConnection, error)
	UpdateStatus(ctx context.Context, orgID int32, status ConnectionStatus, lastError string) (*OrgConnection, error)
	UpdateCredentials(ctx context.Context, orgID int32, clientIDEnc, clientSecretEnc []byte, nit, companyName string) (*OrgConnection, error)
	Delete(ctx context.Context, orgID int32) error
	ListByStatus(ctx context.Context, provider string, status ConnectionStatus) ([]*OrgConnection, error)
	ListAll(ctx context.Context) ([]*OrgConnection, error)
}

// CredentialProvider supplies plaintext Siigo credentials for an organization
// at call time. Implemented by the infra glue (repository + envelope decrypt);
// domain never touches ciphertext or keys directly.
type CredentialProvider interface {
	ResolveCredentials(ctx context.Context, orgID int32) (clientID, clientSecret string, err error)
}

// ProviderCompany is the minimal company projection returned by provider
// credential validation (connect step).
type ProviderCompany struct {
	Nit  string
	Name string
}

// ConnectionValidator verifies a raw credential pair against the provider
// before anything is persisted. Implemented by the adapter.
type ConnectionValidator interface {
	ValidateCredentials(ctx context.Context, clientID, clientSecret string) (ProviderCompany, error)
}
