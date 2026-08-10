package services

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/moasq/go-b2b-starter/internal/modules/crm/domain"
	igDomain "github.com/moasq/go-b2b-starter/internal/modules/instagram/domain"
	igEvents "github.com/moasq/go-b2b-starter/internal/modules/instagram/domain/events"
	whatsappDomain "github.com/moasq/go-b2b-starter/internal/modules/whatsapp/domain"
	"github.com/moasq/go-b2b-starter/internal/modules/whatsapp/domain/events"
	"github.com/moasq/go-b2b-starter/internal/platform/outbox"
)

type OutboundService interface {
	SendMessage(ctx context.Context, orgID, convID int32, content string) (*domain.Message, error)
}

type outboundService struct {
	convRepo     domain.ConversationRepository
	contactRepo  domain.ContactRepository
	msgRepo      domain.MessageRepository
	whatsappRepo whatsappDomain.ConfigRepository
	igConfigRepo igDomain.ConfigRepository
	outboxRepo   outbox.Repository
}

func NewOutboundService(
	convRepo domain.ConversationRepository,
	contactRepo domain.ContactRepository,
	msgRepo domain.MessageRepository,
	whatsappRepo whatsappDomain.ConfigRepository,
	igConfigRepo igDomain.ConfigRepository,
	outboxRepo outbox.Repository,
) OutboundService {
	return &outboundService{
		convRepo:     convRepo,
		contactRepo:  contactRepo,
		msgRepo:      msgRepo,
		whatsappRepo: whatsappRepo,
		igConfigRepo: igConfigRepo,
		outboxRepo:   outboxRepo,
	}
}

func (s *outboundService) SendMessage(ctx context.Context, orgID, convID int32, content string) (*domain.Message, error) {
	if content == "" {
		return nil, fmt.Errorf("message content is required")
	}

	conv, err := s.convRepo.GetByID(ctx, orgID, convID)
	if err != nil {
		return nil, fmt.Errorf("conversation not found: %w", err)
	}

	channel := conv.Channel
	if channel == "" {
		channel = domain.ChannelWhatsapp
	}

	contact, err := s.contactRepo.GetByID(ctx, orgID, conv.ContactID)
	if err != nil {
		return nil, fmt.Errorf("contact not found: %w", err)
	}

	switch channel {
	case domain.ChannelInstagram:
		return s.sendInstagram(ctx, orgID, conv, contact, content)
	default:
		return s.sendWhatsApp(ctx, orgID, conv, contact, content)
	}
}

func (s *outboundService) sendWhatsApp(ctx context.Context, orgID int32, conv *domain.Conversation, contact *domain.Contact, content string) (*domain.Message, error) {
	whatsConfig, err := s.whatsappRepo.GetByOrganizationID(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("whatsapp not configured: %w", err)
	}
	if !whatsConfig.IsActive {
		return nil, fmt.Errorf("whatsapp_not_configured: config is inactive")
	}
	if whatsConfig.AccessToken == "" {
		return nil, fmt.Errorf("whatsapp_no_access_token: access token is missing")
	}

	var msgID string
	if os.Getenv("AUTH_MOCK_ENABLED") == "true" {
		// Mock-auth e2e mode: never call the real Meta Graph API. Synthesize a
		// message id so the outbound reply flow works fully offline.
		msgID = fmt.Sprintf("wamid.mock.outbound.%d", time.Now().UnixNano())
	} else {
		// Production: enqueue a durable send request instead of calling Meta
		// synchronously. The outbox dispatcher delivers it with retry/backoff
		// and the send survives process restarts.
		msg := &domain.Message{
			OrganizationID: orgID,
			ConversationID: conv.ID,
			ContactID:      conv.ContactID,
			Channel:        domain.ChannelWhatsapp,
			Direction:      domain.MessageDirectionOutbound,
			MessageType:    domain.MessageTypeText,
			Content:        content,
			Status:         "queued",
		}

		created, err := s.msgRepo.Create(ctx, msg)
		if err != nil {
			return nil, fmt.Errorf("failed to persist queued outbound message: %w", err)
		}

		payload, err := json.Marshal(events.NewMessageSend(orgID, conv.ID, created.ID, contact.PhoneNumber, content))
		if err != nil {
			_, _ = s.msgRepo.UpdateStatus(ctx, created.ID, "failed", "")
			return nil, fmt.Errorf("failed to marshal send event: %w", err)
		}
		if _, err := s.outboxRepo.Insert(ctx, events.MessageSendEventType, payload, &orgID); err != nil {
			_, _ = s.msgRepo.UpdateStatus(ctx, created.ID, "failed", "")
			return nil, fmt.Errorf("failed to enqueue outbound message: %w", err)
		}

		return created, nil
	}

	msg := &domain.Message{
		OrganizationID:    orgID,
		ConversationID:    conv.ID,
		ContactID:         conv.ContactID,
		Channel:           domain.ChannelWhatsapp,
		ProviderMessageID: msgID,
		Direction:         domain.MessageDirectionOutbound,
		MessageType:       domain.MessageTypeText,
		Content:           content,
		Status:            "sent",
	}

	created, err := s.msgRepo.Create(ctx, msg)
	if err != nil {
		return nil, fmt.Errorf("failed to persist outbound message: %w", err)
	}

	return created, nil
}

