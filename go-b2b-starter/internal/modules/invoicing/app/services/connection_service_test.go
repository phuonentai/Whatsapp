package services

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/moasq/go-b2b-starter/internal/modules/invoicing/domain"
)

type mockConnectionRepo struct {
	conns map[int32]*domain.OrgConnection
	upserts []*domain.OrgConnection
	statusUpdates []struct{ org int32; status domain.ConnectionStatus }
}

func newMockConnectionRepo() *mockConnectionRepo {
	return &mockConnectionRepo{conns: map[int32]*domain.OrgConnection{}}
}

func (m *mockConnectionRepo) Get(ctx context.Context, orgID int32) (*domain.OrgConnection, error) {
	if c, ok := m.conns[orgID]; ok {
		return c, nil
	}
	return nil, domain.ErrConnectionNotFound
}

func (m *mockConnectionRepo) Upsert(ctx context.Context, conn *domain.OrgConnection) (*domain.OrgConnection, error) {
	m.upserts = append(m.upserts, conn)
	m.conns[conn.OrganizationID] = conn
	return conn, nil
}

func (m *mockConnectionRepo) UpdateStatus(ctx context.Context, orgID int32, status domain.ConnectionStatus, lastError string) (*domain.OrgConnection, error) {
	c, ok := m.conns[orgID]
	if !ok {
		return nil, domain.ErrConnectionNotFound
	}
	m.statusUpdates = append(m.statusUpdates, struct{ org int32; status domain.ConnectionStatus }{orgID, status})
	c.Status = status
	return c, nil
}

func (m *mockConnectionRepo) UpdateCredentials(ctx context.Context, orgID int32, clientIDEnc, clientSecretEnc []byte, nit, companyName string) (*domain.OrgConnection, error) {
	c, ok := m.conns[orgID]
	if !ok {
		return nil, domain.ErrConnectionNotFound
	}
	c.ClientIDEnc = clientIDEnc
	c.ClientSecretEnc = clientSecretEnc
	c.Nit = nit
	c.SiigoCompanyName = companyName
	return c, nil
}

func (m *mockConnectionRepo) Delete(ctx context.Context, orgID int32) error {
	delete(m.conns, orgID)
	return nil
}

func (m *mockConnectionRepo) ListByStatus(ctx context.Context, provider string, status domain.ConnectionStatus) ([]*domain.OrgConnection, error) {
	var out []*domain.OrgConnection
	for _, c := range m.conns {
		if c.Provider == provider && c.Status == status {
			out = append(out, c)
		}
	}
	return out, nil
}

type fakeValidator struct {
	company domain.ProviderCompany
	err     error
}

func (f *fakeValidator) ValidateCredentials(ctx context.Context, clientID, clientSecret string) (domain.ProviderCompany, error) {
	return f.company, f.err
}

type fakeCipher struct{}

func (fakeCipher) Encrypt(plaintext string) ([]byte, error) { return []byte("enc:" + plaintext), nil }
func (fakeCipher) Decrypt(blob []byte) (string, error) {
	return string(blob[len("enc:"):]), nil
}

func newTestConnService(validator domain.ConnectionValidator) (ConnectionService, *mockConnectionRepo) {
	repo := newMockConnectionRepo()
	svc := NewConnectionService(repo, validator, fakeCipher{}, nopLogger{})
	return svc, repo
}

func seedConnection(repo *mockConnectionRepo, orgID int32, status domain.ConnectionStatus) {
	repo.conns[orgID] = &domain.OrgConnection{
		OrganizationID: orgID,
		Provider:       "siigo",
		Status:         status,
	}
}

func TestConnect_SuccessValidatesAndStores(t *testing.T) {
	svc, _ := newTestConnService(&fakeValidator{company: domain.ProviderCompany{Nit: "900.123.456-7", Name: "Mi Empresa"}})

	conn, err := svc.Connect(context.Background(), 10, ConnectRequest{ClientID: "cid", ClientSecret: "csec", Nit: "9001234567"})
	if err != nil {
		t.Fatalf("Connect failed: %v", err)
	}
	if conn.Status != domain.ConnStatusConnected {
		t.Fatalf("expected connected, got %s", conn.Status)
	}
	if conn.Provider != "siigo" {
		t.Fatalf("expected siigo provider, got %s", conn.Provider)
	}
	if string(conn.ClientIDEnc) != "enc:cid" || string(conn.ClientSecretEnc) != "enc:csec" {
		t.Fatalf("credentials not encrypted: %+v", conn)
	}
	if conn.SiigoCompanyName != "Mi Empresa" {
		t.Fatalf("expected company name stored, got %q", conn.SiigoCompanyName)
	}
}

func TestConnect_NitMismatchRejectsWithoutPersistence(t *testing.T) {
	svc, repo := newTestConnService(&fakeValidator{company: domain.ProviderCompany{Nit: "900111222", Name: "Otra"}})

	_, err := svc.Connect(context.Background(), 10, ConnectRequest{ClientID: "cid", ClientSecret: "csec", Nit: "9001234567"})
	if !errors.Is(err, domain.ErrNitMismatch) {
		t.Fatalf("expected ErrNitMismatch, got %v", err)
	}
	if len(repo.upserts) != 0 {
		t.Fatal("connect must not persist on NIT mismatch")
	}
}

