package services

import (
	"context"
	"fmt"

	"github.com/moasq/go-b2b-starter/internal/modules/whatsapp/domain/events"
	"github.com/moasq/go-b2b-starter/internal/platform/logger"
	loggerdomain "github.com/moasq/go-b2b-starter/internal/platform/logger/domain"
)

type MessageListener interface {
	HandleMessageReceived(ctx context.Context, event *events.MessageReceived) error
}

type messageListener struct {
	crmService CRMService
	logger     logger.Logger
}

func NewMessageListener(crmService CRMService, log logger.Logger) MessageListener {
	return &messageListener{
		crmService: crmService,
		logger:     log,
	}
}

func (l *messageListener) HandleMessageReceived(ctx context.Context, event *events.MessageReceived) error {
	if err := l.crmService.ProcessInboundMessage(ctx, event); err != nil {
		l.logger.Error("failed to process inbound message", loggerdomain.Fields{
			"org_id":  event.OrganizationID,
			"from":    event.From,
			"msg_sid": event.MessageSID,
			"error":   err.Error(),
		})
		return fmt.Errorf("failed to process inbound message: %w", err)
	}

	l.logger.Info("inbound message processed", loggerdomain.Fields{
		"org_id":  event.OrganizationID,
		"from":    event.From,
		"msg_sid": event.MessageSID,
		"type":    event.MessageType,
	})

	return nil
}
