package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"time"

	whatsappDomain "github.com/moasq/go-b2b-starter/internal/modules/whatsapp/domain"
	"github.com/moasq/go-b2b-starter/internal/modules/whatsapp/domain/events"
	"github.com/moasq/go-b2b-starter/internal/platform/logger"
	loggerdomain "github.com/moasq/go-b2b-starter/internal/platform/logger/domain"
	"github.com/moasq/go-b2b-starter/internal/platform/outbox"
	"github.com/moasq/go-b2b-starter/pkg/whatsapp"
)

type WebhookService interface {
	ProcessWebhook(ctx context.Context, rawBody []byte, parsedPayload map[string]any, signatureHeader string) error
	VerifyChallenge(ctx context.Context, mode, verifyToken, challenge string) (string, error)
	GetWebhookLogStats(ctx context.Context, orgID int32) (*whatsappDomain.WebhookLogStats, error)
	Replay(ctx context.Context, orgID, logID int32) (int, error)
}

type webhookService struct {
	configRepo whatsappDomain.ConfigRepository
	logRepo    whatsappDomain.WebhookLogRepository
	outboxRepo outbox.Repository
	logger     logger.Logger
}

func NewWebhookService(
	configRepo whatsappDomain.ConfigRepository,
	logRepo whatsappDomain.WebhookLogRepository,
	outboxRepo outbox.Repository,
	logger logger.Logger,
) WebhookService {
	return &webhookService{
		configRepo: configRepo,
		logRepo:    logRepo,
		outboxRepo: outboxRepo,
		logger:     logger,
	}
}

func (s *webhookService) ProcessWebhook(ctx context.Context, rawBody []byte, parsedPayload map[string]any, signatureHeader string) error {
	phoneNumberID := extractMetadataPhoneNumberID(parsedPayload)
	if phoneNumberID == "" {
		s.logFailed(ctx, rawBody, nil, phoneNumberID, "phone_number_id not found in payload")
		return fmt.Errorf("phone_number_id not found in payload")
	}

	config, err := s.configRepo.GetByPhoneNumberID(ctx, phoneNumberID)
	if err != nil {
		s.logFailed(ctx, rawBody, nil, phoneNumberID, "config not found for phone_number_id "+phoneNumberID)
		return fmt.Errorf("%w: config not found for phone_number_id %s", whatsappDomain.ErrUnknownPhoneNumber, phoneNumberID)
	}

	if err := whatsapp.VerifySignature(config.WebhookSecret, rawBody, signatureHeader); err != nil {
		s.logFailed(ctx, rawBody, &config.OrganizationID, phoneNumberID, "invalid signature")
		return fmt.Errorf("%w: %v", whatsappDomain.ErrInvalidSignature, err)
	}

	messages := extractMessagesFromPayload(parsedPayload)
	outboxEvents := s.buildOutboxEvents(config, messages, rawBody)

	webhookLog := &whatsappDomain.WebhookLog{
		OrganizationID: &config.OrganizationID,
		Status:         "received",
		EventType:      extractEventType(parsedPayload),
		PhoneNumberID:  phoneNumberID,
		RawBody:        rawBody,
		DeliveryKey:    computeDeliveryKey(parsedPayload),
	}

	if _, err := s.logRepo.InsertWithOutbox(ctx, webhookLog, outboxEvents); err != nil {
		if errors.Is(err, whatsappDomain.ErrDuplicateDelivery) {
			// At-least-once retry: acknowledge without re-dispatching.
			duplicate := &whatsappDomain.WebhookLog{
				OrganizationID: &config.OrganizationID,
				Status:         "duplicate",
				EventType:      extractEventType(parsedPayload),
				PhoneNumberID:  phoneNumberID,
				RawBody:        rawBody,
				ErrorMessage:   "duplicate delivery acknowledged",
			}
			if _, insErr := s.logRepo.Insert(ctx, duplicate); insErr != nil {
				s.logger.Error("failed to log duplicate delivery", loggerdomain.Fields{"error": insErr.Error()})
			}
			s.logger.Info("duplicate webhook delivery acknowledged", loggerdomain.Fields{
				"phone_number_id": phoneNumberID,
				"delivery_key":    webhookLog.DeliveryKey,
			})
			return nil
		}
		s.logger.Error("failed to persist webhook with outbox events", loggerdomain.Fields{"error": err.Error()})
		return fmt.Errorf("failed to persist webhook delivery: %w", err)
	}

	return nil
}

