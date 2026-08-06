package services

import (
	"context"
	"fmt"
	"time"

	"github.com/moasq/go-b2b-starter/internal/platform/eventbus"
	"github.com/moasq/go-b2b-starter/internal/platform/logger"
	loggerdomain "github.com/moasq/go-b2b-starter/internal/platform/logger/domain"
	whatsappDomain "github.com/moasq/go-b2b-starter/internal/modules/whatsapp/domain"
	"github.com/moasq/go-b2b-starter/internal/modules/whatsapp/domain/events"
	"github.com/moasq/go-b2b-starter/pkg/whatsapp"
)

type WebhookService interface {
	ProcessWebhook(ctx context.Context, rawBody []byte, parsedPayload map[string]any, signatureHeader string) error
	VerifyChallenge(ctx context.Context, mode, verifyToken, challenge string) (string, error)
	GetWebhookLogStats(ctx context.Context, orgID int32) (*whatsappDomain.WebhookLogStats, error)
}

type webhookService struct {
	configRepo whatsappDomain.ConfigRepository
	logRepo    whatsappDomain.WebhookLogRepository
	eventBus   eventbus.EventBus
	logger     logger.Logger
}

func NewWebhookService(
	configRepo whatsappDomain.ConfigRepository,
	logRepo whatsappDomain.WebhookLogRepository,
	eventBus eventbus.EventBus,
	logger logger.Logger,
) WebhookService {
	return &webhookService{
		configRepo: configRepo,
		logRepo:    logRepo,
		eventBus:   eventBus,
		logger:     logger,
	}
}

func (s *webhookService) ProcessWebhook(ctx context.Context, rawBody []byte, parsedPayload map[string]any, signatureHeader string) error {
	phoneNumberID := extractMetadataPhoneNumberID(parsedPayload)
	if phoneNumberID == "" {
		return fmt.Errorf("phone_number_id not found in payload")
	}

	config, err := s.configRepo.GetByPhoneNumberID(ctx, phoneNumberID)
	if err != nil {
		return fmt.Errorf("config not found for phone_number_id %s: %w", phoneNumberID, err)
	}

	if err := whatsapp.VerifySignature(config.WebhookSecret, rawBody, signatureHeader); err != nil {
		return fmt.Errorf("signature verification failed: %w", err)
	}

	webhookLog := &whatsappDomain.WebhookLog{
		OrganizationID: config.OrganizationID,
		Status:         "received",
		EventType:      extractEventType(parsedPayload),
		PhoneNumberID:  phoneNumberID,
		RawBody:        rawBody,
	}

	if _, err := s.logRepo.Insert(ctx, webhookLog); err != nil {
		s.logger.Error("failed to log webhook", loggerdomain.Fields{"error": err.Error()})
	}

	messages := extractMessagesFromPayload(parsedPayload)
	for _, msg := range messages {
		event := events.NewMessageReceived(
			config.OrganizationID,
			msg.MessageSID,
			msg.From,
			config.BusinessPhone,
			msg.MessageType,
			msg.Content,
			msg.MediaURL,
			msg.Timestamp,
			rawBody,
		)

		if err := s.eventBus.Publish(ctx, event); err != nil {
			s.logger.Error("failed to publish message event", loggerdomain.Fields{
				"error":  err.Error(),
				"msg_id": msg.MessageSID,
			})
		}
	}

	return nil
}

func (s *webhookService) GetWebhookLogStats(ctx context.Context, orgID int32) (*whatsappDomain.WebhookLogStats, error) {
	return s.logRepo.GetStatsByOrganization(ctx, orgID)
}

func (s *webhookService) VerifyChallenge(ctx context.Context, mode, verifyToken, challenge string) (string, error) {
	if mode != "subscribe" {
		return "", fmt.Errorf("invalid verification mode: %s", mode)
	}
	if verifyToken == "" {
		return "", fmt.Errorf("verify token is required")
	}
	if challenge == "" {
		return "", fmt.Errorf("challenge is required")
	}

	if _, err := s.configRepo.GetByVerifyToken(ctx, verifyToken); err != nil {
		return "", fmt.Errorf("%w: verify token does not match any active config", whatsappDomain.ErrWebhookVerificationFail)
	}

	return challenge, nil
}

type parsedMessage struct {
	MessageSID  string
	From        string
	MessageType string
	Content     string
	MediaURL    string
	Timestamp   time.Time
}

func extractMetadataPhoneNumberID(payload map[string]any) string {
	entry := firstEntry(payload)
	change := firstChange(entry)
	value := changeValue(change)
	if value == nil {
		return ""
	}
	meta, _ := value["metadata"].(map[string]any)
	if meta == nil {
		return ""
	}
	id, _ := meta["phone_number_id"].(string)
	return id
}

func extractEventType(payload map[string]any) string {
	entry := firstEntry(payload)
	change := firstChange(entry)
	value := changeValue(change)
	if value == nil {
		return ""
	}
	if msgs, ok := value["messages"].([]any); ok && len(msgs) > 0 {
		if msg, ok := msgs[0].(map[string]any); ok {
			t, _ := msg["type"].(string)
			return t
		}
	}
	if _, ok := value["statuses"].([]any); ok {
		return "status"
	}
	return "unknown"
}

func extractMessagesFromPayload(payload map[string]any) []parsedMessage {
	entry := firstEntry(payload)
	change := firstChange(entry)
	value := changeValue(change)
	if value == nil {
		return nil
	}

	rawMessages, ok := value["messages"].([]any)
	if !ok {
		return nil
	}

	var result []parsedMessage
	for _, raw := range rawMessages {
		msg, ok := raw.(map[string]any)
		if !ok {
			continue
		}

		p := parsedMessage{
			MessageSID:  stringField(msg, "id"),
			From:        stringField(msg, "from"),
			MessageType: stringField(msg, "type"),
		}

		if ts := stringField(msg, "timestamp"); ts != "" {
			var sec int64
			if _, err := fmt.Sscanf(ts, "%d", &sec); err == nil {
				p.Timestamp = time.Unix(sec, 0)
			}
		}

		switch p.MessageType {
		case "text":
			if text, ok := msg["text"].(map[string]any); ok {
				p.Content = stringField(text, "body")
			}
		case "image", "video", "audio", "document":
			if media, ok := msg[p.MessageType].(map[string]any); ok {
				p.MediaURL = stringField(media, "link")
				if p.MediaURL == "" {
					p.MediaURL = stringField(media, "id")
				}
				p.Content = stringField(media, "caption")
			}
		case "location":
			if loc, ok := msg["location"].(map[string]any); ok {
				lat, _ := loc["latitude"].(float64)
				lng, _ := loc["longitude"].(float64)
				p.Content = fmt.Sprintf("%f,%f", lat, lng)
			}
		}

		result = append(result, p)
	}
	return result
}

func firstEntry(payload map[string]any) map[string]any {
	if entries, ok := payload["entry"].([]any); ok && len(entries) > 0 {
		if e, ok := entries[0].(map[string]any); ok {
			return e
		}
	}
	return nil
}

func firstChange(entry map[string]any) map[string]any {
	if entry == nil {
		return nil
	}
	if changes, ok := entry["changes"].([]any); ok && len(changes) > 0 {
		if c, ok := changes[0].(map[string]any); ok {
			return c
		}
	}
	return nil
}

func changeValue(change map[string]any) map[string]any {
	if change == nil {
		return nil
	}
	if v, ok := change["value"].(map[string]any); ok {
		return v
	}
	return nil
}

func stringField(m map[string]any, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}
