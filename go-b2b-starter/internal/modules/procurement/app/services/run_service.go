package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/moasq/go-b2b-starter/internal/modules/procurement/domain"
	procurementEvents "github.com/moasq/go-b2b-starter/internal/modules/procurement/domain/events"
	logger "github.com/moasq/go-b2b-starter/internal/platform/logger/domain"
)

// runService implements run creation (with one metered draft per supplier),
// the durable fan-out trigger, and run listing.
type runService struct {
	runs      domain.InquiryRunRepository
	suppliers domain.SupplierRepository
	products  domain.ProductRepository
	audit     domain.AuditRepository
	drafting  DraftingService
	metrics   MetricsSink
	log       logger.Logger
}

// NewRunService builds the run service.
func NewRunService(
	runs domain.InquiryRunRepository,
	suppliers domain.SupplierRepository,
	products domain.ProductRepository,
	audit domain.AuditRepository,
	drafting DraftingService,
	metrics MetricsSink,
	log logger.Logger,
) *runService {
	if metrics == nil {
		metrics = noopMetrics{}
	}
	return &runService{runs: runs, suppliers: suppliers, products: products, audit: audit, drafting: drafting, metrics: metrics, log: log}
}

// CreateRun creates the run in draft, drafts exactly one metered Spanish
// message per supplier (D3), and stores one recipient per supplier. On credit
// exhaustion or LLM failure the run is escalated with an audit and NO further
// unmetered calls happen; the escalated run is returned.
func (s *runService) CreateRun(ctx context.Context, orgID int32, memberID string, in CreateRunInput) (*domain.InquiryRun, error) {
	if len(in.SupplierIDs) == 0 {
		return nil, domain.ErrNoSuppliers
	}
	if len(in.Products) == 0 {
		return nil, domain.ErrNoProducts
	}

	suppliers, err := s.runs.ListSuppliersWithDisplay(ctx, orgID, in.SupplierIDs)
	if err != nil {
		return nil, err
	}
	if len(suppliers) != len(in.SupplierIDs) {
		return nil, domain.ErrSupplierNotFound
	}
	for _, sw := range suppliers {
		if !sw.Supplier.IsActive {
			return nil, fmt.Errorf("%w: %s", domain.ErrSupplierInactive, sw.Supplier.NIT)
		}
	}

	productIDs := make([]int32, 0, len(in.Products))
	quantities := make(map[int32]float64, len(in.Products))
	for _, p := range in.Products {
		productIDs = append(productIDs, p.ProductID)
		quantities[p.ProductID] = p.Quantity
	}
	products, err := s.products.ListByIDs(ctx, orgID, productIDs)
	if err != nil {
		return nil, err
	}
	if len(products) != len(productIDs) {
		return nil, domain.ErrProductNotFound
	}

	run, err := s.runs.CreateRun(ctx, orgID, in.Nota, memberID)
	if err != nil {
		return nil, err
	}

	for _, sw := range suppliers {
		draft, err := s.drafting.DraftForSupplier(ctx, orgID, sw.Supplier, sw.DisplayName, products, quantities)
		if err != nil {
			s.escalate(ctx, orgID, run.ID, fmt.Sprintf("drafting failed: %v", err))
			return run, err
		}
		if _, err := s.runs.CreateRecipient(ctx, orgID, run.ID, sw.Supplier.ID, sw.Supplier.ContactID, &draft); err != nil {
			return nil, err
		}
	}
	return run, nil
}

// SendRun transitions draft→sending and enqueues exactly one durable outbox
// event per recipient, atomically (no enqueue-without-transition, D6/D14).
func (s *runService) SendRun(ctx context.Context, orgID, runID int32) (*domain.InquiryRun, error) {
	run, err := s.runs.GetRun(ctx, orgID, runID)
	if err != nil {
		return nil, err
	}
	if run.Status != domain.RunDraft {
		return nil, domain.ErrRunNotDraft
	}

	recipients, err := s.runs.ListRunRecipientsWithPhone(ctx, orgID, runID)
	if err != nil {
		return nil, err
	}
	if len(recipients) == 0 {
		return nil, domain.ErrNoDraftedMessages
	}
	for _, r := range recipients {
		if r.Recipient.DraftedMessage == nil || *r.Recipient.DraftedMessage == "" {
			return nil, domain.ErrNoDraftedMessages
		}
		if r.ContactPhone == "" {
			return nil, fmt.Errorf("recipient %d has no phone", r.Recipient.ID)
		}
	}

	events := make([]domain.OutboxEventInput, 0, len(recipients))
	for _, r := range recipients {
		ev := procurementEvents.NewInquirySend(
			orgID, runID, r.Recipient.ID, r.Recipient.SupplierID, r.Recipient.ContactID,
			r.ContactPhone, *r.Recipient.DraftedMessage,
		)
		payload, err := json.Marshal(ev)
		if err != nil {
			return nil, err
		}
		events = append(events, domain.OutboxEventInput{
			EventType: procurementEvents.InquirySendEventType,
			Payload:   payload,
		})
	}

	return s.runs.SendFanOut(ctx, orgID, runID, events)
}

func (s *runService) ListRuns(ctx context.Context, orgID int32, limit, offset int32) ([]*domain.InquiryRun, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	return s.runs.ListRuns(ctx, orgID, limit, offset)
}

// escalate transitions the run to escalated (always allowed while
// in-progress) and audits it. Failures are logged, never fatal.
func (s *runService) escalate(ctx context.Context, orgID, runID int32, reason string) {
	if _, err := s.runs.TransitionRun(ctx, orgID, runID, domain.RunDraft, domain.RunEscalated); err != nil {
		if !errors.Is(err, domain.ErrInvalidTransition) {
			s.log.Error("escalate run failed", map[string]any{"run_id": runID, "error": err.Error()})
		}
	}
	if err := s.audit.Record(ctx, domain.AuditEntry{
		OrganizationID: orgID,
		EntityType:     "inquiry_run",
		EntityID:       &runID,
		Action:         "escalate",
		Decision:       "skip",
		Reason:         strPtr2("drafting_failed"),
		Metadata:       map[string]any{"error": reason},
	}); err != nil {
		s.log.Error("audit escalate failed", map[string]any{"error": err.Error()})
	}
}
