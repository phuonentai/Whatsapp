package services

import (
	"context"
	"fmt"

	"github.com/moasq/go-b2b-starter/internal/modules/instagram/domain/events"
	"github.com/moasq/go-b2b-starter/internal/platform/logger"
	loggerdomain "github.com/moasq/go-b2b-starter/internal/platform/logger/domain"
)

type InstagramEchoListener interface {
	HandleMessageEcho(ctx context.Context, event *events.MessageEcho) error
}

type instagramEchoListener struct {
	crmService CRMService
	logger     logger.Logger
}

func NewInstagramEchoListener(crmService CRMService, log logger.Logger) InstagramEchoListener {
	return &instagramEchoListener{
		crmService: crmService,
		logger:     log,
	}
}

func (l *instagramEchoListener) HandleMessageEcho(ctx context.Context, event *events.MessageEcho) error {
	if err := l.crmService.ProcessInstagramEchoMessage(ctx, event); err != nil {
		l.logger.Error("failed to process instagram echo message", loggerdomain.Fields{
			"org_id":       event.OrganizationID,
			"from_ig_user": event.FromIGUserID,
			"msg_sid":      event.MessageSID,
			"error":        err.Error(),
		})
		return fmt.Errorf("failed to process instagram echo message: %w", err)
	}

	l.logger.Info("instagram echo message processed", loggerdomain.Fields{
		"org_id":       event.OrganizationID,
		"from_ig_user": event.FromIGUserID,
		"msg_sid":      event.MessageSID,
		"type":         event.MessageType,
	})

	return nil
}