// buildOutboxEvents serializes parsed messages into durable outbox payloads.
// Dispatch is asynchronous via the outbox dispatcher; nothing runs in the
// webhook request path.
func (s *webhookService) buildOutboxEvents(config *whatsappDomain.WhatsAppConfig, messages []parsedMessage, rawBody []byte) []whatsappDomain.OutboxEventInput {
	eventsOut := make([]whatsappDomain.OutboxEventInput, 0, len(messages))
	for _, msg := range messages {
		if msg.IsEcho {
			echo := events.NewMessageEcho(
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
			payload, err := json.Marshal(echo)
			if err != nil {
				s.logger.Error("failed to marshal echo event", loggerdomain.Fields{"error": err.Error(), "msg_id": msg.MessageSID})
				continue
			}
			eventsOut = append(eventsOut, whatsappDomain.OutboxEventInput{
				EventType: events.MessageEchoEventType,
				Payload:   payload,
			})
			continue
		}

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
		payload, err := json.Marshal(event)
		if err != nil {
			s.logger.Error("failed to marshal message event", loggerdomain.Fields{"error": err.Error(), "msg_id": msg.MessageSID})
			continue
		}
		eventsOut = append(eventsOut, whatsappDomain.OutboxEventInput{
			EventType: events.MessageReceivedEventType,
			Payload:   payload,
		})
	}
	return eventsOut
}

// Replay re-enqueues the events of a stored webhook log from its raw payload.
// Returns the number of events re-enqueued.
func (s *webhookService) Replay(ctx context.Context, orgID, logID int32) (int, error) {
	log, err := s.logRepo.GetByID(ctx, logID)
	if err != nil {
		return 0, err
	}
	if log.OrganizationID == nil || *log.OrganizationID != orgID {
		return 0, whatsappDomain.ErrWebhookLogNotFound
	}

	config, err := s.configRepo.GetByOrganizationID(ctx, orgID)
	if err != nil {
		return 0, fmt.Errorf("failed to resolve config for replay: %w", err)
	}

	var payload map[string]any
	if err := json.Unmarshal(log.RawBody, &payload); err != nil {
		return 0, fmt.Errorf("failed to parse stored payload for replay: %w", err)
	}

	messages := extractMessagesFromPayload(payload)
	eventsOut := s.buildOutboxEvents(config, messages, log.RawBody)

	for _, ev := range eventsOut {
		if _, err := s.outboxRepo.Insert(ctx, ev.EventType, ev.Payload, log.OrganizationID); err != nil {
			return 0, fmt.Errorf("failed to enqueue replay event %q: %w", ev.EventType, err)
		}
	}

	if _, err := s.logRepo.Insert(ctx, &whatsappDomain.WebhookLog{
		OrganizationID: log.OrganizationID,
		Status:         "replay",
		EventType:      log.EventType,
		PhoneNumberID:  log.PhoneNumberID,
		RawBody:        log.RawBody,
		ErrorMessage:   fmt.Sprintf("replayed from log %d", logID),
	}); err != nil {
		s.logger.Warn("failed to record replay action", loggerdomain.Fields{"error": err.Error(), "log_id": logID})
	}

	return len(eventsOut), nil
}

func (s *webhookService) logFailed(ctx context.Context, rawBody []byte, orgID *int32, phoneNumberID, reason string) {
	failedLog := &whatsappDomain.WebhookLog{
		OrganizationID: orgID,
		Status:         "failed",
		PhoneNumberID:  phoneNumberID,
		ErrorMessage:   reason,
		RawBody:        rawBody,
	}
	if _, err := s.logRepo.Insert(ctx, failedLog); err != nil {
		s.logger.Error("failed to log failed webhook", loggerdomain.Fields{"error": err.Error()})
	}
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

// computeDeliveryKey derives a stable dedup key from the provider entry id
// and the contained message ids. Meta retries of the same delivery carry the
// same ids, so a retried delivery is acknowledged without re-dispatch.
func computeDeliveryKey(payload map[string]any) string {
	entry := firstEntry(payload)
	if entry == nil {
		return ""
	}
	entryID, _ := entry["id"].(string)

	var messageIDs []string
	for _, change := range changesOf(entry) {
		for _, raw := range messagesOf(change) {
			if msg, ok := raw.(map[string]any); ok {
				if id, ok := msg["id"].(string); ok && id != "" {
					messageIDs = append(messageIDs, id)
				}
			}
		}
	}

	if entryID == "" && len(messageIDs) == 0 {
		return ""
	}

	key := entryID
	if len(messageIDs) > 0 {
		sort.Strings(messageIDs)
		for _, id := range messageIDs {
			key += "|" + id
		}
	}
	return key
}

func changesOf(entry map[string]any) []any {
	if changes, ok := entry["changes"].([]any); ok {
		return changes
	}
	return nil
}

func messagesOf(change any) []any {
	changeMap, ok := change.(map[string]any)
	if !ok {
		return nil
	}
	value, ok := changeMap["value"].(map[string]any)
	if !ok {
		return nil
	}
	messages, ok := value["messages"].([]any)
	if !ok {
		return nil
	}
	return messages
}

type parsedMessage struct {
	MessageSID  string
	From        string
	MessageType string
	Content     string
	MediaURL    string
	Timestamp   time.Time
	IsEcho      bool
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
			IsEcho:      isEchoMessage(msg),
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

// isEchoMessage reports whether a message entry is a coexistence echo
// (sent from the organization's phone WhatsApp Business app).
func isEchoMessage(msg map[string]any) bool {
	origin, ok := msg["origin"].(map[string]any)
	if !ok {
		return false
	}
	return stringField(origin, "type") == "echo"
}

func stringField(m map[string]any, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}
