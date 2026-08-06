package services

import (
	"context"
	"fmt"

	"github.com/moasq/go-b2b-starter/internal/modules/crm/domain"
	whatsappDomain "github.com/moasq/go-b2b-starter/internal/modules/whatsapp/domain"
	"github.com/moasq/go-b2b-starter/pkg/whatsapp"
)

type OutboundService interface {
	SendMessage(ctx context.Context, orgID, convID int32, content string) (*domain.Message, error)
}

type outboundService struct {
	convRepo      domain.ConversationRepository
	contactRepo   domain.ContactRepository
	msgRepo       domain.MessageRepository
	whatsappRepo  whatsappDomain.ConfigRepository
	whatsappClient *whatsapp.ClientWithBreaker
}

func NewOutboundService(
	convRepo domain.ConversationRepository,
	contactRepo domain.ContactRepository,
	msgRepo domain.MessageRepository,
	whatsappRepo whatsappDomain.ConfigRepository,
) OutboundService {
	return &outboundService{
		convRepo:       convRepo,
		contactRepo:    contactRepo,
		msgRepo:        msgRepo,
		whatsappRepo:   whatsappRepo,
		whatsappClient: whatsapp.NewClientWithBreaker(5, 30*1000*1000*1000, 2),
	}
}

func (s *outboundService) SendMessage(ctx context.Context, orgID, convID int32, content string) (*domain.Message, error) {
	if content == "" {
		return nil, fmt.Errorf("message content is required")
	}

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

	conv, err := s.convRepo.GetByID(ctx, orgID, convID)
	if err != nil {
		return nil, fmt.Errorf("conversation not found: %w", err)
	}

	contact, err := s.contactRepo.GetByID(ctx, orgID, conv.ContactID)
	if err != nil {
		return nil, fmt.Errorf("contact not found: %w", err)
	}

	apiVersion := whatsConfig.APIVersion
	if apiVersion == "" {
		apiVersion = "v21.0"
	}
	graphAPIURL := whatsConfig.GraphAPIURL
	if graphAPIURL == "" {
		graphAPIURL = "https://graph.facebook.com"
	}

	msgID, err := s.whatsappClient.SendTextMessage(
		ctx,
		whatsConfig.AccessToken,
		graphAPIURL,
		apiVersion,
		whatsConfig.PhoneNumberID,
		contact.PhoneNumber,
		content,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to send whatsapp message: %w", err)
	}

	msg := &domain.Message{
		OrganizationID:    orgID,
		ConversationID:    conv.ID,
		ContactID:         conv.ContactID,
		WhatsAppMessageID: msgID,
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
