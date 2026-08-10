package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/moasq/go-b2b-starter/internal/modules/crm/domain"
	instagramEvents "github.com/moasq/go-b2b-starter/internal/modules/instagram/domain/events"
	"github.com/moasq/go-b2b-starter/internal/modules/whatsapp/domain/events"
	"github.com/moasq/go-b2b-starter/internal/platform/features"
	"github.com/moasq/go-b2b-starter/internal/platform/logger"
	loggerdomain "github.com/moasq/go-b2b-starter/internal/platform/logger/domain"
	"github.com/moasq/go-b2b-starter/internal/platform/outbox"
	"github.com/moasq/go-b2b-starter/pkg/whatsapp"
)

type CRMService interface {
	ProcessInboundMessage(ctx context.Context, event *events.MessageReceived) error
	ProcessEchoMessage(ctx context.Context, event *events.MessageEcho) error
	ProcessInstagramInboundMessage(ctx context.Context, event *instagramEvents.MessageReceived) error
	ProcessInstagramEchoMessage(ctx context.Context, event *instagramEvents.MessageEcho) error
}

type crmService struct {
	contactRepo      domain.ContactRepository
	conversationRepo domain.ConversationRepository
	messageRepo      domain.MessageRepository
	activityRepo     domain.ActivityRepository
	outboxRepo       outbox.Repository
	featureProvider  features.FeatureProvider
	logger           logger.Logger
}

func NewCRMService(
	contactRepo domain.ContactRepository,
	conversationRepo domain.ConversationRepository,
	messageRepo domain.MessageRepository,
	activityRepo domain.ActivityRepository,
	outboxRepo outbox.Repository,
	featureProvider features.FeatureProvider,
	logger logger.Logger,
) CRMService {
	return &crmService{
		contactRepo: contactRepo, conversationRepo: conversationRepo,
		messageRepo: messageRepo, activityRepo: activityRepo,
		outboxRepo: outboxRepo, featureProvider: featureProvider, logger: logger,
	}
}

func (s *crmService) ProcessInboundMessage(ctx context.Context, event *events.MessageReceived) error {
	return s.persistWhatsAppMessage(ctx, event.OrganizationID, event.From, event.WhatsAppTimestamp, event.MessageType, event.Content, event.MessageSID, domain.MessageDirectionInbound, map[string]interface{}{})
}

func (s *crmService) ProcessEchoMessage(ctx context.Context, event *events.MessageEcho) error {
	return s.persistWhatsAppMessage(ctx, event.OrganizationID, event.From, event.WhatsAppTimestamp, event.MessageType, event.Content, event.MessageSID, domain.MessageDirectionOutbound, map[string]interface{}{"origin": "echo"})
}

func (s *crmService) ProcessInstagramInboundMessage(ctx context.Context, event *instagramEvents.MessageReceived) error {
	return s.persistInstagramMessage(ctx, event.OrganizationID, event.FromIGUserID, event.IGTimestamp, event.MessageType, event.Content, event.MessageSID, domain.MessageDirectionInbound, map[string]interface{}{})
}

func (s *crmService) ProcessInstagramEchoMessage(ctx context.Context, event *instagramEvents.MessageEcho) error {
	return s.persistInstagramMessage(ctx, event.OrganizationID, event.FromIGUserID, event.IGTimestamp, event.MessageType, event.Content, event.MessageSID, domain.MessageDirectionOutbound, map[string]interface{}{"origin": "echo"})
}

// enqueueProfileBackfill durably schedules the async username/avatar backfill
// for an Instagram contact. Failures here are non-fatal: the inbox falls back
// to the IG-scoped id as display identity.
func (s *crmService) enqueueProfileBackfill(ctx context.Context, contact *domain.Contact) {
	payload, err := json.Marshal(instagramEvents.NewProfileBackfill(contact.OrganizationID, contact.ID, contact.InstagramUserID))
	if err != nil {
		s.logger.Warn("failed to marshal instagram profile backfill", loggerdomain.Fields{"error": err.Error()})
		return
	}
	orgID := contact.OrganizationID
	if _, err := s.outboxRepo.Insert(ctx, instagramEvents.ProfileBackfillEventType, payload, &orgID); err != nil {
		s.logger.Warn("failed to enqueue instagram profile backfill", loggerdomain.Fields{"error": err.Error()})
	}
}

// persistWhatsAppMessage keeps the legacy phone-first path: contact upsert by
// E.164 phone, conversation per (org, contact, whatsapp channel).
func (s *crmService) persistWhatsAppMessage(ctx context.Context, orgID int32, from string, ts time.Time, messageType, content, messageSID string, direction domain.MessageDirection, messageData map[string]interface{}) error {
	phone, err := whatsapp.CanonicalizeE164(from)
	if err != nil {
		s.logger.Warn("canonicalizacion de telefono", loggerdomain.Fields{
			"original": from, "resultado": phone, "error": err.Error(),
		})
	}

	contact := &domain.Contact{
		OrganizationID: orgID, PhoneNumber: phone,
		DisplayName: phone, LastMessageAt: &ts, Source: domain.ContactSourceWhatsApp,
	}
	createdContact, err := s.contactRepo.UpsertByPhone(ctx, contact)
	if err != nil {
		return fmt.Errorf("error al crear contacto: %w", err)
	}

	conv := s.conversation(ctx, orgID, createdContact.ID, domain.ChannelWhatsapp, &ts)
	if err := s.persist(ctx, orgID, conv, createdContact.ID, domain.ChannelWhatsapp, ts, messageType, content, messageSID, direction, messageData, phone, domain.ActivityTypeWhatsAppMessage); err != nil {
		return err
	}
	return nil
}