func TestConnect_InvalidCredentialsRejected(t *testing.T) {
	svc, repo := newTestConnService(&fakeValidator{err: fmt.Errorf("%w: bad creds", domain.ErrInvalidCredentials)})

	_, err := svc.Connect(context.Background(), 10, ConnectRequest{ClientID: "bad", ClientSecret: "bad", Nit: "9001234567"})
	if !errors.Is(err, domain.ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials, got %v", err)
	}
	if len(repo.upserts) != 0 {
		t.Fatal("connect must not persist on invalid credentials")
	}
}

func TestConnect_EmptyCredentialsRejected(t *testing.T) {
	svc, _ := newTestConnService(&fakeValidator{company: domain.ProviderCompany{Nit: "900123", Name: "X"}})
	if _, err := svc.Connect(context.Background(), 10, ConnectRequest{Nit: "900123"}); err == nil {
		t.Fatal("expected error for empty credentials")
	}
}

func TestConnect_InvalidTransitionFromConnected(t *testing.T) {
	svc, repo := newTestConnService(&fakeValidator{})
	seedConnection(repo, 10, domain.ConnStatusConnected)

	_, err := svc.Connect(context.Background(), 10, ConnectRequest{ClientID: "c", ClientSecret: "s", Nit: "1"})
	if !errors.Is(err, domain.ErrInvalidTransition) {
		t.Fatalf("expected ErrInvalidTransition, got %v", err)
	}
}

func TestStateMachine_GuardsTransitions(t *testing.T) {
	svc, repo := newTestConnService(&fakeValidator{})

	// Request assisted
	conn, err := svc.RequestAssisted(context.Background(), 1)
	if err != nil {
		t.Fatal(err)
	}
	if conn.Status != domain.ConnStatusAwaitingSetup {
		t.Fatalf("expected awaiting_setup, got %s", conn.Status)
	}
	// Duplicate request assisted from awaiting_setup must fail.
	if _, err := svc.RequestAssisted(context.Background(), 1); !errors.Is(err, domain.ErrInvalidTransition) {
		t.Fatalf("expected ErrInvalidTransition for duplicate assisted request, got %v", err)
	}

	// Provision advances awaiting_setup → connected.
	svc2 := NewConnectionService(repo, &fakeValidator{company: domain.ProviderCompany{Nit: "1", Name: "N"}}, fakeCipher{}, nopLogger{})
	if _, err := svc2.Provision(context.Background(), 1, ConnectRequest{ClientID: "c", ClientSecret: "s", Nit: "1"}); err != nil {
		t.Fatalf("provision failed: %v", err)
	}

	// Walk the machine through to live.
	seedConnection(repo, 1, domain.ConnStatusConnected)
	if _, err := svc2.Activate(context.Background(), 1); !errors.Is(err, domain.ErrInvalidTransition) {
		t.Fatalf("activate from connected must fail: %v", err)
	}
	if _, err := repo.UpdateStatus(context.Background(), 1, domain.ConnStatusNumeracionOK, ""); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.UpdateStatus(context.Background(), 1, domain.ConnStatusSandboxOK, ""); err != nil {
		t.Fatal(err)
	}
	if _, err := svc2.Activate(context.Background(), 1); err != nil {
		t.Fatalf("activate from sandbox_ok failed: %v", err)
	}

	// Pause/resume
	if _, err := svc2.Pause(context.Background(), 1); err != nil {
		t.Fatalf("pause failed: %v", err)
	}
	live, err := svc2.IsLive(context.Background(), 1)
	if err != nil || live {
		t.Fatalf("paused must not be live: %v %v", live, err)
	}
	if _, err := svc2.Pause(context.Background(), 1); !errors.Is(err, domain.ErrInvalidTransition) {
		t.Fatalf("double pause must fail: %v", err)
	}
	if _, err := svc2.Resume(context.Background(), 1); err != nil {
		t.Fatalf("resume failed: %v", err)
	}
	live, err = svc2.IsLive(context.Background(), 1)
	if err != nil || !live {
		t.Fatalf("resumed must be live: %v %v", live, err)
	}

	// Disable is reachable from anywhere.
	if _, err := svc2.Disable(context.Background(), 1); err != nil {
		t.Fatalf("disable failed: %v", err)
	}
	if _, err := svc2.Resume(context.Background(), 1); !errors.Is(err, domain.ErrInvalidTransition) {
		t.Fatalf("resume from disabled must fail: %v", err)
	}
}

func TestStatus_ImplicitNone(t *testing.T) {
	svc, _ := newTestConnService(&fakeValidator{})
	conn, err := svc.Status(context.Background(), 99)
	if err != nil {
		t.Fatal(err)
	}
	if conn.Status != domain.ConnStatusNone || conn.Provider != "none" {
		t.Fatalf("expected implicit none, got %+v", conn)
	}
}

func TestIsLive_MissingOrgIsNotLive(t *testing.T) {
	svc, _ := newTestConnService(&fakeValidator{})
	live, err := svc.IsLive(context.Background(), 123)
	if err != nil || live {
		t.Fatalf("missing org must not be live: %v %v", live, err)
	}
}
