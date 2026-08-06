package services

import (
	"context"
	"fmt"

	"github.com/moasq/go-b2b-starter/internal/modules/crm/domain"
	"github.com/moasq/go-b2b-starter/internal/modules/whatsapp/domain/events"
	"github.com/moasq/go-b2b-starter/internal/platform/features"
	"github.com/moasq/go-b2b-starter/internal/platform/logger"
	loggerdomain "github.com/moasq/go-b2b-starter/internal/platform/logger/domain"
	"github.com/moasq/go-b2b-starter/pkg/whatsapp"
)

type CRMService interface {
	ProcessInboundMessage(ctx context.Context, event *events.MessageReceived) error
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
	phone, err := whatsapp.CanonicalizeE164(event.From)
	if err != nil {
		s.logger.Warn("canonicalizacion de telefono", loggerdomain.Fields{
			"original": event.From, "resultado": phone, "error": err.Error(),
		})
	}

	contact := &domain.Contact{
		OrganizationID: event.OrganizationID, PhoneNumber: phone,
		DisplayName: phone, LastMessageAt: &event.WhatsAppTimestamp,
	}
	createdContact, err := s.contactRepo.UpsertByPhone(ctx, contact)
	if err != nil {
		return fmt.Errorf("error al crear contacto: %w", err)
	}

	conv, err := s.conversationRepo.GetActiveByContact(ctx, event.OrganizationID, createdContact.ID)
	if err != nil {
		conv = &domain.Conversation{
			OrganizationID: event.OrganizationID, ContactID: createdContact.ID,
			Status: domain.ConversationStatusActive, LastMessageAt: &event.WhatsAppTimestamp,
		}
		createdConv, createErr := s.conversationRepo.Create(ctx, conv)
		if createErr != nil { return fmt.Errorf("error al crear conversacion: %w", err) }
		conv = createdConv
	} else if conv.LastMessageAt == nil || event.WhatsAppTimestamp.After(*conv.LastMessageAt) {
		if updatedConv, updateErr := s.conversationRepo.UpdateLastMessageAt(ctx, event.OrganizationID, conv.ID, &event.WhatsAppTimestamp); updateErr == nil {
			conv = updatedConv
		}
	}

	existing, err := s.messageRepo.GetByWhatsAppID(ctx, event.OrganizationID, event.MessageSID)
	if err == nil && existing != nil {
		s.logger.Debug("mensaje duplicado ignorado", loggerdomain.Fields{"whatsapp_message_id": event.MessageSID})
		return nil
	}

	msg := &domain.Message{
		OrganizationID: event.OrganizationID, ConversationID: conv.ID,
		ContactID: createdContact.ID, WhatsAppMessageID: event.MessageSID,
		Direction: domain.MessageDirectionInbound, MessageType: domain.MessageType(event.MessageType),
		Content: event.Content, Status: "received", ChatTimestamp: &event.WhatsAppTimestamp,
	}
	if _, err = s.messageRepo.Create(ctx, msg); err != nil {
		return fmt.Errorf("error al guardar mensaje: %w", err)
	}

	entitlement, _ := s.featureProvider.GetEntitlement(ctx, event.OrganizationID)
	if entitlement != nil && entitlement.Features["crm_activities"] {
		content := event.Content
		if len(content) > 500 { content = content[:500] }
		_, actErr := s.activityRepo.Create(ctx, &domain.Activity{
			OrganizationID: event.OrganizationID, ContactID: &createdContact.ID,
			ConversationID: &conv.ID, Tipo: domain.ActivityTypeWhatsAppMessage,
			Asunto: fmt.Sprintf("Mensaje de WhatsApp de %s", phone),
			Contenido: content, RealizadaEn: event.WhatsAppTimestamp,
			Metadata: map[string]interface{}{"message_id": msg.ID, "direction": "inbound"},
		})
		if actErr != nil {
			s.logger.Warn("error al crear actividad de WhatsApp", loggerdomain.Fields{"error": actErr.Error()})
		}
	}
	return nil
}
