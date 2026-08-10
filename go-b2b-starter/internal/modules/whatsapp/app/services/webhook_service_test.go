package services

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/moasq/go-b2b-starter/internal/modules/whatsapp/domain"
	"github.com/moasq/go-b2b-starter/internal/modules/whatsapp/domain/events"
	"github.com/moasq/go-b2b-starter/internal/platform/outbox"
)

type fakeOutboxRepo struct {
	inserted []*outbox.OutboxEvent
	claims   int
}

func (f *fakeOutboxRepo) Insert(_ context.Context, eventType string, payload json.RawMessage, orgID *int32) (*outbox.OutboxEvent, error) {
	f.inserted = append(f.inserted, &outbox.OutboxEvent{
		EventType:      eventType,
		Payload:        payload,
		OrganizationID: orgID,
	})
	return f.inserted[len(f.inserted)-1], nil
}
func (f *fakeOutboxRepo) ClaimPending(context.Context, int32) ([]*outbox.OutboxEvent, error) {
	f.claims++
	return nil, nil
}
func (f *fakeOutboxRepo) MarkDispatched(context.Context, int64) error { return nil }
func (f *fakeOutboxRepo) Retry(context.Context, int64, time.Time, string) error {
	return nil
}
func (f *fakeOutboxRepo) DeadLetter(context.Context, int64, string) error { return nil }
func (f *fakeOutboxRepo) ListDeadLetter(context.Context, int32) ([]*outbox.OutboxEvent, error) {
	return nil, nil
}
func (f *fakeOutboxRepo) Requeue(context.Context, int64) error { return nil }
func (f *fakeOutboxRepo) Prune(context.Context, time.Time) error { return nil }

type fakeWebhookConfigRepo struct {
	cfg *domain.WhatsAppConfig
}

func (f *fakeWebhookConfigRepo) GetByPhoneNumberID(_ context.Context, _ string) (*domain.WhatsAppConfig, error) {
	if f.cfg == nil {
		return nil, errors.New("not found")
	}
	return f.cfg, nil
}
func (f *fakeWebhookConfigRepo) GetByOrganizationID(_ context.Context, _ int32) (*domain.WhatsAppConfig, error) {
	if f.cfg == nil {
		return nil, errors.New("not found")
	}
	return f.cfg, nil
}
func (f *fakeWebhookConfigRepo) GetByVerifyToken(context.Context, string) (*domain.WhatsAppConfig, error) {
	return nil, errors.New("not found")
}
func (f *fakeWebhookConfigRepo) Create(context.Context, *domain.WhatsAppConfig) (*domain.WhatsAppConfig, error) {
	return nil, errors.New("not implemented")
}
func (f *fakeWebhookConfigRepo) Update(context.Context, *domain.WhatsAppConfig) (*domain.WhatsAppConfig, error) {
	return nil, errors.New("not implemented")
}

type fakeLogRepo struct {
	inserted       []*domain.WebhookLog
	outboxEvents   []domain.OutboxEventInput
	duplicateOnKey string
	insertErr      error
	store          map[int32]*domain.WebhookLog
}

func (f *fakeLogRepo) Insert(_ context.Context, l *domain.WebhookLog) (*domain.WebhookLog, error) {
	if f.insertErr != nil && l.Status != "duplicate" && l.Status != "replay" {
		return nil, f.insertErr
	}
	f.inserted = append(f.inserted, l)
	if f.store != nil {
		l.ID = int32(len(f.inserted))
		f.store[l.ID] = l
	}
	return l, nil
}

func (f *fakeLogRepo) GetByID(_ context.Context, id int32) (*domain.WebhookLog, error) {
	if f.store == nil {
		return nil, domain.ErrWebhookLogNotFound
	}
	log, ok := f.store[id]
	if !ok {
		return nil, domain.ErrWebhookLogNotFound
	}
	return log, nil
}

