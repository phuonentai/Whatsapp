package services

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/moasq/go-b2b-starter/internal/modules/invoicing/domain"
	loggerDomain "github.com/moasq/go-b2b-starter/internal/platform/logger/domain"
)

// ConnectRequest carries the raw credentials + the organization's own NIT for
// the connect/provision flows.
type ConnectRequest struct {
	ClientID     string
	ClientSecret string
	Nit          string
}

// CredentialCipher encrypts/decrypts credentials at rest. Implemented by the
// envelope-encryption infrastructure; the service depends on the behaviour,
// not the implementation.
type CredentialCipher interface {
	Encrypt(plaintext string) ([]byte, error)
	Decrypt(blob []byte) (string, error)
}

// ConnectionService owns the per-organization invoicing connection state
// machine and the connect flow. Every state change goes through the guarded
// transition table; there is no free-form status update.
type ConnectionService interface {
	Status(ctx context.Context, orgID int32) (*domain.OrgConnection, error)
	Connect(ctx context.Context, orgID int32, req ConnectRequest) (*domain.OrgConnection, error)
	RequestAssisted(ctx context.Context, orgID int32) (*domain.OrgConnection, error)
	Provision(ctx context.Context, orgID int32, req ConnectRequest) (*domain.OrgConnection, error)
	Pause(ctx context.Context, orgID int32) (*domain.OrgConnection, error)
	Resume(ctx context.Context, orgID int32) (*domain.OrgConnection, error)
	Activate(ctx context.Context, orgID int32) (*domain.OrgConnection, error)
	Disable(ctx context.Context, orgID int32) (*domain.OrgConnection, error)
	ConfirmNumeration(ctx context.Context, orgID int32) (*domain.OrgConnection, error)
	ConfirmSandboxOK(ctx context.Context, orgID int32) (*domain.OrgConnection, error)
	IsLive(ctx context.Context, orgID int32) (bool, error)
}

type connectionAction string

const (
	actionSelfConnect       connectionAction = "self_connect"
	actionRequestAssisted   connectionAction = "request_assisted"
	actionAssistedProvision connectionAction = "assisted_provision"
	actionConfirmNumeration connectionAction = "confirm_numeration"
	actionSandboxOK         connectionAction = "sandbox_ok"
	actionActivate          connectionAction = "activate"
	actionPause             connectionAction = "pause"
	actionResume            connectionAction = "resume"
	actionDisable           connectionAction = "disable"
)

type connectionService struct {
	repo      domain.ConnectionRepository
	validator domain.ConnectionValidator
	cipher    CredentialCipher
	logger    loggerDomain.Logger
}

func NewConnectionService(
	repo domain.ConnectionRepository,
	validator domain.ConnectionValidator,
	cipher CredentialCipher,
	logger loggerDomain.Logger,
) ConnectionService {
	return &connectionService{repo: repo, validator: validator, cipher: cipher, logger: logger}
}

// nextState is the guarded transition table. Unknown (from, action) pairs are
// rejected; the DB CHECK constraint only bounds the allowed status values.
func nextState(current domain.ConnectionStatus, action connectionAction) (domain.ConnectionStatus, error) {
	switch action {
	case actionSelfConnect, actionAssistedProvision:
		if current == domain.ConnStatusNone || current == domain.ConnStatusAwaitingSetup {
			return domain.ConnStatusConnected, nil
		}
	case actionRequestAssisted:
		if current == domain.ConnStatusNone {
			return domain.ConnStatusAwaitingSetup, nil
		}
	case actionConfirmNumeration:
		if current == domain.ConnStatusConnected {
			return domain.ConnStatusNumeracionOK, nil
		}
	case actionSandboxOK:
		if current == domain.ConnStatusNumeracionOK {
			return domain.ConnStatusSandboxOK, nil
		}
	case actionActivate:
		if current == domain.ConnStatusSandboxOK {
			return domain.ConnStatusLive, nil
		}
	case actionPause:
		if current == domain.ConnStatusLive {
			return domain.ConnStatusPaused, nil
		}
	case actionResume:
		if current == domain.ConnStatusPaused {
			return domain.ConnStatusLive, nil
		}
	case actionDisable:
		// Terminal state reachable from anywhere.
		return domain.ConnStatusInvoicingDisabled, nil
	}
	return "", fmt.Errorf("%w: %s from %s", domain.ErrInvalidTransition, action, current)
}

func (s *connectionService) Status(ctx context.Context, orgID int32) (*domain.OrgConnection, error) {
	conn, err := s.repo.Get(ctx, orgID)
	if err != nil {
		if errors.Is(err, domain.ErrConnectionNotFound) {
			// Every org has a status: the implicit none state.
			return &domain.OrgConnection{
				OrganizationID: orgID,
				Provider:       "none",
				Status:         domain.ConnStatusNone,
			}, nil
		}
		return nil, err
	}
	return conn, nil
}

func (s *connectionService) IsLive(ctx context.Context, orgID int32) (bool, error) {
	conn, err := s.repo.Get(ctx, orgID)
	if err != nil {
		if errors.Is(err, domain.ErrConnectionNotFound) {
			return false, nil
		}
		return false, err
	}
	return conn.Status.IsLive(), nil
}

