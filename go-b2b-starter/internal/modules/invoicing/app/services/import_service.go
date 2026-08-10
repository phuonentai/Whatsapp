package services

import (
	"context"
	"fmt"

	crmdomain "github.com/moasq/go-b2b-starter/internal/modules/crm/domain"
	"github.com/moasq/go-b2b-starter/internal/modules/invoicing/domain"
	loggerDomain "github.com/moasq/go-b2b-starter/internal/platform/logger/domain"
)

// ImportCounts mirrors the preview/result counters exposed to the UI.
type ImportCounts struct {
	Total       int32 `json:"total"`
	Nuevos      int32 `json:"nuevos"`
	Existentes  int32 `json:"existentes"`
	Duplicados  int32 `json:"duplicados"`
	SinNit      int32 `json:"sin_nit"`
	SinNombre   int32 `json:"sin_nombre"`
	Contactos   int32 `json:"contactos"`
	SinContacto int32 `json:"sin_contacto"`
}

// ImportService imports provider customers into the CRM with NIT dedupe.
// Preview writes nothing; confirm/delta upsert and record an import run.
type ImportService interface {
	Preview(ctx context.Context, orgID int32) (*ImportCounts, error)
	Confirm(ctx context.Context, orgID int32) (*ImportCounts, error)
	DeltaSync(ctx context.Context, orgID int32) (*ImportCounts, error)
}

type importService struct {
	reader       domain.CustomerReader
	companyRepo  crmdomain.CompanyRepository
	contactRepo  crmdomain.ContactRepository
	runRepo      domain.ImportRunRepository
	logger       loggerDomain.Logger
}

func NewImportService(
	reader domain.CustomerReader,
	companyRepo crmdomain.CompanyRepository,
	contactRepo crmdomain.ContactRepository,
	runRepo domain.ImportRunRepository,
	logger loggerDomain.Logger,
) ImportService {
	return &importService{reader: reader, companyRepo: companyRepo, contactRepo: contactRepo, runRepo: runRepo, logger: logger}
}

// Preview pulls the provider customers and reports counts without writing a
// single row (including no import run).
func (s *importService) Preview(ctx context.Context, orgID int32) (*ImportCounts, error) {
	records, err := s.pullAll(ctx, orgID)
	if err != nil {
		return nil, err
	}
	counts, unique := s.group(records)
	// Classify against existing companies (read-only).
	for _, rec := range unique {
		if _, err := s.companyRepo.GetByNit(ctx, orgID, normalizeNit(rec.Identification)); err == nil {
			counts.Existentes++
		} else if err != crmdomain.ErrCompanyNotFound {
			return nil, err
		} else {
			counts.Nuevos++
		}
	}
	return counts, nil
}

// Confirm applies the full pull as upserts and records the run.
func (s *importService) Confirm(ctx context.Context, orgID int32) (*ImportCounts, error) {
	return s.apply(ctx, orgID, domain.ImportRunConfirm)
}

// DeltaSync is the confirmation path without preview, recorded as a delta run
// (used by the on-demand endpoint and the nightly job).
func (s *importService) DeltaSync(ctx context.Context, orgID int32) (*ImportCounts, error) {
	return s.apply(ctx, orgID, domain.ImportRunDelta)
}

func (s *importService) apply(ctx context.Context, orgID int32, kind domain.ImportRunKind) (*ImportCounts, error) {
	records, err := s.pullAll(ctx, orgID)
	if err != nil {
		return nil, err
	}
	counts, unique := s.group(records)

	// Upsert phase: NIT-first dedupe on the unique set; contact failures are
	// logged and never fail the company import.
	for _, rec := range unique {
		if err := s.upsertCustomer(ctx, orgID, rec, normalizeNit(rec.Identification), counts); err != nil {
			s.logger.Warn("import upsert failed for customer", map[string]any{
				"organization_id": orgID,
				"external_id":     rec.ExternalID,
				"error":           err.Error(),
			})
			continue
		}
	}

	if _, err := s.runRepo.Record(ctx, &domain.ImportRun{
		OrganizationID: orgID,
		Kind:           kind,
		Counts:         counts.toMap(),
	}); err != nil {
		return nil, fmt.Errorf("failed to record import run: %w", err)
	}
	return counts, nil
}