func (f *fakeLogRepo) InsertWithOutbox(_ context.Context, l *domain.WebhookLog, events []domain.OutboxEventInput) (*domain.WebhookLog, error) {
	if f.insertErr != nil {
		return nil, f.insertErr
	}
	if f.duplicateOnKey != "" && l.DeliveryKey == f.duplicateOnKey {
		return nil, domain.ErrDuplicateDelivery
	}
	f.inserted = append(f.inserted, l)
	f.outboxEvents = append(f.outboxEvents, events...)
	if f.store != nil {
		l.ID = int32(len(f.inserted))
		f.store[l.ID] = l
	}
	return l, nil
}

func (f *fakeLogRepo) GetStatsByOrganization(context.Context, int32) (*domain.WebhookLogStats, error) {
	return &domain.WebhookLogStats{}, nil
}

func hmacHeader(secret string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

const echoPayload = `{
  "object": "whatsapp_business_account",
  "entry": [{
    "id": "waba-1",
    "changes": [{
      "value": {
        "messaging_product": "whatsapp",
        "metadata": {"phone_number_id": "phone-1"},
        "messages": [{
          "from": "573001234567",
          "id": "wamid.ECHO1",
          "timestamp": "1723000000",
          "type": "text",
          "origin": {"type": "echo"},
          "text": {"body": "sent from the phone app"}
        }]
      },
      "field": "messages"
    }]
  }]
}`

const inboundPayload = `{
  "object": "whatsapp_business_account",
  "entry": [{
    "id": "waba-1",
    "changes": [{
      "value": {
        "messaging_product": "whatsapp",
        "metadata": {"phone_number_id": "phone-1"},
        "messages": [{
          "from": "573001234567",
          "id": "wamid.IN1",
          "timestamp": "1723000000",
          "type": "text",
          "text": {"body": "hello"}
        }]
      },
      "field": "messages"
    }]
  }]
}`

const noOriginPayload = `{
  "object": "whatsapp_business_account",
  "entry": [{
    "id": "waba-1",
    "changes": [{
      "value": {
        "messaging_product": "whatsapp",
        "metadata": {"phone_number_id": "phone-1"},
        "messages": [{
          "from": "573001234567",
          "id": "wamid.NO1",
          "timestamp": "1723000000",
          "type": "text",
          "text": {"body": "no origin"}
        }]
      },
      "field": "messages"
    }]
  }]
}`

func newWebhookServiceForTest(cfg *domain.WhatsAppConfig) (*webhookService, *fakeOutboxRepo, *fakeLogRepo) {
	outboxRepo := &fakeOutboxRepo{}
	logs := &fakeLogRepo{store: make(map[int32]*domain.WebhookLog)}
	return &webhookService{
		configRepo: &fakeWebhookConfigRepo{cfg: cfg},
		logRepo:    logs,
		outboxRepo: outboxRepo,
		logger:     noopLogger{},
	}, outboxRepo, logs
}

func TestProcessWebhook_EchoPayloadPublishesEchoEvent(t *testing.T) {
	cfg := &domain.WhatsAppConfig{OrganizationID: 7, PhoneNumberID: "phone-1", BusinessPhone: "+573001234567", WebhookSecret: "whsec_test"}
	svc, _, logs := newWebhookServiceForTest(cfg)

	err := svc.ProcessWebhook(context.Background(), []byte(echoPayload), parsePayload(t, echoPayload), hmacHeader(cfg.WebhookSecret, []byte(echoPayload)))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(logs.outboxEvents) != 1 {
		t.Fatalf("expected exactly 1 outbox event, got %d", len(logs.outboxEvents))
	}
	if logs.outboxEvents[0].EventType != events.MessageEchoEventType {
		t.Fatalf("expected echo event type, got %q", logs.outboxEvents[0].EventType)
	}
	var echo events.MessageEcho
	if err := json.Unmarshal(logs.outboxEvents[0].Payload, &echo); err != nil {
		t.Fatalf("failed to decode echo payload: %v", err)
	}
	if echo.MessageSID != "wamid.ECHO1" || echo.MessageType != "text" || echo.Content != "sent from the phone app" {
		t.Fatalf("unexpected echo event: %+v", echo)
	}
}

func TestProcessWebhook_EchoNeverPublishedAsInbound(t *testing.T) {
	cfg := &domain.WhatsAppConfig{OrganizationID: 7, PhoneNumberID: "phone-1", BusinessPhone: "+573001234567", WebhookSecret: "whsec_test"}
	svc, _, logs := newWebhookServiceForTest(cfg)

	err := svc.ProcessWebhook(context.Background(), []byte(echoPayload), parsePayload(t, echoPayload), hmacHeader(cfg.WebhookSecret, []byte(echoPayload)))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if logs.outboxEvents[0].EventType == events.MessageReceivedEventType {
		t.Fatal("echo must not be enqueued as whatsapp.message.received")
	}
}

func TestProcessWebhook_InboundPublishesReceivedEvent(t *testing.T) {
	cfg := &domain.WhatsAppConfig{OrganizationID: 7, PhoneNumberID: "phone-1", BusinessPhone: "+573001234567", WebhookSecret: "whsec_test"}
	svc, _, logs := newWebhookServiceForTest(cfg)

	err := svc.ProcessWebhook(context.Background(), []byte(inboundPayload), parsePayload(t, inboundPayload), hmacHeader(cfg.WebhookSecret, []byte(inboundPayload)))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(logs.outboxEvents) != 1 {
		t.Fatalf("expected exactly 1 outbox event, got %d", len(logs.outboxEvents))
	}
	if logs.outboxEvents[0].EventType != events.MessageReceivedEventType {
		t.Fatalf("expected received event type, got %q", logs.outboxEvents[0].EventType)
	}
	var msg events.MessageReceived
	if err := json.Unmarshal(logs.outboxEvents[0].Payload, &msg); err != nil {
		t.Fatalf("failed to decode received payload: %v", err)
	}
	if msg.MessageSID != "wamid.IN1" {
		t.Fatalf("unexpected message sid: %s", msg.MessageSID)
	}
}

func TestProcessWebhook_NoOriginTreatsAsInbound(t *testing.T) {
	cfg := &domain.WhatsAppConfig{OrganizationID: 7, PhoneNumberID: "phone-1", BusinessPhone: "+573001234567", WebhookSecret: "whsec_test"}
	svc, _, logs := newWebhookServiceForTest(cfg)

	err := svc.ProcessWebhook(context.Background(), []byte(noOriginPayload), parsePayload(t, noOriginPayload), hmacHeader(cfg.WebhookSecret, []byte(noOriginPayload)))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if logs.outboxEvents[0].EventType != events.MessageReceivedEventType {
		t.Fatalf("expected received event for no-origin message, got %q", logs.outboxEvents[0].EventType)
	}
	var msg events.MessageReceived
	if err := json.Unmarshal(logs.outboxEvents[0].Payload, &msg); err != nil {
		t.Fatalf("failed to decode payload: %v", err)
	}
	if msg.MessageSID != "wamid.NO1" {
		t.Fatalf("unexpected event sid: %+v", msg)
	}
}

func TestProcessWebhook_InvalidSignatureRejected(t *testing.T) {
	cfg := &domain.WhatsAppConfig{OrganizationID: 7, PhoneNumberID: "phone-1", BusinessPhone: "+573001234567", WebhookSecret: "whsec_test"}
	svc, _, logs := newWebhookServiceForTest(cfg)

	err := svc.ProcessWebhook(context.Background(), []byte(echoPayload), parsePayload(t, echoPayload), "sha256=deadbeef")
	if err == nil {
		t.Fatal("expected signature error")
	}
	if len(logs.outboxEvents) != 0 {
		t.Fatalf("expected no outbox events on bad signature, got %d", len(logs.outboxEvents))
	}
}

func TestProcessWebhook_InvalidSignatureLogsFailedWithResolvedOrg(t *testing.T) {
	cfg := &domain.WhatsAppConfig{OrganizationID: 7, PhoneNumberID: "phone-1", BusinessPhone: "+573001234567", WebhookSecret: "whsec_test"}
	svc, _, logs := newWebhookServiceForTest(cfg)

	err := svc.ProcessWebhook(context.Background(), []byte(echoPayload), parsePayload(t, echoPayload), "sha256=deadbeef")
	if err == nil {
		t.Fatal("expected signature error")
	}
	if len(logs.inserted) != 1 {
		t.Fatalf("expected 1 failed log insert, got %d", len(logs.inserted))
	}
	log := logs.inserted[0]
	if log.Status != "failed" {
		t.Fatalf("expected status failed, got %q", log.Status)
	}
	if log.OrganizationID == nil || *log.OrganizationID != 7 {
		t.Fatalf("expected failed log with resolved org 7, got %v", log.OrganizationID)
	}
	if log.ErrorMessage == "" {
		t.Fatal("expected error message on failed log")
	}
}

func TestProcessWebhook_UnknownPhoneLogsFailedWithNullOrg(t *testing.T) {
	svc, _, logs := newWebhookServiceForTest(nil)

	err := svc.ProcessWebhook(context.Background(), []byte(inboundPayload), parsePayload(t, inboundPayload), hmacHeader("whsec_test", []byte(inboundPayload)))
	if err == nil {
		t.Fatal("expected unknown-phone error")
	}
	if !errors.Is(err, domain.ErrUnknownPhoneNumber) {
		t.Fatalf("expected ErrUnknownPhoneNumber, got %v", err)
	}
	if len(logs.inserted) != 1 {
		t.Fatalf("expected 1 failed log insert, got %d", len(logs.inserted))
	}
	log := logs.inserted[0]
	if log.Status != "failed" {
		t.Fatalf("expected status failed, got %q", log.Status)
	}
	if log.OrganizationID != nil {
		t.Fatalf("expected NULL org on unresolvable failure, got %v", *log.OrganizationID)
	}
}

func TestProcessWebhook_MissingPhoneNumberLogsFailed(t *testing.T) {
	cfg := &domain.WhatsAppConfig{OrganizationID: 7, PhoneNumberID: "phone-1", BusinessPhone: "+573001234567", WebhookSecret: "whsec_test"}
	svc, _, logs := newWebhookServiceForTest(cfg)

	payload := `{"object":"whatsapp_business_account","entry":[{"id":"waba-1","changes":[{"value":{"messaging_product":"whatsapp","messages":[]},"field":"messages"}]}]}`

	err := svc.ProcessWebhook(context.Background(), []byte(payload), parsePayload(t, payload), hmacHeader(cfg.WebhookSecret, []byte(payload)))
	if err == nil {
		t.Fatal("expected missing phone_number_id error")
	}
	if len(logs.inserted) != 1 {
		t.Fatalf("expected 1 failed log insert, got %d", len(logs.inserted))
	}
	if logs.inserted[0].OrganizationID != nil {
		t.Fatalf("expected NULL org on missing-phone failure, got %v", *logs.inserted[0].OrganizationID)
	}
}

func TestProcessWebhook_SuccessLogsReceived(t *testing.T) {
	cfg := &domain.WhatsAppConfig{OrganizationID: 7, PhoneNumberID: "phone-1", BusinessPhone: "+573001234567", WebhookSecret: "whsec_test"}
	svc, _, logs := newWebhookServiceForTest(cfg)

	err := svc.ProcessWebhook(context.Background(), []byte(inboundPayload), parsePayload(t, inboundPayload), hmacHeader(cfg.WebhookSecret, []byte(inboundPayload)))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(logs.inserted) != 1 {
		t.Fatalf("expected 1 log insert, got %d", len(logs.inserted))
	}
	log := logs.inserted[0]
	if log.Status != "received" {
		t.Fatalf("expected status received, got %q", log.Status)
	}
	if log.OrganizationID == nil || *log.OrganizationID != 7 {
		t.Fatalf("expected received log with resolved org 7, got %v", log.OrganizationID)
	}
	if log.DeliveryKey == "" {
		t.Fatal("expected delivery_key on successful log")
	}
}

func TestProcessWebhook_DuplicateDeliveryAcknowledgedWithoutRedispatch(t *testing.T) {
	cfg := &domain.WhatsAppConfig{OrganizationID: 7, PhoneNumberID: "phone-1", BusinessPhone: "+573001234567", WebhookSecret: "whsec_test"}
	svc, outboxRepo, logs := newWebhookServiceForTest(cfg)

	body := []byte(inboundPayload)
	payload := parsePayload(t, inboundPayload)
	key := computeDeliveryKey(payload)
	logs.duplicateOnKey = key

	err := svc.ProcessWebhook(context.Background(), body, payload, hmacHeader(cfg.WebhookSecret, body))
	if err != nil {
		t.Fatalf("duplicate delivery must be acknowledged, got error: %v", err)
	}
	if len(outboxRepo.inserted) != 0 {
		t.Fatalf("expected no outbox events for duplicate, got %d", len(outboxRepo.inserted))
	}
	if len(logs.inserted) != 1 {
		t.Fatalf("expected 1 duplicate log insert, got %d", len(logs.inserted))
	}
	if logs.inserted[0].Status != "duplicate" {
		t.Fatalf("expected duplicate status, got %q", logs.inserted[0].Status)
	}
}

func TestProcessWebhook_AtomicFailureReturnsError(t *testing.T) {
	cfg := &domain.WhatsAppConfig{OrganizationID: 7, PhoneNumberID: "phone-1", BusinessPhone: "+573001234567", WebhookSecret: "whsec_test"}
	svc, outboxRepo, logs := newWebhookServiceForTest(cfg)

	// Force the atomic insert to fail with a non-duplicate error.
	logs.insertErr = errors.New("db unavailable")

	err := svc.ProcessWebhook(context.Background(), []byte(inboundPayload), parsePayload(t, inboundPayload), hmacHeader(cfg.WebhookSecret, []byte(inboundPayload)))
	if err == nil {
		t.Fatal("expected error when atomic persistence fails")
	}
	if len(outboxRepo.inserted) != 0 {
		t.Fatalf("expected no outbox events after failed transaction, got %d", len(outboxRepo.inserted))
	}
}

func TestReplay_ReenqueuesEventsFromStoredLog(t *testing.T) {
	cfg := &domain.WhatsAppConfig{OrganizationID: 7, PhoneNumberID: "phone-1", BusinessPhone: "+573001234567", WebhookSecret: "whsec_test"}
	svc, outboxRepo, logs := newWebhookServiceForTest(cfg)

	err := svc.ProcessWebhook(context.Background(), []byte(inboundPayload), parsePayload(t, inboundPayload), hmacHeader(cfg.WebhookSecret, []byte(inboundPayload)))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Reset captured inserts; replay should re-enqueue from the stored log.
	outboxRepo.inserted = nil
	logID := int32(1)
	if len(logs.inserted) == 0 || logs.inserted[0].ID == 0 {
		t.Fatal("expected stored log with id")
	}
	logID = logs.inserted[0].ID

	count, err := svc.Replay(context.Background(), 7, logID)
	if err != nil {
		t.Fatalf("replay failed: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 replayed event, got %d", count)
	}
	if len(outboxRepo.inserted) != 1 {
		t.Fatalf("expected 1 outbox insert from replay, got %d", len(outboxRepo.inserted))
	}
	if outboxRepo.inserted[0].EventType != events.MessageReceivedEventType {
		t.Fatalf("unexpected replay event type: %q", outboxRepo.inserted[0].EventType)
	}
}

func TestReplay_UnknownLogFails(t *testing.T) {
	cfg := &domain.WhatsAppConfig{OrganizationID: 7, PhoneNumberID: "phone-1", BusinessPhone: "+573001234567", WebhookSecret: "whsec_test"}
	svc, _, _ := newWebhookServiceForTest(cfg)

	if _, err := svc.Replay(context.Background(), 7, 9999); !errors.Is(err, domain.ErrWebhookLogNotFound) {
		t.Fatalf("expected ErrWebhookLogNotFound, got %v", err)
	}
}

func parsePayload(t *testing.T, raw string) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal([]byte(raw), &m); err != nil {
		t.Fatalf("invalid fixture: %v", err)
	}
	return m
}