// Connect validates the credential pair against the provider, verifies the
// provider company NIT against the organization's NIT, then persists the
// encrypted credentials and advances the state. Nothing is persisted on
// validation failure.
func (s *connectionService) Connect(ctx context.Context, orgID int32, req ConnectRequest) (*domain.OrgConnection, error) {
	cur, err := s.currentOrSynthetic(ctx, orgID)
	if err != nil {
		return nil, err
	}
	next, err := nextState(cur.Status, actionSelfConnect)
	if err != nil {
		return nil, err
	}
	return s.connectTo(ctx, orgID, cur, req, next)
}

func (s *connectionService) RequestAssisted(ctx context.Context, orgID int32) (*domain.OrgConnection, error) {
	cur, err := s.currentOrSynthetic(ctx, orgID)
	if err != nil {
		return nil, err
	}
	next, err := nextState(cur.Status, actionRequestAssisted)
	if err != nil {
		return nil, err
	}
	return s.repo.Upsert(ctx, &domain.OrgConnection{
		OrganizationID: orgID,
		Provider:       "none",
		Status:         next,
	})
}

func (s *connectionService) Provision(ctx context.Context, orgID int32, req ConnectRequest) (*domain.OrgConnection, error) {
	cur, err := s.currentOrSynthetic(ctx, orgID)
	if err != nil {
		return nil, err
	}
	next, err := nextState(cur.Status, actionAssistedProvision)
	if err != nil {
		return nil, err
	}
	// Same validation path as self-serve connect; admin-scoping happens at
	// the HTTP layer.
	return s.connectTo(ctx, orgID, cur, req, next)
}

func (s *connectionService) connectTo(ctx context.Context, orgID int32, cur *domain.OrgConnection, req ConnectRequest, next domain.ConnectionStatus) (*domain.OrgConnection, error) {
	if req.ClientID == "" || req.ClientSecret == "" {
		return nil, fmt.Errorf("%w: credentials are required", domain.ErrInvalidCredentials)
	}

	company, err := s.validator.ValidateCredentials(ctx, req.ClientID, req.ClientSecret)
	if err != nil {
		return nil, err
	}

	// Spike finding: Siigo exposes no company endpoint, so the provider may
	// return no NIT. When provider data is absent, the client-declared NIT is
	// stored as-is (wizard shows it for confirmation); when provider data IS
	// available, a mismatch is a hard failure.
	if company.Nit != "" && (normalizeNit(req.Nit) == "" || normalizeNit(req.Nit) != normalizeNit(company.Nit)) {
		return nil, domain.ErrNitMismatch
	}

	clientIDEnc, err := s.cipher.Encrypt(req.ClientID)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt client id: %w", err)
	}
	clientSecretEnc, err := s.cipher.Encrypt(req.ClientSecret)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt client secret: %w", err)
	}

	if _, err := s.repo.Upsert(ctx, &domain.OrgConnection{
		OrganizationID: orgID,
		Provider:       "siigo",
		Status:         next,
	}); err != nil {
		return nil, err
	}
	return s.repo.UpdateCredentials(ctx, orgID, clientIDEnc, clientSecretEnc, req.Nit, company.Name)
}

func (s *connectionService) Pause(ctx context.Context, orgID int32) (*domain.OrgConnection, error) {
	return s.transition(ctx, orgID, actionPause)
}

func (s *connectionService) Resume(ctx context.Context, orgID int32) (*domain.OrgConnection, error) {
	return s.transition(ctx, orgID, actionResume)
}

func (s *connectionService) Activate(ctx context.Context, orgID int32) (*domain.OrgConnection, error) {
	return s.transition(ctx, orgID, actionActivate)
}

func (s *connectionService) ConfirmNumeration(ctx context.Context, orgID int32) (*domain.OrgConnection, error) {
	return s.transition(ctx, orgID, actionConfirmNumeration)
}

func (s *connectionService) ConfirmSandboxOK(ctx context.Context, orgID int32) (*domain.OrgConnection, error) {
	return s.transition(ctx, orgID, actionSandboxOK)
}

func (s *connectionService) Disable(ctx context.Context, orgID int32) (*domain.OrgConnection, error) {
	return s.transition(ctx, orgID, actionDisable)
}

func (s *connectionService) transition(ctx context.Context, orgID int32, action connectionAction) (*domain.OrgConnection, error) {
	cur, err := s.currentOrSynthetic(ctx, orgID)
	if err != nil {
		return nil, err
	}
	next, err := nextState(cur.Status, action)
	if err != nil {
		return nil, err
	}
	return s.repo.UpdateStatus(ctx, orgID, next, "")
}

func (s *connectionService) currentOrSynthetic(ctx context.Context, orgID int32) (*domain.OrgConnection, error) {
	conn, err := s.repo.Get(ctx, orgID)
	if err != nil {
		if errors.Is(err, domain.ErrConnectionNotFound) {
			return &domain.OrgConnection{
				OrganizationID: orgID,
				Provider:       "none",
				Status:         domain.ConnStatusNone,
			}, nil
		}
		return nil, err
	}
	return conn, nil
}

// normalizeNit strips punctuation/spacing for comparison (e.g. "900.123.456-7"
// vs "9001234567"). Digits only, or empty when no digits present.
func normalizeNit(nit string) string {
	var b strings.Builder
	for _, r := range nit {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}
