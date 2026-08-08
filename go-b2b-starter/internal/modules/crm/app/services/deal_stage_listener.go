package services

import (
	"context"
	"fmt"

	"github.com/moasq/go-b2b-starter/internal/modules/crm/domain"
	"github.com/moasq/go-b2b-starter/internal/modules/crm/domain/events"
	"github.com/moasq/go-b2b-starter/internal/platform/logger"
	loggerdomain "github.com/moasq/go-b2b-starter/internal/platform/logger/domain"
)

type DealStageListener interface {
	HandleStageChanged(ctx context.Context, event *events.DealStageChanged) error
}

type dealStageListener struct {
	activityService ActivityService
	logger          logger.Logger
}

func NewDealStageListener(activityService ActivityService, log logger.Logger) DealStageListener {
	return &dealStageListener{
		activityService: activityService,
		logger:          log,
	}
}

func (l *dealStageListener) HandleStageChanged(ctx context.Context, event *events.DealStageChanged) error {
	dealID := event.DealID
	if _, err := l.activityService.Create(ctx, event.OrganizationID, &CreateActivityRequest{
		DealID: &dealID,
		Tipo:   domain.ActivityTypeSistema,
		Asunto: fmt.Sprintf("Etapa cambiada a %s", event.NewStageName),
		Contenido: fmt.Sprintf("Negocio movido de %s a %s", event.OldStageName, event.NewStageName),
		Metadata: map[string]interface{}{
			"old_stage_name": event.OldStageName,
			"new_stage_name": event.NewStageName,
			"changed_by":     event.ChangedBy,
		},
	}); err != nil {
		l.logger.Error("failed to record stage change activity", loggerdomain.Fields{
			"org_id": event.OrganizationID,
			"deal_id": event.DealID,
			"error":   err.Error(),
		})
		return fmt.Errorf("failed to record stage change activity: %w", err)
	}

	l.logger.Info("stage change activity recorded", loggerdomain.Fields{
		"org_id":   event.OrganizationID,
		"deal_id":  event.DealID,
		"new_stage": event.NewStageName,
	})

	return nil
}
