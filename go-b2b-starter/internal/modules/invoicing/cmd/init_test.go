package cmd

import (
	"context"
	"errors"
	"testing"

	"github.com/moasq/go-b2b-starter/internal/modules/invoicing/app/services"
	"github.com/moasq/go-b2b-starter/internal/modules/invoicing/domain"
	loggerDomain "github.com/moasq/go-b2b-starter/internal/platform/logger/domain"
)

type stubDeltaImportSvc struct {
	calls     []int32
	failOnOrg int32
}

func (s *stubDeltaImportSvc) Preview(ctx context.Context, orgID int32) (*services.ImportCounts, error) {
	return nil, nil
}

func (s *stubDeltaImportSvc) Confirm(ctx context.Context, orgID int32) (*services.ImportCounts, error) {
	return nil, nil
}

func (s *stubDeltaImportSvc) DeltaSync(ctx context.Context, orgID int32) (*services.ImportCounts, error) {
	s.calls = append(s.calls, orgID)
	if orgID == s.failOnOrg {
		return nil, errors.New("sync boom")
	}
	return &services.ImportCounts{}, nil
}

type stubDeltaConnRepo struct {
	conns []*domain.OrgConnection
	err   error
}

func (s *stubDeltaConnRepo) Get(ctx context.Context, orgID int32) (*domain.OrgConnection, error) {
	return nil, nil
}

func (s *stubDeltaConnRepo) Upsert(ctx context.Context, conn *domain.OrgConnection) (*domain.OrgConnection, error) {
	return nil, nil
}

func (s *stubDeltaConnRepo) UpdateStatus(ctx context.Context, orgID int32, status domain.ConnectionStatus, lastError string) (*domain.OrgConnection, error) {
	return nil, nil
}

func (s *stubDeltaConnRepo) UpdateCredentials(ctx context.Context, orgID int32, clientIDEnc, clientSecretEnc []byte, nit, companyName string) (*domain.OrgConnection, error) {
	return nil, nil
}

func (s *stubDeltaConnRepo) Delete(ctx context.Context, orgID int32) error {
	return nil
}

func (s *stubDeltaConnRepo) ListByStatus(ctx context.Context, provider string, status domain.ConnectionStatus) ([]*domain.OrgConnection, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.conns, nil
}

func (s *stubDeltaConnRepo) ListAll(ctx context.Context) ([]*domain.OrgConnection, error) {
	return nil, nil
}

func nopCmdLogger() loggerDomain.Logger {
	return nopLogger{}
}

type nopLogger struct{}

func (nopLogger) Debug(msg string, fields ...loggerDomain.Fields) {}
func (nopLogger) Info(msg string, fields ...loggerDomain.Fields)  {}
func (nopLogger) Warn(msg string, fields ...loggerDomain.Fields)  {}
func (nopLogger) Error(msg string, fields ...loggerDomain.Fields) {}
func (nopLogger) Fatal(msg string, fields ...loggerDomain.Fields) {}
func (nopLogger) WithFields(fields loggerDomain.Fields) loggerDomain.Logger {
	return nopLogger{}
}

func TestRunDeltaSyncOnce_SyncsLiveOrgs(t *testing.T) {
	importSvc := &stubDeltaImportSvc{}
	repo := &stubDeltaConnRepo{conns: []*domain.OrgConnection{
		{OrganizationID: 1, Provider: "siigo", Status: domain.ConnStatusLive},
		{OrganizationID: 2, Provider: "siigo", Status: domain.ConnStatusLive},
	}}

	runDeltaSyncOnce(importSvc, repo, nopCmdLogger())

	if len(importSvc.calls) != 2 || importSvc.calls[0] != 1 || importSvc.calls[1] != 2 {
		t.Fatalf("expected delta sync for orgs 1,2, got %v", importSvc.calls)
	}
}

func TestRunDeltaSyncOnce_NoLiveOrgsNoCalls(t *testing.T) {
	importSvc := &stubDeltaImportSvc{}
	repo := &stubDeltaConnRepo{conns: nil}

	runDeltaSyncOnce(importSvc, repo, nopCmdLogger())

	if len(importSvc.calls) != 0 {
		t.Fatalf("expected no calls with no live orgs, got %v", importSvc.calls)
	}
}

func TestRunDeltaSyncOnce_FailureLoggedLoopContinues(t *testing.T) {
	importSvc := &stubDeltaImportSvc{failOnOrg: 2}
	repo := &stubDeltaConnRepo{conns: []*domain.OrgConnection{
		{OrganizationID: 1, Provider: "siigo", Status: domain.ConnStatusLive},
		{OrganizationID: 2, Provider: "siigo", Status: domain.ConnStatusLive},
		{OrganizationID: 3, Provider: "siigo", Status: domain.ConnStatusLive},
	}}

	runDeltaSyncOnce(importSvc, repo, nopCmdLogger())

	// All orgs attempted; failure on org 2 does not abort org 3.
	if len(importSvc.calls) != 3 {
		t.Fatalf("expected all 3 orgs attempted despite one failure, got %v", importSvc.calls)
	}
}

func TestRunDeltaSyncOnce_ListFailureLogged(t *testing.T) {
	importSvc := &stubDeltaImportSvc{}
	repo := &stubDeltaConnRepo{err: errors.New("list boom")}

	runDeltaSyncOnce(importSvc, repo, nopCmdLogger())

	if len(importSvc.calls) != 0 {
		t.Fatalf("expected no sync calls on list failure, got %v", importSvc.calls)
	}
}
