package services

import (
	"context"
	"fmt"

	"github.com/moasq/go-b2b-starter/internal/modules/whatsapp/domain/events"
	"github.com/moasq/go-b2b-starter/internal/platform/logger"
	loggerdomain "github.com/moasq/go-b2b-starter/internal/platform/logger/domain"
)

type EchoListener interface {
	HandleMessageEcho(ctx context.Context, event *events.MessageEcho) error
}

type echoListener struct {
	crmService CRMService
	logger     logger.Logger
}

func NewEchoListener(crmService CRMService, log logger.Logger) EchoListener {
	return &echoListener{
		crmService: crmService,
		logger:     log,
	}
}

func (l *echoListener) HandleMessageEcho(ctx context.Context, event *events.MessageEcho) error {
	if err := l.crmService.ProcessEchoMessage(ctx, event); err != nil {
		l.logger.Error("failed to process echo message", loggerdomain.Fields{
			"org_id":  event.OrganizationID,
			"to":      event.To,
			"msg_sid": event.MessageSID,
			"error":   err.Error(),
		})
		return fmt.Errorf("failed to process echo message: %w", err)
	}

	l.logger.Info("echo message processed", loggerdomain.Fields{
		"org_id":  event.OrganizationID,
		"to":      event.To,
		"msg_sid": event.MessageSID,
		"type":    event.MessageType,
	})

	return nil
}