// persistInstagramMessage uses the IG-scoped user id as the contact key and
// instagram channel conversations. Username/avatar are backfilled
// asynchronously by the profile backfill listener.
func (s *crmService) persistInstagramMessage(ctx context.Context, orgID int32, fromIGUserID string, ts time.Time, messageType, content, messageSID string, direction domain.MessageDirection, messageData map[string]interface{}) error {
	if fromIGUserID == "" {
		return fmt.Errorf("instagram user id is required")
	}

	contact := &domain.Contact{
		OrganizationID:  orgID,
		InstagramUserID: fromIGUserID,
		DisplayName:     fromIGUserID,
		LastMessageAt:   &ts,
		Source:          domain.ContactSourceInstagram,
	}
	createdContact, err := s.contactRepo.UpsertByIGUser(ctx, contact)
	if err != nil {
		return fmt.Errorf("error al crear contacto de instagram: %w", err)
	}

	// Username/avatar are only knowable via the Graph API: enqueue a durable
	// backfill when the contact has no username yet (new or previously failed).
	if createdContact.InstagramUsername == "" {
		s.enqueueProfileBackfill(ctx, createdContact)
	}

	conv := s.conversation(ctx, orgID, createdContact.ID, domain.ChannelInstagram, &ts)
	if err := s.persist(ctx, orgID, conv, createdContact.ID, domain.ChannelInstagram, ts, messageType, content, messageSID, direction, messageData, fromIGUserID, domain.ActivityTypeInstagramMessage); err != nil {
		return err
	}
	return nil
}

func (s *crmService) conversation(ctx context.Context, orgID, contactID int32, channel string, ts *time.Time) *domain.Conversation {
	conv := &domain.Conversation{
		OrganizationID: orgID, ContactID: contactID, Channel: channel,
		Status: domain.ConversationStatusActive, LastMessageAt: ts,
	}
	return conv
}

func (s *crmService) persist(
	ctx context.Context,
	orgID int32,
	conv *domain.Conversation,
	contactID int32,
	channel string,
	ts time.Time,
	messageType, content, messageSID string,
	direction domain.MessageDirection,
	messageData map[string]interface{},
	subjectID string,
	activityType domain.ActivityType,
) error {
	resolved, err := s.conversationRepo.EnsureActive(ctx, conv)
	if err != nil {
		return fmt.Errorf("error al obtener/crear conversacion activa: %w", err)
	}
	conv = resolved
	if conv.LastMessageAt == nil || ts.After(*conv.LastMessageAt) {
		if updatedConv, updateErr := s.conversationRepo.UpdateLastMessageAt(ctx, orgID, conv.ID, &ts); updateErr == nil {
			conv = updatedConv
		}
	}

	msg := &domain.Message{
		OrganizationID:    orgID,
		ConversationID:    conv.ID,
		ContactID:         contactID,
		Channel:           channel,
		ProviderMessageID: messageSID,
		Direction:         direction,
		MessageType:       domain.MessageType(messageType),
		Content:           content,
		Status:            "received",
		ChatTimestamp:     &ts,
		MessageData:       messageData,
	}
	createdMsg, inserted, err := s.messageRepo.InsertIdempotent(ctx, msg)
	if err != nil {
		return fmt.Errorf("error al guardar mensaje: %w", err)
	}
	if !inserted {
		s.logger.Debug("mensaje duplicado ignorado", loggerdomain.Fields{
			"provider_message_id": messageSID,
			"channel":             channel,
		})
		return nil
	}

	entitlement, _ := s.featureProvider.GetEntitlement(ctx, orgID)
	if entitlement != nil && entitlement.Features["crm_activities"] {
		activityContent := content
		if len(activityContent) > 500 {
			activityContent = activityContent[:500]
		}
		subject := fmt.Sprintf("Mensaje de WhatsApp de %s", subjectID)
		if channel == domain.ChannelInstagram {
			subject = fmt.Sprintf("Mensaje de Instagram de %s", subjectID)
		}
		_, actErr := s.activityRepo.Create(ctx, &domain.Activity{
			OrganizationID: orgID, ContactID: &contactID,
			ConversationID: &conv.ID, Tipo: activityType,
			Asunto:    subject,
			Contenido: activityContent, RealizadaEn: ts,
			Metadata: map[string]interface{}{
				"message_id": createdMsg.ID,
				"direction":  string(direction),
				"channel":    channel,
			},
		})
		if actErr != nil {
			s.logger.Warn("error al crear actividad de mensaje", loggerdomain.Fields{"error": actErr.Error()})
		}
	}
	return nil
}
