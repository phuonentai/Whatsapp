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

	igDomain "github.com/moasq/go-b2b-starter/internal/modules/instagram/domain"
	"github.com/moasq/go-b2b-starter/internal/modules/instagram/domain/events"
	loggerdomain "github.com/moasq/go-b2b-starter/internal/platform/logger/domain"
	"github.com/moasq/go-b2b-starter/internal/platform/outbox"
)

// ---------- fakes ----------

type fakeIGConfigRepo struct {
	cfg *igDomain.InstagramConfig
}

func (f *fakeIGConfigRepo) GetByIGUserID(_ context.Context, _ string) (*igDomain.InstagramConfig, error) {
	if f.cfg == nil {
		return nil, igDomain.ErrConfigNotFound
	}
	return f.cfg, nil
}
func (f *fakeIGConfigRepo) GetByOrganizationID(_ context.Context, _ int32) (*igDomain.InstagramConfig, error) {
	if f.cfg == nil {
		return nil, igDomain.ErrConfigNotFound
	}
	return f.cfg, nil
}
func (f *fakeIGConfigRepo) GetByVerifyToken(ctx context.Context, token string) (*igDomain.InstagramConfig, error) {
	if f.cfg != nil && f.cfg.VerifyToken == token {
		return f.cfg, nil
	}
	return nil, igDomain.ErrConfigNotFound
}
func (f *fakeIGConfigRepo) Create(_ context.Context, cfg *igDomain.InstagramConfig) (*igDomain.InstagramConfig, error) {
	f.cfg = cfg
	return cfg, nil
}
func (f *fakeIGConfigRepo) Update(_ context.Context, cfg *igDomain.InstagramConfig) (*igDomain.InstagramConfig, error) {
	f.cfg = cfg
	return cfg, nil
}

type fakeIGLogRepo struct {
	inserted     []*igDomain.WebhookLog
	outboxEvents []igDomain.OutboxEventInput
	duplicate    bool
	insertErr    error
	store        map[int32]*igDomain.WebhookLog
}

func (f *fakeIGLogRepo) Insert(_ context.Context, l *igDomain.WebhookLog) (*igDomain.WebhookLog, error) {
	if f.insertErr != nil {
		return nil, f.insertErr
	}
	f.inserted = append(f.inserted, l)
	return l, nil
}
func (f *fakeIGLogRepo) GetByID(_ context.Context, id int32) (*igDomain.WebhookLog, error) {
	if l, ok := f.store[id]; ok {
		return l, nil
	}
	return nil, igDomain.ErrWebhookLogNotFound
}
func (f *fakeIGLogRepo) InsertWithOutbox(_ context.Context, l *igDomain.WebhookLog, evs []igDomain.OutboxEventInput) (*igDomain.WebhookLog, error) {
	if f.duplicate {
		return nil, igDomain.ErrDuplicateDelivery
	}
	if f.insertErr != nil {
		return nil, f.insertErr
	}
	f.inserted = append(f.inserted, l)
	f.outboxEvents = append(f.outboxEvents, evs...)
	return l, nil
}
func (f *fakeIGLogRepo) GetStatsByOrganization(context.Context, int32) (*igDomain.WebhookLogStats, error) {
	return &igDomain.WebhookLogStats{}, nil
}

type fakeIGOutbox struct{ inserted int }

func (f *fakeIGOutbox) Insert(context.Context, string, json.RawMessage, *int32) (*outbox.OutboxEvent, error) {
	f.inserted++
	return &outbox.OutboxEvent{}, nil
}
func (f *fakeIGOutbox) ClaimPending(context.Context, int32) ([]*outbox.OutboxEvent, error) {
	return nil, nil
}
func (f *fakeIGOutbox) MarkDispatched(context.Context, int64) error           { return nil }
func (f *fakeIGOutbox) Retry(context.Context, int64, time.Time, string) error { return nil }
func (f *fakeIGOutbox) DeadLetter(context.Context, int64, string) error       { return nil }
func (f *fakeIGOutbox) ListDeadLetter(context.Context, int32) ([]*outbox.OutboxEvent, error) {
	return nil, nil
}
func (f *fakeIGOutbox) Requeue(context.Context, int64) error { return nil }
func (f *fakeIGOutbox) Prune(context.Context, time.Time) error {
	return nil
}

// ---------- helpers ----------

