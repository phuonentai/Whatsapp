package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"time"

	igDomain "github.com/moasq/go-b2b-starter/internal/modules/instagram/domain"
	"github.com/moasq/go-b2b-starter/internal/modules/instagram/domain/events"
	"github.com/moasq/go-b2b-starter/internal/platform/logger"
	loggerdomain "github.com/moasq/go-b2b-starter/internal/platform/logger/domain"
	"github.com/moasq/go-b2b-starter/internal/platform/outbox"
	"github.com/moasq/go-b2b-starter/pkg/whatsapp"
)

type WebhookService interface {
	ProcessWebhook(ctx context.Context, rawBody []byte, parsedPayload map[string]any, signatureHeader string) error
	VerifyChallenge(ctx context.Context, mode, verifyToken, challenge string) error
	GetWebhookLogStats(ctx context.Context, orgID int32) (*igDomain.WebhookLogStats, error)
	Replay(ctx context.Context, orgID, logID int32) (int, error)
}

type webhookService struct {
	configRepo     igDomain.ConfigRepository
	logRepo        igDomain.WebhookLogRepository
	outboxRepo     outbox.Repository
	platformVerify string
	logger         logger.Logger
}

func NewWebhookService(
	configRepo igDomain.ConfigRepository,
	logRepo igDomain.WebhookLogRepository,
	outboxRepo outbox.Repository,
	platformVerify string,
	logger logger.Logger,
) WebhookService {
	return &webhookService{
		configRepo:     configRepo,
		logRepo:        logRepo,
		outboxRepo:     outboxRepo,
		platformVerify: platformVerify,
		logger:         logger,
	}
}

// ProcessWebhook resolves the org from the payload's recipient.id, verifies
// the HMAC signature against the resolved config, and durably enqueues
// message events. Instagram DMs arrive on app-level webhooks with a single
// shared verify token, so org resolution happens BEFORE signature checks via
// the unique ig_user_id lookup.
func (s *webhookService) ProcessWebhook(ctx context.Context, rawBody []byte, parsedPayload map[string]any, signatureHeader string) error {
	value := extractFirstValue(parsedPayload)
	if value == nil {
		s.logFailed(ctx, rawBody, nil, "", "no entry[].changes[].value in payload")
		return fmt.Errorf("no entry[].changes[].value in payload")
	}

	recipientID := stringField(value, "recipient", "id")
	if recipientID == "" {
		// Legacy shape: messaging[].recipient.id
		recipientID = extractLegacyRecipientID(parsedPayload)
	}
	if recipientID == "" {
		s.logFailed(ctx, rawBody, nil, "", "recipient.id not found in payload")
		return fmt.Errorf("recipient.id not found in payload")
	}

	config, err := s.configRepo.GetByIGUserID(ctx, recipientID)
	if err != nil {
		s.logFailed(ctx, rawBody, nil, recipientID, "config not found for ig_user_id "+recipientID)
		return fmt.Errorf("%w: config not found for ig_user_id %s", igDomain.ErrUnknownIGUser, recipientID)
	}

	if err := whatsapp.VerifySignature(config.WebhookSecret, rawBody, signatureHeader); err != nil {
		s.logFailed(ctx, rawBody, &config.OrganizationID, recipientID, "invalid signature")
		return fmt.Errorf("%w: %v", igDomain.ErrInvalidSignature, err)
	}

	messages := extractMessagesFromValue(value, parsedPayload)
	outboxEvents := s.buildOutboxEvents(config, messages, rawBody)

	webhookLog := &igDomain.WebhookLog{
		OrganizationID: &config.OrganizationID,
		Status:         "received",
		EventType:      extractEventType(value),
		IGUserID:       recipientID,
		RawBody:        rawBody,
		DeliveryKey:    computeDeliveryKey(parsedPayload),
	}

	if _, err := s.logRepo.InsertWithOutbox(ctx, webhookLog, outboxEvents); err != nil {
		if errors.Is(err, igDomain.ErrDuplicateDelivery) {
			// At-least-once retry: acknowledge without re-dispatching.
			duplicate := &igDomain.WebhookLog{
				OrganizationID: &config.OrganizationID,
				Status:         "duplicate",
				EventType:      extractEventType(value),
				IGUserID:       recipientID,
				RawBody:        rawBody,
				ErrorMessage:   "duplicate delivery acknowledged",
			}
			if _, insErr := s.logRepo.Insert(ctx, duplicate); insErr != nil {
				s.logger.Error("failed to log duplicate delivery", loggerdomain.Fields{"error": insErr.Error()})
			}
			s.logger.Info("duplicate instagram webhook delivery acknowledged", loggerdomain.Fields{
				"ig_user_id":   recipientID,
				"delivery_key": webhookLog.DeliveryKey,
			})
			return nil
		}
		s.logger.Error("failed to persist instagram webhook with outbox events", loggerdomain.Fields{"error": err.Error()})
		return fmt.Errorf("failed to persist instagram webhook delivery: %w", err)
	}

	return nil
}

