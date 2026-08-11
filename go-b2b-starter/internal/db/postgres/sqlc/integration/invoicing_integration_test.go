//go:build integration

package integration

import (
	"context"
	"testing"

	crmrepos "github.com/moasq/go-b2b-starter/internal/modules/crm/infra/repositories"
	"github.com/moasq/go-b2b-starter/internal/modules/invoicing/app/services"
	"github.com/moasq/go-b2b-starter/internal/modules/invoicing/domain"
	"github.com/moasq/go-b2b-starter/internal/modules/invoicing/infra/repositories"
	loggerDomain "github.com/moasq/go-b2b-starter/internal/platform/logger/domain"
)

type itLogger struct{}

func (itLogger) Debug(msg string, fields ...loggerDomain.Fields) {}
func (itLogger) Info(msg string, fields ...loggerDomain.Fields)  {}
func (itLogger) Warn(msg string, fields ...loggerDomain.Fields)  {}
func (itLogger) Error(msg string, fields ...loggerDomain.Fields) {}
func (itLogger) Fatal(msg string, fields ...loggerDomain.Fields) {}
func (itLogger) WithFields(fields loggerDomain.Fields) loggerDomain.Logger {
	return itLogger{}
}

func TestOrgConnectionRepository_RoundTrip(t *testing.T) {
	ctx := context.Background()
	orgID, _ := createOrgWithAccount(t, ctx, testStore)
	repo := repositories.NewConnectionRepository(testStore)

	// Upsert creates the row with the initial state.
	conn, err := repo.Upsert(ctx, &domain.OrgConnection{
		OrganizationID: orgID,
		Provider:       "siigo",
		Status:         domain.ConnStatusConnected,
	})
	if err != nil {
		t.Fatalf("upsert: %v", err)
	}
	if conn.Status != domain.ConnStatusConnected || conn.Provider != "siigo" {
		t.Fatalf("unexpected upsert result: %+v", conn)
	}

	// Get round trip.
	got, err := repo.Get(ctx, orgID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.OrganizationID != orgID || got.Status != domain.ConnStatusConnected {
		t.Fatalf("unexpected get: %+v", got)
	}

	// UpdateStatus transitions the state.
	updated, err := repo.UpdateStatus(ctx, orgID, domain.ConnStatusLive, "")
	if err != nil {
		t.Fatalf("update status: %v", err)
	}
	if updated.Status != domain.ConnStatusLive {
		t.Fatalf("expected live, got %s", updated.Status)
	}

	// UpdateCredentials stores ciphertext columns and clears last_error.
	withCreds, err := repo.UpdateCredentials(ctx, orgID, []byte("enc:client-id"), []byte("enc:client-secret"), "9001234567", "Mi Empresa")
	if err != nil {
		t.Fatalf("update credentials: %v", err)
	}
	if string(withCreds.ClientIDEnc) != "enc:client-id" || string(withCreds.ClientSecretEnc) != "enc:client-secret" {
		t.Fatalf("ciphertext columns not stored: %+v", withCreds)
	}
	if withCreds.Nit != "9001234567" || withCreds.SiigoCompanyName != "Mi Empresa" {
		t.Fatalf("unexpected credential fields: %+v", withCreds)
	}

	// UpdateStatus with last_error round trip.
	withErr, err := repo.UpdateStatus(ctx, orgID, domain.ConnStatusPaused, "some transient failure")
	if err != nil {
		t.Fatalf("update status with error: %v", err)
	}
	if withErr.LastError != "some transient failure" {
		t.Fatalf("expected last_error persisted, got %q", withErr.LastError)
	}

	// ListByStatus finds the row; Delete removes it.
	live, err := repo.ListByStatus(ctx, "siigo", domain.ConnStatusLive)
	if err != nil {
		t.Fatalf("list by status: %v", err)
	}
	if len(live) != 0 {
		t.Fatalf("expected no live rows, got %d", len(live))
	}
	paused, err := repo.ListByStatus(ctx, "siigo", domain.ConnStatusPaused)
	if err != nil {
		t.Fatalf("list paused: %v", err)
	}
	if len(paused) != 1 || paused[0].OrganizationID != orgID {
		t.Fatalf("expected 1 paused row for org, got %+v", paused)
	}

	if err := repo.Delete(ctx, orgID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := repo.Get(ctx, orgID); err != domain.ErrConnectionNotFound {
		t.Fatalf("expected ErrConnectionNotFound after delete, got %v", err)
	}
}

func TestOrgConnectionRepository_ListAll(t *testing.T) {
	ctx := context.Background()
	orgID, _ := createOrgWithAccount(t, ctx, testStore)
	repo := repositories.NewConnectionRepository(testStore)

	if _, err := repo.Upsert(ctx, &domain.OrgConnection{
		OrganizationID: orgID,
		Provider:       "none",
		Status:         domain.ConnStatusAwaitingSetup,
	}); err != nil {
		t.Fatalf("upsert: %v", err)
	}

	all, err := repo.ListAll(ctx)
	if err != nil {
		t.Fatalf("list all: %v", err)
	}
	found := false
	for _, c := range all {
		if c.OrganizationID == orgID && c.Status == domain.ConnStatusAwaitingSetup {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected org %d in ListAll, got %+v", orgID, all)
	}
}

type itCustomerReader struct {
	records []domain.CustomerRecord
}

func (r *itCustomerReader) ListCustomers(ctx context.Context, orgID int32, page int32) ([]domain.CustomerRecord, error) {
	if page == 0 {
		return r.records, nil
	}
	return nil, nil
}

func TestImportConfirm_IsIdempotentOnSecondRun(t *testing.T) {
	ctx := context.Background()
	orgID, _ := createOrgWithAccount(t, ctx, testStore)

	importSvc := services.NewImportService(
		&itCustomerReader{records: []domain.CustomerRecord{
			{ExternalID: "c1", Name: "Cliente Uno", Identification: "900.111.222-3", Phone: "+57300111111"},
			{ExternalID: "c2", Name: "Cliente Dos", Identification: "800333444", Email: "dos@mock.co"},
		}},
		crmrepos.NewCompanyRepository(testStore),
		crmrepos.NewContactRepository(testStore),
		repositories.NewImportRunRepository(testStore),
		itLogger{},
	)

	first, err := importSvc.Confirm(ctx, orgID)
	if err != nil {
		t.Fatalf("first confirm: %v", err)
	}
	if first.Nuevos != 2 || first.Existentes != 0 {
		t.Fatalf("unexpected first counts: %+v", first)
	}

	// Second run must dedupe by NIT: no new companies, all existentes.
	second, err := importSvc.Confirm(ctx, orgID)
	if err != nil {
		t.Fatalf("second confirm: %v", err)
	}
	if second.Nuevos != 0 || second.Existentes != 2 {
		t.Fatalf("second confirm must be idempotent: %+v", second)
	}

	// Both runs recorded.
	runRepo := repositories.NewImportRunRepository(testStore)
	runs, err := runRepo.ListByOrg(ctx, orgID, 10)
	if err != nil {
		t.Fatalf("list runs: %v", err)
	}
	if len(runs) != 2 {
		t.Fatalf("expected 2 import runs, got %d", len(runs))
	}
	if runs[0].Kind != domain.ImportRunConfirm {
		t.Fatalf("expected confirm runs, got %+v", runs[0])
	}
}
