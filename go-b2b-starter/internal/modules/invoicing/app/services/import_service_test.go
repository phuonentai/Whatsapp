package services

import (
	"fmt"
	"context"
	"errors"
	"testing"

	crmdomain "github.com/moasq/go-b2b-starter/internal/modules/crm/domain"
	"github.com/moasq/go-b2b-starter/internal/modules/invoicing/domain"
)

type fakeCustomerReader struct {
	pages [][]domain.CustomerRecord
}

func (f *fakeCustomerReader) ListCustomers(ctx context.Context, orgID int32, page int32) ([]domain.CustomerRecord, error) {
	if int(page) >= len(f.pages) {
		return nil, nil
	}
	return f.pages[page], nil
}

type fakeImportCompanyRepo struct {
	byNit  map[string]*crmdomain.Company
	nextID int32
	creates []string
}

func newFakeImportCompanyRepo() *fakeImportCompanyRepo {
	return &fakeImportCompanyRepo{byNit: map[string]*crmdomain.Company{}}
}

func (f *fakeImportCompanyRepo) Create(ctx context.Context, company *crmdomain.Company) (*crmdomain.Company, error) {
	f.nextID++
	company.ID = f.nextID
	f.byNit[normalizeNit(company.Nit)] = company
	f.creates = append(f.creates, company.Name)
	return company, nil
}

func (f *fakeImportCompanyRepo) GetByID(ctx context.Context, orgID, companyID int32) (*crmdomain.CompanyWithCounts, error) {
	return nil, errors.New("not implemented")
}

func (f *fakeImportCompanyRepo) GetByNit(ctx context.Context, orgID int32, nit string) (*crmdomain.Company, error) {
	if c, ok := f.byNit[normalizeNit(nit)]; ok {
		return c, nil
	}
	return nil, crmdomain.ErrCompanyNotFound
}

func (f *fakeImportCompanyRepo) CountList(ctx context.Context, orgID int32) (int32, error) {
	return int32(len(f.byNit)), nil
}

func (f *fakeImportCompanyRepo) List(ctx context.Context, orgID int32, limit, offset int32) ([]*crmdomain.CompanyWithCounts, error) {
	return nil, errors.New("not implemented")
}

func (f *fakeImportCompanyRepo) CountSearch(ctx context.Context, orgID int32, query string) (int32, error) {
	return 0, nil
}

func (f *fakeImportCompanyRepo) Search(ctx context.Context, orgID int32, query string, limit, offset int32) ([]*crmdomain.CompanyWithCounts, error) {
	return nil, errors.New("not implemented")
}

func (f *fakeImportCompanyRepo) Update(ctx context.Context, company *crmdomain.Company) (*crmdomain.Company, error) {
	f.byNit[normalizeNit(company.Nit)] = company
	return company, nil
}

func (f *fakeImportCompanyRepo) Delete(ctx context.Context, orgID, companyID int32) error {
	return errors.New("not implemented")
}

type fakeImportContactRepo struct {
	upserts []*crmdomain.Contact
}

func (f *fakeImportContactRepo) UpsertByPhone(ctx context.Context, contact *crmdomain.Contact) (*crmdomain.Contact, error) {
	f.upserts = append(f.upserts, contact)
	return contact, nil
}

func (f *fakeImportContactRepo) GetByID(ctx context.Context, orgID, contactID int32) (*crmdomain.Contact, error) {
	return nil, errors.New("not implemented")
}

func (f *fakeImportContactRepo) GetByPhone(ctx context.Context, orgID int32, phoneNumber string) (*crmdomain.Contact, error) {
	return nil, errors.New("not implemented")
}

func (f *fakeImportContactRepo) List(ctx context.Context, orgID int32, limit, offset int32) ([]*crmdomain.Contact, error) {
	return nil, errors.New("not implemented")
}

func (f *fakeImportContactRepo) CountFiltered(ctx context.Context, orgID int32, source, leadStatus string, companyID, assignedTo int32) (int32, error) {
	return 0, nil
}

func (f *fakeImportContactRepo) ListFiltered(ctx context.Context, orgID int32, source, leadStatus string, companyID, assignedTo, limit, offset int32) ([]*crmdomain.Contact, error) {
	return nil, errors.New("not implemented")
}

func (f *fakeImportContactRepo) CountSearch(ctx context.Context, orgID int32, query string) (int32, error) {
	return 0, nil
}

func (f *fakeImportContactRepo) Search(ctx context.Context, orgID int32, query string, limit, offset int32) ([]*crmdomain.Contact, error) {
	return nil, errors.New("not implemented")
}

func (f *fakeImportContactRepo) Update(ctx context.Context, contact *crmdomain.Contact) (*crmdomain.Contact, error) {
	return contact, nil
}

func (f *fakeImportContactRepo) Delete(ctx context.Context, orgID, contactID int32) error {
	return errors.New("not implemented")
}

type fakeRunRepo struct {
	runs []*domain.ImportRun
}

func (f *fakeRunRepo) Record(ctx context.Context, run *domain.ImportRun) (*domain.ImportRun, error) {
	f.runs = append(f.runs, run)
	return run, nil
}

func (f *fakeRunRepo) ListByOrg(ctx context.Context, orgID int32, limit int32) ([]*domain.ImportRun, error) {
	return f.runs, nil
}

func newTestImportService() (*importService, *fakeImportCompanyRepo, *fakeImportContactRepo, *fakeRunRepo) {
	companies := newFakeImportCompanyRepo()
	contacts := &fakeImportContactRepo{}
	runs := &fakeRunRepo{}
	svc := &importService{
		reader:      nil,
		companyRepo: companies,
		contactRepo: contacts,
		runRepo:     runs,
		logger:      nopLogger{},
	}
	return svc, companies, contacts, runs
}

