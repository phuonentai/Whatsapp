package services

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"testing"

	"github.com/moasq/go-b2b-starter/internal/modules/whatsapp/domain"
	"github.com/moasq/go-b2b-starter/internal/modules/whatsapp/domain/events"
	"github.com/moasq/go-b2b-starter/internal/platform/eventbus"
)

type fakeBus struct {
	events []eventbus.Event
}

func (b *fakeBus) Publish(_ context.Context, e eventbus.Event) error {
	b.events = append(b.events, e)
	return nil
}
func (b *fakeBus) Subscribe(string, eventbus.EventHandler[eventbus.Event]) error { return nil }
func (b *fakeBus) Unsubscribe(string, eventbus.EventHandler[eventbus.Event]) error {
	return nil
}
func (b *fakeBus) Close() error { return nil }

type fakeWebhookConfigRepo struct {
	cfg *domain.WhatsAppConfig
}

func (f *fakeWebhookConfigRepo) GetByPhoneNumberID(_ context.Context, _ string) (*domain.WhatsAppConfig, error) {
	if f.cfg == nil {
		return nil, errors.New("not found")
	}
	return f.cfg, nil
}
func (f *fakeWebhookConfigRepo) GetByOrganizationID(context.Context, int32) (*domain.WhatsAppConfig, error) {
	return nil, errors.New("not found")
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

type fakeLogRepo struct{}

func (f *fakeLogRepo) Insert(context.Context, *domain.WebhookLog) (*domain.WebhookLog, error) {
	return &domain.WebhookLog{}, nil
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

func newWebhookServiceForTest(cfg *domain.WhatsAppConfig) (*webhookService, *fakeBus) {
	bus := &fakeBus{}
	return &webhookService{
		configRepo: &fakeWebhookConfigRepo{cfg: cfg},
		logRepo:    &fakeLogRepo{},
		eventBus:   bus,
		logger:     noopLogger{},
	}, bus
}

func TestProcessWebhook_EchoPayloadPublishesEchoEvent(t *testing.T) {
	cfg := &domain.WhatsAppConfig{OrganizationID: 7, PhoneNumberID: "phone-1", BusinessPhone: "+573001234567", WebhookSecret: "whsec_test"}
	svc, bus := newWebhookServiceForTest(cfg)

	err := svc.ProcessWebhook(context.Background(), []byte(echoPayload), parsePayload(t, echoPayload), hmacHeader(cfg.WebhookSecret, []byte(echoPayload)))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(bus.events) != 1 {
		t.Fatalf("expected exactly 1 published event, got %d", len(bus.events))
	}
	echo, ok := bus.events[0].(*events.MessageEcho)
	if !ok {
		t.Fatalf("expected *events.MessageEcho, got %T", bus.events[0])
	}
	if echo.MessageSID != "wamid.ECHO1" || echo.MessageType != "text" || echo.Content != "sent from the phone app" {
		t.Fatalf("unexpected echo event: %+v", echo)
	}
}

func TestProcessWebhook_EchoNeverPublishedAsInbound(t *testing.T) {
	cfg := &domain.WhatsAppConfig{OrganizationID: 7, PhoneNumberID: "phone-1", BusinessPhone: "+573001234567", WebhookSecret: "whsec_test"}
	svc, bus := newWebhookServiceForTest(cfg)

	err := svc.ProcessWebhook(context.Background(), []byte(echoPayload), parsePayload(t, echoPayload), hmacHeader(cfg.WebhookSecret, []byte(echoPayload)))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := bus.events[0].(*events.MessageReceived); ok {
		t.Fatal("echo must not be published as whatsapp.message.received")
	}
}

func TestProcessWebhook_InboundPublishesReceivedEvent(t *testing.T) {
	cfg := &domain.WhatsAppConfig{OrganizationID: 7, PhoneNumberID: "phone-1", BusinessPhone: "+573001234567", WebhookSecret: "whsec_test"}
	svc, bus := newWebhookServiceForTest(cfg)

	err := svc.ProcessWebhook(context.Background(), []byte(inboundPayload), parsePayload(t, inboundPayload), hmacHeader(cfg.WebhookSecret, []byte(inboundPayload)))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(bus.events) != 1 {
		t.Fatalf("expected exactly 1 published event, got %d", len(bus.events))
	}
	if _, ok := bus.events[0].(*events.MessageReceived); !ok {
		t.Fatalf("expected *events.MessageReceived, got %T", bus.events[0])
	}
}

func TestProcessWebhook_NoOriginTreatsAsInbound(t *testing.T) {
	cfg := &domain.WhatsAppConfig{OrganizationID: 7, PhoneNumberID: "phone-1", BusinessPhone: "+573001234567", WebhookSecret: "whsec_test"}
	svc, bus := newWebhookServiceForTest(cfg)

	err := svc.ProcessWebhook(context.Background(), []byte(noOriginPayload), parsePayload(t, noOriginPayload), hmacHeader(cfg.WebhookSecret, []byte(noOriginPayload)))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := bus.events[0].(*events.MessageReceived); !ok {
		t.Fatalf("expected *events.MessageReceived for no-origin message, got %T", bus.events[0])
	}
	if bus.events[0].(*events.MessageReceived).MessageSID != "wamid.NO1" {
		t.Fatalf("unexpected event sid: %+v", bus.events[0])
	}
}

func TestProcessWebhook_InvalidSignatureRejected(t *testing.T) {
	cfg := &domain.WhatsAppConfig{OrganizationID: 7, PhoneNumberID: "phone-1", BusinessPhone: "+573001234567", WebhookSecret: "whsec_test"}
	svc, bus := newWebhookServiceForTest(cfg)

	err := svc.ProcessWebhook(context.Background(), []byte(echoPayload), parsePayload(t, echoPayload), "sha256=deadbeef")
	if err == nil {
		t.Fatal("expected signature error")
	}
	if len(bus.events) != 0 {
		t.Fatalf("expected no published events on bad signature, got %d", len(bus.events))
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
