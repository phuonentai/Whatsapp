package services

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"golang.org/x/time/rate"

	"github.com/moasq/go-b2b-starter/internal/modules/crm/domain"
	"github.com/moasq/go-b2b-starter/internal/modules/crm/domain/conversationscope"
	igDomain "github.com/moasq/go-b2b-starter/internal/modules/instagram/domain"
	igEvents "github.com/moasq/go-b2b-starter/internal/modules/instagram/domain/events"
	whatsappDomain "github.com/moasq/go-b2b-starter/internal/modules/whatsapp/domain"
	"github.com/moasq/go-b2b-starter/internal/modules/whatsapp/domain/events"
	"github.com/moasq/go-b2b-starter/internal/platform/outbox"
)

type OutboundService interface {
	SendMessage(ctx context.Context, orgID, convID int32, content string) (*domain.Message, error)
	// SendTemplateMessage sends a Meta-approved template outside the 24h window,
	// persists the outbound message as sent with the wamid, and enforces the
	// org-level send rate limit (10 messages / 10 seconds).
	SendTemplateMessage(ctx context.Context, orgID, convID int32, templateID int64, params []string) (*domain.Message, error)
}

type outboundService struct {
	convRepo       domain.ConversationRepository
	contactRepo    domain.ContactRepository
	msgRepo        domain.MessageRepository
	whatsappRepo   whatsappDomain.ConfigRepository
	igConfigRepo   igDomain.ConfigRepository
	outboxRepo     outbox.Repository
	templateRepo   whatsappDomain.TemplateRepository
	templateSender templateSender
	// orgLimiters enforces the 10 msgs / 10s outbound rate limit per org.
	orgLimiters map[int32]*rate.Limiter
	limiterMu   sync.Mutex
}

// templateSender is the Cloud API template send seam (pkg/whatsapp client).
type templateSender interface {
	SendTemplateMessage(ctx context.Context, accessToken, graphAPIURL, apiVersion, phoneNumberID, to, templateName, language string, params []string) (string, error)
}

func NewOutboundService(
	convRepo domain.ConversationRepository,
	contactRepo domain.ContactRepository,
	msgRepo domain.MessageRepository,
	whatsappRepo whatsappDomain.ConfigRepository,
	igConfigRepo igDomain.ConfigRepository,
	outboxRepo outbox.Repository,
	templateRepo whatsappDomain.TemplateRepository,
	templateSender templateSender,
) OutboundService {
	return &outboundService{
		convRepo:       convRepo,
		contactRepo:    contactRepo,
		msgRepo:        msgRepo,
		whatsappRepo:   whatsappRepo,
		igConfigRepo:   igConfigRepo,
		outboxRepo:     outboxRepo,
		templateRepo:   templateRepo,
		templateSender: templateSender,
		orgLimiters:    map[int32]*rate.Limiter{},
	}
}

// orgLimiter returns (creating on first use) the per-org send rate limiter:
// 10 messages per 10 seconds, matching the WhatsApp Cloud API throughput cap.
func (s *outboundService) orgLimiter(orgID int32) *rate.Limiter {
	s.limiterMu.Lock()
	defer s.limiterMu.Unlock()
	if l, ok := s.orgLimiters[orgID]; ok {
		return l
	}
	l := rate.NewLimiter(rate.Every(1*time.Second), 10)
	s.orgLimiters[orgID] = l
	return l
}

// rateLimitError is returned when the org exceeds the 10 msgs / 10s cap.
var rateLimitError = fmt.Errorf("rate_limit: exceeded 10 messages per 10 seconds")

// SendTemplateMessage resolves the org-scoped approved template, validates the
// parameter count, enforces the org send rate limit, sends through the Cloud
// API template path (which bypasses the 24-hour window), and persists the
// outbound message as sent.
func (s *outboundService) SendTemplateMessage(ctx context.Context, orgID, convID int32, templateID int64, params []string) (*domain.Message, error) {
	conv, err := s.convRepo.GetByID(ctx, orgID, convID, conversationscope.FromContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("conversation not found: %w", err)
	}

	contact, err := s.contactRepo.GetByID(ctx, orgID, conv.ContactID)
	if err != nil {
		return nil, fmt.Errorf("contact not found: %w", err)
	}

	template, err := s.templateRepo.GetByID(ctx, orgID, templateID)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", whatsappDomain.ErrTemplateNotFound, err)
	}
	if template.Status != whatsappDomain.TemplateStatusApproved || !template.IsActive {
		return nil, whatsappDomain.ErrTemplateNotApproved
	}
	if len(params) != template.ParamCount {
		return nil, fmt.Errorf("%w: se esperaban %d parámetros y se recibieron %d", whatsappDomain.ErrTemplateParamCountMismatch, template.ParamCount, len(params))
	}

	whatsConfig, err := s.whatsappRepo.GetByOrganizationID(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("whatsapp_not_configured: %w", err)
	}
	if !whatsConfig.IsActive {
		return nil, fmt.Errorf("whatsapp_not_configured: config is inactive")
	}
	if whatsConfig.AccessToken == "" {
		return nil, fmt.Errorf("whatsapp_no_access_token: access token is missing")
	}

	if !s.orgLimiter(orgID).Allow() {
		return nil, rateLimitError
	}

	apiVersion := whatsConfig.APIVersion
	if apiVersion == "" {
		apiVersion = "v21.0"
	}
	graphAPIURL := whatsConfig.GraphAPIURL
	if graphAPIURL == "" {
		graphAPIURL = "https://graph.facebook.com"
	}

	var msgID string
	if os.Getenv("AUTH_MOCK_ENABLED") == "true" {
		msgID = fmt.Sprintf("wamid.mock.template.%d", time.Now().UnixNano())
	} else {
		msgID, err = s.templateSender.SendTemplateMessage(
			ctx,
			whatsConfig.AccessToken,
			graphAPIURL,
			apiVersion,
			whatsConfig.PhoneNumberID,
			contact.PhoneNumber,
			template.Name,
			template.Language,
			params,
		)
		if err != nil {
			return nil, fmt.Errorf("whatsapp_api_error: %w", err)
		}
	}

	msg := &domain.Message{
		OrganizationID:    orgID,
		ConversationID:    conv.ID,
		ContactID:         conv.ContactID,
		Channel:           domain.ChannelWhatsapp,
		ProviderMessageID: msgID,
		Direction:         domain.MessageDirectionOutbound,
		MessageType:       domain.MessageTypeText,
		Content:           template.Body,
		Status:            "sent",
	}

	created, err := s.msgRepo.Create(ctx, msg)
	if err != nil {
		return nil, fmt.Errorf("failed to persist template message: %w", err)
	}
	return created, nil
}

func (s *outboundService) SendMessage(ctx context.Context, orgID, convID int32, content string) (*domain.Message, error) {
	if content == "" {
		return nil, fmt.Errorf("message content is required")
	}

	conv, err := s.convRepo.GetByID(ctx, orgID, convID, conversationscope.FromContext(ctx))
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