func signBody(secret string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

func igPayload(recipientID, mid, senderID, text string, echo bool) map[string]any {
	msg := map[string]any{
		"mid":     mid,
		"text":    text,
		"is_echo": echo,
	}
	return map[string]any{
		"object": "instagram",
		"entry": []any{
			map[string]any{
				"id":   "entry-1",
				"time": 1723000000,
				"changes": []any{
					map[string]any{
						"field": "messages",
						"value": map[string]any{
							"sender":    map[string]any{"id": senderID},
							"recipient": map[string]any{"id": recipientID},
							"timestamp": 1723000000,
							"messages":  []any{msg},
						},
					},
				},
			},
		},
	}
}

func newIGWebhookService(cfg *igDomain.InstagramConfig, logRepo igDomain.WebhookLogRepository, outboxRepo outbox.Repository, platformVerify string) WebhookService {
	cfgRepo := &fakeIGConfigRepo{cfg: cfg}
	return NewWebhookService(cfgRepo, logRepo, outboxRepo, platformVerify, noopLogger{})
}

type noopLogger struct{}

func (noopLogger) Debug(string, ...loggerdomain.Fields)               {}
func (noopLogger) Info(string, ...loggerdomain.Fields)                {}
func (noopLogger) Warn(string, ...loggerdomain.Fields)                {}
func (noopLogger) Error(string, ...loggerdomain.Fields)               {}
func (noopLogger) Fatal(string, ...loggerdomain.Fields)               {}
func (noopLogger) WithFields(loggerdomain.Fields) loggerdomain.Logger { return noopLogger{} }

// ---------- tests ----------

func TestProcessWebhook_ValidTextMessagePublishesInbound(t *testing.T) {
	cfg := &igDomain.InstagramConfig{
		OrganizationID: 7,
		IGUserID:       "business-ig-1",
		WebhookSecret:  "secret-1",
	}
	logRepo := &fakeIGLogRepo{}
	svc := newIGWebhookService(cfg, logRepo, &fakeIGOutbox{}, "")

	body, _ := json.Marshal(igPayload("business-ig-1", "mid.1", "customer-ig-1", "Hola!", false))
	payload := map[string]any{}
	_ = json.Unmarshal(body, &payload)

	err := svc.ProcessWebhook(context.Background(), body, payload, signBody("secret-1", body))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(logRepo.outboxEvents) != 1 {
		t.Fatalf("expected 1 outbox event, got %d", len(logRepo.outboxEvents))
	}
	if logRepo.outboxEvents[0].EventType != events.MessageReceivedEventType {
		t.Fatalf("expected %s, got %s", events.MessageReceivedEventType, logRepo.outboxEvents[0].EventType)
	}
	var ev events.MessageReceived
	if err := json.Unmarshal(logRepo.outboxEvents[0].Payload, &ev); err != nil {
		t.Fatalf("failed to decode event: %v", err)
	}
	if ev.FromIGUserID != "customer-ig-1" || ev.ToIGUserID != "business-ig-1" || ev.MessageSID != "mid.1" {
		t.Fatalf("unexpected event fields: %+v", ev)
	}
}

func TestProcessWebhook_EchoMessagePublishesEchoEvent(t *testing.T) {
	cfg := &igDomain.InstagramConfig{
		OrganizationID: 7,
		IGUserID:       "business-ig-1",
		WebhookSecret:  "secret-1",
	}
	logRepo := &fakeIGLogRepo{}
	svc := newIGWebhookService(cfg, logRepo, &fakeIGOutbox{}, "")

	body, _ := json.Marshal(igPayload("business-ig-1", "mid.echo", "customer-ig-1", "Hola!", true))
	payload := map[string]any{}
	_ = json.Unmarshal(body, &payload)

	if err := svc.ProcessWebhook(context.Background(), body, payload, signBody("secret-1", body)); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if logRepo.outboxEvents[0].EventType != events.MessageEchoEventType {
		t.Fatalf("expected %s, got %s", events.MessageEchoEventType, logRepo.outboxEvents[0].EventType)
	}
}

func TestProcessWebhook_InvalidSignatureRejected(t *testing.T) {
	cfg := &igDomain.InstagramConfig{
		OrganizationID: 7,
		IGUserID:       "business-ig-1",
		WebhookSecret:  "secret-1",
	}
	logRepo := &fakeIGLogRepo{}
	svc := newIGWebhookService(cfg, logRepo, &fakeIGOutbox{}, "")

	body, _ := json.Marshal(igPayload("business-ig-1", "mid.1", "customer-ig-1", "Hola!", false))
	payload := map[string]any{}
	_ = json.Unmarshal(body, &payload)

	err := svc.ProcessWebhook(context.Background(), body, payload, "sha256=deadbeef")
	if !errors.Is(err, igDomain.ErrInvalidSignature) {
		t.Fatalf("expected ErrInvalidSignature, got %v", err)
	}
	if len(logRepo.outboxEvents) != 0 {
		t.Fatalf("no outbox events expected, got %d", len(logRepo.outboxEvents))
	}
}

func TestProcessWebhook_UnknownIGUserRejected(t *testing.T) {
	logRepo := &fakeIGLogRepo{}
	svc := newIGWebhookService(nil, logRepo, &fakeIGOutbox{}, "")

	body, _ := json.Marshal(igPayload("unknown-ig", "mid.1", "customer-ig-1", "Hola!", false))
	payload := map[string]any{}
	_ = json.Unmarshal(body, &payload)

	err := svc.ProcessWebhook(context.Background(), body, payload, signBody("any", body))
	if !errors.Is(err, igDomain.ErrUnknownIGUser) {
		t.Fatalf("expected ErrUnknownIGUser, got %v", err)
	}
}

func TestProcessWebhook_MissingSignatureRejected(t *testing.T) {
	cfg := &igDomain.InstagramConfig{
		OrganizationID: 7,
		IGUserID:       "business-ig-1",
		WebhookSecret:  "secret-1",
	}
	logRepo := &fakeIGLogRepo{}
	svc := newIGWebhookService(cfg, logRepo, &fakeIGOutbox{}, "")

	body, _ := json.Marshal(igPayload("business-ig-1", "mid.1", "customer-ig-1", "Hola!", false))
	payload := map[string]any{}
	_ = json.Unmarshal(body, &payload)

	err := svc.ProcessWebhook(context.Background(), body, payload, "")
	if !errors.Is(err, igDomain.ErrInvalidSignature) {
		t.Fatalf("expected ErrInvalidSignature, got %v", err)
	}
}

func TestProcessWebhook_DuplicateDeliveryAcknowledged(t *testing.T) {
	cfg := &igDomain.InstagramConfig{
		OrganizationID: 7,
		IGUserID:       "business-ig-1",
		WebhookSecret:  "secret-1",
	}
	logRepo := &fakeIGLogRepo{duplicate: true}
	svc := newIGWebhookService(cfg, logRepo, &fakeIGOutbox{}, "")

	body, _ := json.Marshal(igPayload("business-ig-1", "mid.1", "customer-ig-1", "Hola!", false))
	payload := map[string]any{}
	_ = json.Unmarshal(body, &payload)

	if err := svc.ProcessWebhook(context.Background(), body, payload, signBody("secret-1", body)); err != nil {
		t.Fatalf("duplicate delivery must be acknowledged without error, got %v", err)
	}
	// No re-dispatch: the outbox was never touched for events.
	if len(logRepo.outboxEvents) != 0 {
		t.Fatalf("no outbox events expected on duplicate, got %d", len(logRepo.outboxEvents))
	}
}

func TestVerifyChallenge_PlatformTokenAccepted(t *testing.T) {
	svc := newIGWebhookService(nil, &fakeIGLogRepo{}, &fakeIGOutbox{}, "platform-token")
	if err := svc.VerifyChallenge(context.Background(), "subscribe", "platform-token", "challenge"); err != nil {
		t.Fatalf("expected platform token to verify, got %v", err)
	}
}

func TestVerifyChallenge_PerConfigTokenAccepted(t *testing.T) {
	cfg := &igDomain.InstagramConfig{
		OrganizationID: 7,
		IGUserID:       "business-ig-1",
		VerifyToken:    "org-token",
	}
	svc := newIGWebhookService(cfg, &fakeIGLogRepo{}, &fakeIGOutbox{}, "")
	if err := svc.VerifyChallenge(context.Background(), "subscribe", "org-token", "challenge"); err != nil {
		t.Fatalf("expected per-config token to verify, got %v", err)
	}
}

func TestVerifyChallenge_InvalidTokenRejected(t *testing.T) {
	svc := newIGWebhookService(nil, &fakeIGLogRepo{}, &fakeIGOutbox{}, "platform-token")
	err := svc.VerifyChallenge(context.Background(), "subscribe", "wrong", "challenge")
	if !errors.Is(err, igDomain.ErrWebhookVerificationFail) {
		t.Fatalf("expected ErrWebhookVerificationFail, got %v", err)
	}
}
