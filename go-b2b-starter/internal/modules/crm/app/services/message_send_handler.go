package services

import (
	"context"
	"fmt"
	"os"

	"github.com/moasq/go-b2b-starter/internal/modules/crm/domain"
	igDomain "github.com/moasq/go-b2b-starter/internal/modules/instagram/domain"
	"github.com/moasq/go-b2b-starter/internal/modules/instagram/domain/events"
	igGraphapi "github.com/moasq/go-b2b-starter/internal/modules/instagram/infra/graphapi"
	whatsappDomain "github.com/moasq/go-b2b-starter/internal/modules/whatsapp/domain"
	whatsappEvents "github.com/moasq/go-b2b-starter/internal/modules/whatsapp/domain/events"
	"github.com/moasq/go-b2b-starter/internal/platform/logger"
	loggerdomain "github.com/moasq/go-b2b-starter/internal/platform/logger/domain"
	"github.com/moasq/go-b2b-starter/pkg/whatsapp"
)

// textSender is the outbound Meta seam used by the send handler.
type textSender interface {
	SendTextMessage(ctx context.Context, accessToken, graphAPIURL, apiVersion, phoneNumberID, to, body string) (string, error)
}

// MessageSendHandler performs durable outbound sends for WhatsApp and
// Instagram. It subscribes to whatsapp.message.send and instagram.message.send
// events dispatched by the outbox; transient failures bubble up so the
// dispatcher retries with backoff and eventually dead-letters. Permanent
// failures (inactive config, missing token) are terminal: the message is
// marked failed and the event completes.
type MessageSendHandler struct {
	msgRepo        domain.MessageRepository
	whatsappRepo   whatsappDomain.ConfigRepository
	igConfigRepo   igDomain.ConfigRepository
	igClient       igGraphapi.IGClient
	whatsappSender textSender
	logger         logger.Logger
}

func NewMessageSendHandler(
	msgRepo domain.MessageRepository,
	whatsappRepo whatsappDomain.ConfigRepository,
	igConfigRepo igDomain.ConfigRepository,
	igClient igGraphapi.IGClient,
	log logger.Logger,
) *MessageSendHandler {
	return &MessageSendHandler{
		msgRepo:        msgRepo,
		whatsappRepo:   whatsappRepo,
		igConfigRepo:   igConfigRepo,
		igClient:       igClient,
		whatsappSender: whatsapp.NewClientWithBreaker(5, 30*1000*1000*1000, 2),
		logger:         log,
	}
}

// Handle dispatches by event type: WhatsApp sends use the Cloud API, Instagram
// sends use the Instagram Messaging API.
func (h *MessageSendHandler) Handle(ctx context.Context, event interface{}) error {
	switch send := event.(type) {
	case *whatsappEvents.MessageSend:
		return h.handleWhatsApp(ctx, send)
	case *events.MessageSend:
		return h.handleInstagram(ctx, send)
	default:
		return fmt.Errorf("unexpected event type %T", event)
	}
}

func (h *MessageSendHandler) handleWhatsApp(ctx context.Context, send *whatsappEvents.MessageSend) error {
	whatsConfig, err := h.whatsappRepo.GetByOrganizationID(ctx, send.OrganizationID)
	if err != nil {
		return fmt.Errorf("resolve whatsapp config: %w", err)
	}
	if !whatsConfig.IsActive {
		return h.fail(ctx, send.MessageID, "whatsapp_not_configured: config is inactive")
	}
	if whatsConfig.AccessToken == "" {
		return h.fail(ctx, send.MessageID, "whatsapp_no_access_token: access token is missing")
	}

	apiVersion := whatsConfig.APIVersion
	if apiVersion == "" {
		apiVersion = "v21.0"
	}
	graphAPIURL := whatsConfig.GraphAPIURL
	if graphAPIURL == "" {
		graphAPIURL = "https://graph.facebook.com"
	}

	if os.Getenv("AUTH_MOCK_ENABLED") == "true" {
		// Mock-auth e2e mode: never call the real Meta Graph API.
		return h.sent(ctx, send.MessageID, fmt.Sprintf("wamid.mock.outbound.%d", send.MessageID))
	}

	msgID, err := h.whatsappSender.SendTextMessage(
		ctx,
		whatsConfig.AccessToken,
		graphAPIURL,
		apiVersion,
		whatsConfig.PhoneNumberID,
		send.To,
		send.Content,
	)
	if err != nil {
		// Return the error so the outbox dispatcher retries with backoff and
		// dead-letters after max attempts; record the failure on the message.
		_ = h.fail(ctx, send.MessageID, err.Error())
		return fmt.Errorf("send whatsapp message: %w", err)
	}

	return h.sent(ctx, send.MessageID, msgID)
}

func (h *MessageSendHandler) handleInstagram(ctx context.Context, send *events.MessageSend) error {
	config, err := h.igConfigRepo.GetByOrganizationID(ctx, send.OrganizationID)
	if err != nil {
		return fmt.Errorf("resolve instagram config: %w", err)
	}
	if !config.IsActive {
		return h.fail(ctx, send.MessageID, "instagram_not_configured: config is inactive")
	}
	if config.AccessToken == "" {
		return h.fail(ctx, send.MessageID, "instagram_no_access_token: access token is missing")
	}

	if os.Getenv("AUTH_MOCK_ENABLED") == "true" {
		// Mock-auth e2e mode: never call the real Meta Graph API.
		return h.sent(ctx, send.MessageID, fmt.Sprintf("mid.mock.outbound.%d", send.MessageID))
	}

	msgID, err := h.igClient.SendTextMessage(
		ctx,
		config.AccessToken,
		config.GraphAPIURL,
		config.APIVersion,
		config.IGUserID,
		send.ToIGUserID,
		send.Content,
	)
	if err != nil {
		_ = h.fail(ctx, send.MessageID, err.Error())
		return fmt.Errorf("send instagram message: %w", err)
	}

	return h.sent(ctx, send.MessageID, msgID)
}

func (h *MessageSendHandler) sent(ctx context.Context, messageID int32, msgID string) error {
	updated, err := h.msgRepo.UpdateStatus(ctx, messageID, "sent", msgID)
	if err != nil {
		return fmt.Errorf("persist sent status: %w", err)
	}
	h.logger.Info("outbound message sent", loggerdomain.Fields{
		"message_id": updated.ID,
		"msg_id":     msgID,
		"channel":    updated.Channel,
	})
	return nil
}

func (h *MessageSendHandler) fail(ctx context.Context, messageID int32, reason string) error {
	_, err := h.msgRepo.UpdateStatus(ctx, messageID, "failed", "")
	if err != nil {
		return fmt.Errorf("persist failed status: %w", err)
	}
	h.logger.Warn("outbound message send failed", loggerdomain.Fields{
		"message_id": messageID,
		"error":      reason,
	})
	return nil
}
