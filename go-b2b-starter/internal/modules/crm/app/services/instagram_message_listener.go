package services

import (
	"context"
	"fmt"

	"github.com/moasq/go-b2b-starter/internal/modules/instagram/domain/events"
	"github.com/moasq/go-b2b-starter/internal/platform/logger"
	loggerdomain "github.com/moasq/go-b2b-starter/internal/platform/logger/domain"
)

type InstagramMessageListener interface {
	HandleMessageReceived(ctx context.Context, event *events.MessageReceived) error
}

type instagramMessageListener struct {
	crmService CRMService
	logger     logger.Logger
}

func NewInstagramMessageListener(crmService CRMService, log logger.Logger) InstagramMessageListener {
	return &instagramMessageListener{
		crmService: crmService,
		logger:     log,
	}
}

func (l *instagramMessageListener) HandleMessageReceived(ctx context.Context, event *events.MessageReceived) error {
	if err := l.crmService.ProcessInstagramInboundMessage(ctx, event); err != nil {
		l.logger.Error("failed to process instagram inbound message", loggerdomain.Fields{
			"org_id":       event.OrganizationID,
			"from_ig_user": event.FromIGUserID,
			"msg_sid":      event.MessageSID,
			"error":        err.Error(),
		})
		return fmt.Errorf("failed to process instagram inbound message: %w", err)
	}

	l.logger.Info("instagram inbound message processed", loggerdomain.Fields{
		"org_id":       event.OrganizationID,
		"from_ig_user": event.FromIGUserID,
		"msg_sid":      event.MessageSID,
		"type":         event.MessageType,
	})

	return nil
}
