package services

import (
	"context"
	"fmt"

	crmDomain "github.com/moasq/go-b2b-starter/internal/modules/crm/domain"
	crmServices "github.com/moasq/go-b2b-starter/internal/modules/crm/app/services"
	crmEvents "github.com/moasq/go-b2b-starter/internal/modules/crm/domain/events"
	loggerDomain "github.com/moasq/go-b2b-starter/internal/platform/logger/domain"
)

// InvoicingStageName is the deal stage name that triggers invoice creation.
// Seeded by the vertical playbooks ("facturado").
const InvoicingStageName = "facturado"

// DealStageListener reacts to crm deal stage changes, dispatching to the
// invoicing service when a deal reaches the invoicing stage AND the
// organization's invoicing connection is live. Any other connection state
// records an "Facturación no activa" activity and makes no provider call.
// Registered as a second subscriber on the same event as the crm listener.
type DealStageListener interface {
	HandleStageChanged(ctx context.Context, event *crmEvents.DealStageChanged) error
}

type dealStageListener struct {
	invoicing   InvoicingService
	connSvc     ConnectionService
	activitySvc crmServices.ActivityService
	logger      loggerDomain.Logger
}

func NewDealStageListener(
	invoicing InvoicingService,
	connSvc ConnectionService,
	activitySvc crmServices.ActivityService,
	log loggerDomain.Logger,
) DealStageListener {
	return &dealStageListener{invoicing: invoicing, connSvc: connSvc, activitySvc: activitySvc, logger: log}
}

func (l *dealStageListener) HandleStageChanged(ctx context.Context, event *crmEvents.DealStageChanged) error {
	if event.NewStageName != InvoicingStageName {
		return nil
	}

	live, err := l.connSvc.IsLive(ctx, event.OrganizationID)
	if err != nil {
		l.logger.Warn("failed to check invoicing connection state", map[string]any{
			"organization_id": event.OrganizationID,
			"deal_id":         event.DealID,
			"error":           err.Error(),
		})
		return fmt.Errorf("failed to check invoicing connection for deal %d: %w", event.DealID, err)
	}

	if !live {
		// Non-live connection: no provider call, no invoice, no WhatsApp
		// invoice message. Record why, loudly, on the deal.
		l.recordInactiveActivity(ctx, event)
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

func (l *dealStageListener) recordInactiveActivity(ctx context.Context, event *crmEvents.DealStageChanged) {
	if _, err := l.activitySvc.Create(ctx, event.OrganizationID, &crmServices.CreateActivityRequest{
		DealID:    &event.DealID,
		Tipo:      crmDomain.ActivityTypeSistema,
		Asunto:    "Facturación no activa",
		Contenido: "El negocio llegó a la etapa facturado, pero la facturación automática no está activa para esta organización. Activa la conexión Siigo en Configuración.",
	}); err != nil {
		l.logger.Warn("failed to record inactive invoicing activity", map[string]any{
			"deal_id": event.DealID, "error": err.Error(),
		})
	}
}
