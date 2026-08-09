package services

import (
	"context"
	"fmt"

	crmEvents "github.com/moasq/go-b2b-starter/internal/modules/crm/domain/events"
	loggerDomain "github.com/moasq/go-b2b-starter/internal/platform/logger/domain"
)

// InvoicingStageName is the deal stage name that triggers invoice creation.
// Seeded by the vertical playbooks ("facturado").
const InvoicingStageName = "facturado"

// DealStageListener reacts to crm deal stage changes, dispatching to the
// invoicing service when a deal reaches the invoicing stage. Registered as a
// second subscriber on the same event as the crm DealStageListener.
type DealStageListener interface {
	HandleStageChanged(ctx context.Context, event *crmEvents.DealStageChanged) error
}

type dealStageListener struct {
	invoicing InvoicingService
	logger    loggerDomain.Logger
}

func NewDealStageListener(invoicing InvoicingService, log loggerDomain.Logger) DealStageListener {
	return &dealStageListener{invoicing: invoicing, logger: log}
}

func (l *dealStageListener) HandleStageChanged(ctx context.Context, event *crmEvents.DealStageChanged) error {
	if event.NewStageName != InvoicingStageName {
		return nil
	}

	if _, err := l.invoicing.CreateForDeal(ctx, event.OrganizationID, event.DealID); err != nil {
		l.logger.Warn("failed to create invoice from deal stage change", map[string]any{
			"organization_id": event.OrganizationID,
			"deal_id":         event.DealID,
			"error":           err.Error(),
		})
		return fmt.Errorf("failed to create invoice for deal %d: %w", event.DealID, err)
	}
	return nil
}