func (s *outboundService) sendInstagram(ctx context.Context, orgID int32, conv *domain.Conversation, contact *domain.Contact, content string) (*domain.Message, error) {
	config, err := s.igConfigRepo.GetByOrganizationID(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("instagram not configured: %w", err)
	}
	if !config.IsActive {
		return nil, fmt.Errorf("instagram_not_configured: config is inactive")
	}
	if config.AccessToken == "" {
		return nil, fmt.Errorf("instagram_no_access_token: access token is missing")
	}

	if contact.InstagramUserID == "" {
		return nil, fmt.Errorf("instagram contact has no IG user id")
	}

	var msgID string
	if os.Getenv("AUTH_MOCK_ENABLED") == "true" {
		msgID = fmt.Sprintf("mid.mock.outbound.%d", time.Now().UnixNano())
	} else {
		msg := &domain.Message{
			OrganizationID: orgID,
			ConversationID: conv.ID,
			ContactID:      conv.ContactID,
			Channel:        domain.ChannelInstagram,
			Direction:      domain.MessageDirectionOutbound,
			MessageType:    domain.MessageTypeText,
			Content:        content,
			Status:         "queued",
		}

		created, err := s.msgRepo.Create(ctx, msg)
		if err != nil {
			return nil, fmt.Errorf("failed to persist queued instagram outbound message: %w", err)
		}

		payload, err := json.Marshal(igEvents.NewMessageSend(orgID, conv.ID, created.ID, contact.InstagramUserID, content))
		if err != nil {
			_, _ = s.msgRepo.UpdateStatus(ctx, created.ID, "failed", "")
			return nil, fmt.Errorf("failed to marshal instagram send event: %w", err)
		}
		if _, err := s.outboxRepo.Insert(ctx, igEvents.MessageSendEventType, payload, &orgID); err != nil {
			_, _ = s.msgRepo.UpdateStatus(ctx, created.ID, "failed", "")
			return nil, fmt.Errorf("failed to enqueue instagram outbound message: %w", err)
		}

		return created, nil
	}

	msg := &domain.Message{
		OrganizationID:    orgID,
		ConversationID:    conv.ID,
		ContactID:         conv.ContactID,
		Channel:           domain.ChannelInstagram,
		ProviderMessageID: msgID,
		Direction:         domain.MessageDirectionOutbound,
		MessageType:       domain.MessageTypeText,
		Content:           content,
		Status:            "sent",
	}

	created, err := s.msgRepo.Create(ctx, msg)
	if err != nil {
		return nil, fmt.Errorf("failed to persist instagram outbound message: %w", err)
	}

	return created, nil
}