func (s *importService) upsertCustomer(ctx context.Context, orgID int32, rec domain.CustomerRecord, nit string, counts *ImportCounts) error {
	company, err := s.companyRepo.GetByNit(ctx, orgID, nit)
	switch {
	case err == nil:
		counts.Existentes++
		// Refresh mutable fields from the provider (idempotent update).
		if company.Phone != rec.Phone || company.Name != rec.Name {
			company.Name = rec.Name
			company.Phone = rec.Phone
			if _, err := s.companyRepo.Update(ctx, company); err != nil {
				return err
			}
		}
	case err == crmdomain.ErrCompanyNotFound:
		counts.Nuevos++
		created, err := s.companyRepo.Create(ctx, &crmdomain.Company{
			OrganizationID: orgID,
			Name:           rec.Name,
			Nit:            rec.Identification,
			Phone:          rec.Phone,
			Metadata:       map[string]any{"siigo_customer_id": rec.ExternalID, "siigo_imported": true},
		})
		if err != nil {
			return err
		}
		company = created
	default:
		return err
	}

	// Linked contact when there is a reachable phone (or email).
	if rec.Phone != "" || rec.Email != "" {
		contact := &crmdomain.Contact{
			OrganizationID: orgID,
			PhoneNumber:     rec.Phone,
			DisplayName:     rec.Name,
			Email:           rec.Email,
			CompanyID:       &company.ID,
			Source:          crmdomain.ContactSourceImport,
			NumeroDocumento: rec.Identification,
			Metadata:        map[string]any{"siigo_customer_id": rec.ExternalID},
		}
		if _, err := s.contactRepo.UpsertByPhone(ctx, contact); err != nil {
			s.logger.Warn("contact upsert failed during import", map[string]any{"error": err.Error()})
			return nil // contact failure must not fail the company import
		}
		counts.Contactos++
	} else {
		counts.SinContacto++
	}
	return nil
}

func (s *importService) pullAll(ctx context.Context, orgID int32) ([]domain.CustomerRecord, error) {
	var all []domain.CustomerRecord
	for page := int32(0); ; page++ {
		pageRecords, err := s.reader.ListCustomers(ctx, orgID, page)
		if err != nil {
			return nil, err
		}
		all = append(all, pageRecords...)
		if len(pageRecords) < 100 {
			break
		}
	}
	return all, nil
}

// group classifies the pull and deduplicates within it by normalized NIT
// (duplicates counted, kept only once for the upsert phase). Returns the
// counts and the unique, importable records.
func (s *importService) group(records []domain.CustomerRecord) (*ImportCounts, []domain.CustomerRecord) {
	counts := &ImportCounts{Total: int32(len(records))}
	seen := map[string]bool{}
	unique := make([]domain.CustomerRecord, 0, len(records))
	for _, rec := range records {
		nit := normalizeNit(rec.Identification)
		if rec.Name == "" {
			counts.SinNombre++
			continue
		}
		if nit == "" {
			counts.SinNit++
			continue
		}
		if seen[nit] {
			counts.Duplicados++
			continue
		}
		seen[nit] = true
		unique = append(unique, rec)
	}
	return counts, unique
}

func (c *ImportCounts) toMap() map[string]int32 {
	return map[string]int32{
		"total": c.Total, "nuevos": c.Nuevos, "existentes": c.Existentes,
		"duplicados": c.Duplicados, "sin_nit": c.SinNit, "sin_nombre": c.SinNombre,
		"contactos": c.Contactos, "sin_contacto": c.SinContacto,
	}
}