// buildOutboxEvents serializes parsed messages into durable outbox payloads.
// Echoes (is_echo) become outbound CRM mirrors; everything else is inbound.
func (s *webhookService) buildOutboxEvents(config *igDomain.InstagramConfig, messages []parsedMessage, rawBody []byte) []igDomain.OutboxEventInput {
	eventsOut := make([]igDomain.OutboxEventInput, 0, len(messages))
	for _, msg := range messages {
		if msg.IsEcho {
			echo := events.NewMessageEcho(
				config.OrganizationID,
				msg.MessageSID,
				msg.From,
				config.IGUserID,
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
			eventsOut = append(eventsOut, igDomain.OutboxEventInput{
				EventType: events.MessageEchoEventType,
				Payload:   payload,
			})
			continue
		}

		event := events.NewMessageReceived(
			config.OrganizationID,
			msg.MessageSID,
			msg.From,
			config.IGUserID,
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
		eventsOut = append(eventsOut, igDomain.OutboxEventInput{
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
		return 0, igDomain.ErrWebhookLogNotFound
	}

	config, err := s.configRepo.GetByOrganizationID(ctx, orgID)
	if err != nil {
		return 0, fmt.Errorf("failed to resolve config for replay: %w", err)
	}

	var payload map[string]any
	if err := json.Unmarshal(log.RawBody, &payload); err != nil {
		return 0, fmt.Errorf("failed to parse stored payload for replay: %w", err)
	}

	value := extractFirstValue(payload)
	messages := extractMessagesFromValue(value, payload)
	eventsOut := s.buildOutboxEvents(config, messages, log.RawBody)

	for _, ev := range eventsOut {
		if _, err := s.outboxRepo.Insert(ctx, ev.EventType, ev.Payload, log.OrganizationID); err != nil {
			return 0, fmt.Errorf("failed to enqueue replay event %q: %w", ev.EventType, err)
		}
	}

	if _, err := s.logRepo.Insert(ctx, &igDomain.WebhookLog{
		OrganizationID: log.OrganizationID,
		Status:         "replay",
		EventType:      log.EventType,
		IGUserID:       log.IGUserID,
		RawBody:        log.RawBody,
		ErrorMessage:   fmt.Sprintf("replayed from log %d", logID),
	}); err != nil {
		s.logger.Warn("failed to record replay action", loggerdomain.Fields{"error": err.Error(), "log_id": logID})
	}

	return len(eventsOut), nil
}

func (s *webhookService) logFailed(ctx context.Context, rawBody []byte, orgID *int32, igUserID, reason string) {
	failedLog := &igDomain.WebhookLog{
		OrganizationID: orgID,
		Status:         "failed",
		IGUserID:       igUserID,
		ErrorMessage:   reason,
		RawBody:        rawBody,
	}
	if _, err := s.logRepo.Insert(ctx, failedLog); err != nil {
		s.logger.Error("failed to log failed webhook", loggerdomain.Fields{"error": err.Error()})
	}
}

func (s *webhookService) GetWebhookLogStats(ctx context.Context, orgID int32) (*igDomain.WebhookLogStats, error) {
	return s.logRepo.GetStatsByOrganization(ctx, orgID)
}

// VerifyChallenge validates the hub handshake: the platform-level
// INSTAGRAM_WEBHOOK_VERIFY_TOKEN, or any active config's verify_token.
func (s *webhookService) VerifyChallenge(ctx context.Context, mode, verifyToken, challenge string) error {
	if mode != "subscribe" {
		return fmt.Errorf("invalid verification mode: %s", mode)
	}
	if verifyToken == "" {
		return fmt.Errorf("verify token is required")
	}
	if challenge == "" {
		return fmt.Errorf("challenge is required")
	}

	if s.platformVerify != "" && verifyToken == s.platformVerify {
		return nil
	}

	if _, err := s.configRepo.GetByVerifyToken(ctx, verifyToken); err != nil {
		return fmt.Errorf("%w: verify token does not match the platform token or any active config", igDomain.ErrWebhookVerificationFail)
	}
	return nil
}

// ============================================================
// Payload parsing
// ============================================================

type parsedMessage struct {
	MessageSID  string
	From        string
	MessageType string
	Content     string
	MediaURL    string
	Timestamp   time.Time
	IsEcho      bool
}

// extractMessagesFromValue parses messages from either the modern
// entry[].changes[].value.messages shape or the legacy entry[].messaging shape.
func extractMessagesFromValue(value map[string]any, payload map[string]any) []parsedMessage {
	if value != nil {
		rawMessages, ok := value["messages"].([]any)
		if !ok {
			return nil
		}

		// IG DM payloads carry sender/recipient at the value level; the
		// message objects carry mid + content.
		valueSender, _ := value["sender"].(map[string]any)
		defaultFrom, _ := valueSender["id"].(string)

		var result []parsedMessage
		for _, raw := range rawMessages {
			msg, ok := raw.(map[string]any)
			if !ok {
				continue
			}
			if p, ok := parseMessage(msg); ok {
				if p.From == "" {
					p.From = defaultFrom
				}
				result = append(result, p)
			}
		}
		return result
	}

	// Legacy shape: entry[].messaging[] with message objects.
	legacy := extractLegacyMessaging(payload)
	if len(legacy) == 0 {
		return nil
	}

	var result []parsedMessage
	for _, entry := range legacy {
		if msg, ok := entry["message"].(map[string]any); ok {
			mid, _ := msg["mid"].(string)
			senderEntry, _ := entry["sender"].(map[string]any)
			from, _ := senderEntry["id"].(string)
			p := parsedMessage{MessageSID: mid, From: from}
			if text, ok := msg["text"].(string); ok {
				p.MessageType = "text"
				p.Content = text
			}
			if isEcho, _ := msg["is_echo"].(bool); isEcho {
				p.IsEcho = true
			}
			if atts, ok := msg["attachments"].([]any); ok {
				p.MessageType = "media"
				for _, a := range atts {
					if am, ok := a.(map[string]any); ok {
						if payloadMap, ok := am["payload"].(map[string]any); ok {
							if u, ok := payloadMap["url"].(string); ok && u != "" {
								p.MediaURL = u
								break
							}
						}
					}
				}
			}
			result = append(result, p)
		}
	}
	return result
}

// parseMessage converts a modern webhook message object into parsedMessage.
func parseMessage(msg map[string]any) (parsedMessage, bool) {
	mid, _ := msg["mid"].(string)
	if mid == "" {
		mid = stringField(msg, "id")
	}
	if mid == "" {
		return parsedMessage{}, false
	}

	sender, _ := msg["sender"].(map[string]any)
	from, _ := sender["id"].(string)
	if from == "" {
		from = stringField(msg, "from")
	}

	p := parsedMessage{
		MessageSID: mid,
		From:       from,
	}

	if isEchoValue, ok := msg["is_echo"].(bool); ok {
		p.IsEcho = isEchoValue
	}

	if text, ok := msg["text"].(string); ok {
		p.MessageType = "text"
		p.Content = text
	}
	if msg["text"] != nil {
		if textMap, ok := msg["text"].(map[string]any); ok {
			p.MessageType = "text"
			p.Content = stringField(textMap, "text")
		}
	}

	if atts, ok := msg["attachments"].([]any); ok && len(atts) > 0 {
		p.MessageType = "media"
		for _, a := range atts {
			am, ok := a.(map[string]any)
			if !ok {
				continue
			}
			if payloadMap, ok := am["payload"].(map[string]any); ok {
				if u, ok := payloadMap["url"].(string); ok && u != "" {
					p.MediaURL = u
					break
				}
			}
		}
	}

	return p, true
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
				if id, ok := msg["mid"].(string); ok && id != "" {
					messageIDs = append(messageIDs, id)
				} else if id, ok := msg["id"].(string); ok && id != "" {
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

func extractFirstValue(payload map[string]any) map[string]any {
	entry := firstEntry(payload)
	change := firstChange(entry)
	return changeValue(change)
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

func extractEventType(value map[string]any) string {
	if value == nil {
		return "unknown"
	}
	if msgs, ok := value["messages"].([]any); ok && len(msgs) > 0 {
		if msg, ok := msgs[0].(map[string]any); ok {
			if t, ok := msg["message_type"].(string); ok {
				return t
			}
		}
	}
	return "unknown"
}

// extractLegacyRecipientID resolves org from the legacy messaging shape.
func extractLegacyRecipientID(payload map[string]any) string {
	messaging := extractLegacyMessaging(payload)
	for _, entry := range messaging {
		if recipient, ok := entry["recipient"].(map[string]any); ok {
			if id, ok := recipient["id"].(string); ok && id != "" {
				return id
			}
		}
	}
	return ""
}

func extractLegacyMessaging(payload map[string]any) []map[string]any {
	entry := firstEntry(payload)
	if entry == nil {
		return nil
	}
	raw, ok := entry["messaging"].([]any)
	if !ok {
		return nil
	}
	var result []map[string]any
	for _, m := range raw {
		if mm, ok := m.(map[string]any); ok {
			result = append(result, mm)
		}
	}
	return result
}

func stringField(m map[string]any, keys ...string) string {
	var cur any = m
	for _, k := range keys {
		mm, ok := cur.(map[string]any)
		if !ok {
			return ""
		}
		cur, ok = mm[k]
		if !ok {
			return ""
		}
	}
	if s, ok := cur.(string); ok {
		return s
	}
	return ""
}