func sampleRecords() []domain.CustomerRecord {
	return []domain.CustomerRecord{
		{ExternalID: "c1", Name: "Cliente Uno", Identification: "900.111.222-3", Email: "a@b.co", Phone: "+57300111"},
		{ExternalID: "c2", Name: "Cliente Dos", Identification: "9001112223"}, // duplicado NIT del anterior
		{ExternalID: "c3", Name: "Cliente Tres", Identification: "800333444", Phone: "+57300222"},
		{ExternalID: "c4", Name: "", Identification: "700000001"},            // sin nombre
		{ExternalID: "c5", Name: "Sin Nit", Identification: ""},              // sin NIT
	}
}

func TestImport_PreviewReportsCountsWithoutWriting(t *testing.T) {
	svc, companies, contacts, runs := newTestImportService()
	svc.reader = &fakeCustomerReader{pages: [][]domain.CustomerRecord{sampleRecords()}}

	counts, err := svc.Preview(context.Background(), 7)
	if err != nil {
		t.Fatalf("preview failed: %v", err)
	}
	if counts.Total != 5 || counts.Duplicados != 1 || counts.SinNombre != 1 || counts.SinNit != 1 {
		t.Fatalf("unexpected preview counts: %+v", counts)
	}
	if counts.Nuevos != 2 {
		t.Fatalf("expected 2 nuevos, got %d", counts.Nuevos)
	}
	if len(companies.creates) != 0 || len(contacts.upserts) != 0 || len(runs.runs) != 0 {
		t.Fatal("preview must not write companies, contacts, or runs")
	}
}

func TestImport_ConfirmUpsertsAndRecordsRun(t *testing.T) {
	svc, companies, contacts, runs := newTestImportService()
	svc.reader = &fakeCustomerReader{pages: [][]domain.CustomerRecord{sampleRecords()}}

	counts, err := svc.Confirm(context.Background(), 7)
	if err != nil {
		t.Fatalf("confirm failed: %v", err)
	}
	if counts.Nuevos != 2 || counts.Existentes != 0 {
		t.Fatalf("unexpected confirm counts: %+v", counts)
	}
	if len(companies.creates) != 2 {
		t.Fatalf("expected 2 companies created, got %d", len(companies.creates))
	}
	if len(contacts.upserts) != 2 {
		t.Fatalf("expected 2 contacts, got %d", len(contacts.upserts))
	}
	if len(runs.runs) != 1 || runs.runs[0].Kind != domain.ImportRunConfirm {
		t.Fatalf("expected one confirm run, got %+v", runs.runs)
	}

	// Re-run: idempotent — no new companies, all existentes.
	counts2, err := svc.Confirm(context.Background(), 7)
	if err != nil {
		t.Fatal(err)
	}
	if counts2.Nuevos != 0 || counts2.Existentes != 2 {
		t.Fatalf("second confirm must not duplicate: %+v", counts2)
	}
	if len(companies.creates) != 2 {
		t.Fatalf("companies duplicated on re-confirm: %d", len(companies.creates))
	}
}

func TestImport_DeltaSyncRecordsDeltaRun(t *testing.T) {
	svc, _, _, runs := newTestImportService()
	svc.reader = &fakeCustomerReader{pages: [][]domain.CustomerRecord{sampleRecords()}}

	if _, err := svc.DeltaSync(context.Background(), 7); err != nil {
		t.Fatalf("delta sync failed: %v", err)
	}
	if len(runs.runs) != 1 || runs.runs[0].Kind != domain.ImportRunDelta {
		t.Fatalf("expected delta run, got %+v", runs.runs)
	}
}

func TestImport_PaginatesUntilShortPage(t *testing.T) {
	svc, companies, _, _ := newTestImportService()
	svc.reader = &fakeCustomerReader{pages: [][]domain.CustomerRecord{
		makeRecords(100, 1),
		makeRecords(100, 101),
		makeRecords(10, 201),
	}}

	counts, err := svc.Confirm(context.Background(), 7)
	if err != nil {
		t.Fatal(err)
	}
	if counts.Total != 210 || counts.Nuevos != 210 {
		t.Fatalf("expected 210 imported across pages, got %+v", counts)
	}
	if len(companies.creates) != 210 {
		t.Fatalf("expected 210 companies, got %d", len(companies.creates))
	}
}

func makeRecords(n, start int) []domain.CustomerRecord {
	records := make([]domain.CustomerRecord, n)
	for i := 0; i < n; i++ {
		id := start + i
		records[i] = domain.CustomerRecord{
			ExternalID:     fmt.Sprintf("c-%d", id),
			Name:           fmt.Sprintf("Cliente %d", id),
			Identification: fmt.Sprintf("900%06d", id),
			Phone:          fmt.Sprintf("+57300%06d", id),
		}
	}
	return records
}

func (f *fakeImportContactRepo) UpsertByIGUser(ctx context.Context, contact *crmdomain.Contact) (*crmdomain.Contact, error) {
	return contact, nil
}

func (f *fakeImportContactRepo) GetByIGUser(ctx context.Context, orgID int32, igUserID string) (*crmdomain.Contact, error) {
	return nil, crmdomain.ErrContactNotFound
}

func (f *fakeImportContactRepo) UpdateInstagramProfile(ctx context.Context, orgID, contactID int32, username, avatarURL, displayName string) (*crmdomain.Contact, error) {
	return nil, crmdomain.ErrContactNotFound
}
