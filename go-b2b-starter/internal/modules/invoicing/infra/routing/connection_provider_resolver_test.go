package routing

import (
	"context"
	"errors"
	"testing"

	"github.com/moasq/go-b2b-starter/internal/modules/invoicing/domain"
)

type stubResolverRepo struct {
	conns map[int32]*domain.OrgConnection
	err   error
}

func (s *stubResolverRepo) Get(ctx context.Context, orgID int32) (*domain.OrgConnection, error) {
	if s.err != nil {
		return nil, s.err
	}
	if c, ok := s.conns[orgID]; ok {
		return c, nil
	}
	return nil, domain.ErrConnectionNotFound
}

func (s *stubResolverRepo) Upsert(ctx context.Context, conn *domain.OrgConnection) (*domain.OrgConnection, error) {
	return nil, errors.New("not implemented")
}

func (s *stubResolverRepo) UpdateStatus(ctx context.Context, orgID int32, status domain.ConnectionStatus, lastError string) (*domain.OrgConnection, error) {
	return nil, errors.New("not implemented")
}

func (s *stubResolverRepo) UpdateCredentials(ctx context.Context, orgID int32, clientIDEnc, clientSecretEnc []byte, nit, companyName string) (*domain.OrgConnection, error) {
	return nil, errors.New("not implemented")
}

func (s *stubResolverRepo) Delete(ctx context.Context, orgID int32) error {
	return errors.New("not implemented")
}

func (s *stubResolverRepo) ListByStatus(ctx context.Context, provider string, status domain.ConnectionStatus) ([]*domain.OrgConnection, error) {
	return nil, errors.New("not implemented")
}

func (s *stubResolverRepo) ListAll(ctx context.Context) ([]*domain.OrgConnection, error) {
	return nil, errors.New("not implemented")
}

func TestConnectionProviderResolver_LiveRoutesToSiigo(t *testing.T) {
	repo := &stubResolverRepo{conns: map[int32]*domain.OrgConnection{
		1: {OrganizationID: 1, Provider: "siigo", Status: domain.ConnStatusLive},
	}}
	resolver := NewConnectionProviderResolver(repo)

	provider, err := resolver.GetInvoicingProvider(context.Background(), 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if provider != "siigo" {
		t.Fatalf("expected siigo for live org, got %q", provider)
	}
}

func TestConnectionProviderResolver_NonLiveRoutesToNone(t *testing.T) {
	cases := map[int32]domain.ConnectionStatus{
		1: domain.ConnStatusConnected,
		2: domain.ConnStatusNumeracionOK,
		3: domain.ConnStatusSandboxOK,
		4: domain.ConnStatusPaused,
		5: domain.ConnStatusInvoicingDisabled,
	}
	repo := &stubResolverRepo{conns: map[int32]*domain.OrgConnection{}}
	for id, status := range cases {
		repo.conns[id] = &domain.OrgConnection{OrganizationID: id, Provider: "siigo", Status: status}
	}
	resolver := NewConnectionProviderResolver(repo)

	for id := range cases {
		provider, err := resolver.GetInvoicingProvider(context.Background(), id)
		if err != nil {
			t.Fatalf("org %d: unexpected error: %v", id, err)
		}
		if provider != "none" {
			t.Fatalf("org %d (%s) must route to none, got %q", id, cases[id], provider)
		}
	}
}

func TestConnectionProviderResolver_MissingOrgRoutesToNone(t *testing.T) {
	resolver := NewConnectionProviderResolver(&stubResolverRepo{conns: map[int32]*domain.OrgConnection{}})
	provider, err := resolver.GetInvoicingProvider(context.Background(), 99)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if provider != "none" {
		t.Fatalf("expected none for missing org, got %q", provider)
	}
}

func TestConnectionProviderResolver_RepoErrorPropagates(t *testing.T) {
	resolver := NewConnectionProviderResolver(&stubResolverRepo{err: errors.New("db down")})
	if _, err := resolver.GetInvoicingProvider(context.Background(), 1); err == nil {
		t.Fatal("expected error from repo")
	}
}
