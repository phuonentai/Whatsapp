package services

import (
	"context"
	"fmt"

	"github.com/moasq/go-b2b-starter/internal/modules/invoicing/domain"
	loggerDomain "github.com/moasq/go-b2b-starter/internal/platform/logger/domain"
)

// TestInvoiceService creates a sandbox test invoice (no deal) and, when the
// provider reports a valid status, advances the connection to sandbox_ok.
type TestInvoiceService interface {
	CreateTestInvoice(ctx context.Context, orgID int32) (*domain.Invoice, error)
}

type testInvoiceService struct {
	provider domain.InvoicingProvider
	repo     domain.InvoiceRepository
	connSvc  ConnectionService
	logger   loggerDomain.Logger
}

func NewTestInvoiceService(
	provider domain.InvoicingProvider,
	repo domain.InvoiceRepository,
	connSvc ConnectionService,
	logger loggerDomain.Logger,
) TestInvoiceService {
	return &testInvoiceService{provider: provider, repo: repo, connSvc: connSvc, logger: logger}
}

// CreateTestInvoice issues a sandbox invoice with no deal (deal_id NULL),
// stores it, checks the provider status, and advances to sandbox_ok when
// valid. Pending results are returned as-is; the webhook/poll path completes
// the transition later.
func (s *testInvoiceService) CreateTestInvoice(ctx context.Context, orgID int32) (*domain.Invoice, error) {
	created, err := s.provider.CreateInvoice(ctx, orgID, &domain.InvoiceRequest{
		OrganizationID: orgID,
		Customer: domain.CustomerInfo{
			Name:             "Prueba Sandbox",
			Identification:   "C.C. 000000000",
			IdentificationType: "CC",
		},
		Amount:      ptr(0),
		Currency:    "COP",
		Description: "Factura de prueba — onboarding Siigo",
	})
	if err != nil {
		return nil, err
	}

	stored, err := s.repo.Insert(ctx, created)
	if err != nil {
		return nil, fmt.Errorf("failed to store test invoice: %w", err)
	}

	// Immediate status check: if the provider already resolved it, apply the
	// transition; otherwise the webhook/polling fallback completes it.
	remote, err := s.provider.GetInvoiceStatus(ctx, orgID, stored.ExternalID)
	if err != nil {
		s.logger.Warn("test invoice status check failed", map[string]any{
			"organization_id": orgID, "error": err.Error(),
		})
		return stored, nil
	}
	if remote.Status == stored.Status {
		return stored, nil
	}
	if _, err := s.repo.UpdateStatus(ctx, stored.ID, remote.Status, remote.Cufe, remote.PdfURL); err != nil {
		s.logger.Warn("failed to update test invoice status", map[string]any{"error": err.Error()})
		return stored, nil
	}
	stored.Status = remote.Status
	stored.Cufe = remote.Cufe
	stored.PdfURL = remote.PdfURL

	if stored.Status == domain.InvoiceStatusValid {
		if _, err := s.connSvc.ConfirmSandboxOK(ctx, orgID); err != nil {
			s.logger.Warn("failed to advance connection after sandbox test", map[string]any{
				"organization_id": orgID, "error": err.Error(),
			})
		}
	}
	return stored, nil
}

func ptr(v float64) *float64 { return &v }
