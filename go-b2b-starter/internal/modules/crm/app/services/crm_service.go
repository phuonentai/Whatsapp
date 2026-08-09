package services

import (
	"context"
	"fmt"
	"time"

	"github.com/moasq/go-b2b-starter/internal/modules/crm/domain"
	"github.com/moasq/go-b2b-starter/internal/modules/whatsapp/domain/events"
	"github.com/moasq/go-b2b-starter/internal/platform/features"
	"github.com/moasq/go-b2b-starter/internal/platform/logger"
	loggerdomain "github.com/moasq/go-b2b-starter/internal/platform/logger/domain"
	"github.com/moasq/go-b2b-starter/pkg/whatsapp"
)

type CRMService interface {
	ProcessInboundMessage(ctx context.Context, event *events.MessageReceived) error
	ProcessEchoMessage(ctx context.Context, event *events.MessageEcho) error
}

type crmService struct {
	contactRepo      domain.ContactRepository
	conversationRepo domain.ConversationRepository
	messageRepo      domain.MessageRepository
	activityRepo     domain.ActivityRepository
	featureProvider  features.FeatureProvider
	logger           logger.Logger
}

func NewCRMService(
	contactRepo domain.ContactRepository,
	conversationRepo domain.ConversationRepository,
	messageRepo domain.MessageRepository,
	activityRepo domain.ActivityRepository,
	featureProvider features.FeatureProvider,
	logger logger.Logger,
) CRMService {
	return &crmService{
		contactRepo: contactRepo, conversationRepo: conversationRepo,
		messageRepo: messageRepo, activityRepo: activityRepo,
		featureProvider: featureProvider, logger: logger,
	}
}

func (s *crmService) ProcessInboundMessage(ctx context.Context, event *events.MessageReceived) error {
	return s.persistMessage(ctx, event.OrganizationID, event.From, event.WhatsAppTimestamp, event.MessageType, event.Content, event.MessageSID, domain.MessageDirectionInbound, map[string]interface{}{})
}

func (s *crmService) ProcessEchoMessage(ctx context.Context, event *events.MessageEcho) error {
	return s.persistMessage(ctx, event.OrganizationID, event.From, event.WhatsAppTimestamp, event.MessageType, event.Content, event.MessageSID, domain.MessageDirectionOutbound, map[string]interface{}{"origin": "echo"})
}

func (s *crmService) persistMessage(ctx context.Context, orgID int32, from string, ts time.Time, messageType, content, messageSID string, direction domain.MessageDirection, messageData map[string]interface{}) error {
	phone, err := whatsapp.CanonicalizeE164(from)
	if err != nil {
		s.logger.Warn("canonicalizacion de telefono", loggerdomain.Fields{
			"original": from, "resultado": phone, "error": err.Error(),
		})
	}

	contact := &domain.Contact{
		OrganizationID: orgID, PhoneNumber: phone,
		DisplayName: phone, LastMessageAt: &ts,
	}
	createdContact, err := s.contactRepo.UpsertByPhone(ctx, contact)
	if err != nil {
		return fmt.Errorf("error al crear contacto: %w", err)
	}

	conv := &domain.Conversation{
		OrganizationID: orgID, ContactID: createdContact.ID,
		Status: domain.ConversationStatusActive, LastMessageAt: &ts,
	}
	conv, err = s.conversationRepo.EnsureActive(ctx, conv)
	if err != nil {
		return fmt.Errorf("error al obtener/crear conversacion activa: %w", err)
	}
	if conv.LastMessageAt == nil || ts.After(*conv.LastMessageAt) {
		if updatedConv, updateErr := s.conversationRepo.UpdateLastMessageAt(ctx, orgID, conv.ID, &ts); updateErr == nil {
			conv = updatedConv
		}
	}

	msg := &domain.Message{
		OrganizationID: orgID, ConversationID: conv.ID,
		ContactID: createdContact.ID, WhatsAppMessageID: messageSID,
		Direction: direction, MessageType: domain.MessageType(messageType),
		Content: content, Status: "received", ChatTimestamp: &ts,
		MessageData: messageData,
	}
	createdMsg, inserted, err := s.messageRepo.InsertIdempotent(ctx, msg)
	if err != nil {
		return fmt.Errorf("error al guardar mensaje: %w", err)
	}
	if !inserted {
		s.logger.Debug("mensaje duplicado ignorado", loggerdomain.Fields{"whatsapp_message_id": messageSID})
		return nil
	}

	entitlement, _ := s.featureProvider.GetEntitlement(ctx, orgID)
	if entitlement != nil && entitlement.Features["crm_activities"] {
		activityContent := content
		if len(activityContent) > 500 {
			activityContent = activityContent[:500]
		}
		_, actErr := s.activityRepo.Create(ctx, &domain.Activity{
			OrganizationID: orgID, ContactID: &createdContact.ID,
			ConversationID: &conv.ID, Tipo: domain.ActivityTypeWhatsAppMessage,
			Asunto:    fmt.Sprintf("Mensaje de WhatsApp de %s", phone),
			Contenido: activityContent, RealizadaEn: ts,
			Metadata: map[string]interface{}{"message_id": createdMsg.ID, "direction": string(direction)},
		})
		if actErr != nil {
			s.logger.Warn("error al crear actividad de WhatsApp", loggerdomain.Fields{"error": actErr.Error()})
		}
	}
	return nil
}
