package services

import (
	"context"
	"errors"
	"testing"

	"github.com/moasq/go-b2b-starter/internal/modules/crm/domain"
	whatsappDomain "github.com/moasq/go-b2b-starter/internal/modules/whatsapp/domain"
	"github.com/moasq/go-b2b-starter/internal/modules/whatsapp/domain/events"
)

type fakeSender struct {
	msgID string
	err   error
}

func (f *fakeSender) SendTextMessage(context.Context, string, string, string, string, string, string) (string, error) {
	return f.msgID, f.err
}

type fakeSendMsgRepo struct {
	domain.MessageRepository
	status string
	msgID  string
	lastID int32
	err    error
}

func (f *fakeSendMsgRepo) UpdateStatus(_ context.Context, id int32, status, whatsappMessageID string) (*domain.Message, error) {
	if f.err != nil {
		return nil, f.err
	}
	f.lastID = id
	f.status = status
	f.msgID = whatsappMessageID
	return &domain.Message{ID: id, Status: status}, nil
}

type fakeSendConfigRepo struct {
	whatsappDomain.ConfigRepository
	cfg *whatsappDomain.WhatsAppConfig
}

func (f *fakeSendConfigRepo) GetByOrganizationID(context.Context, int32) (*whatsappDomain.WhatsAppConfig, error) {
	if f.cfg == nil {
		return nil, errors.New("not configured")
	}
	return f.cfg, nil
}

func newTestHandler(t *testing.T, cfg *whatsappDomain.WhatsAppConfig, sender textSender) (*MessageSendHandler, *fakeSendMsgRepo) {
	t.Helper()
	msgRepo := &fakeSendMsgRepo{}
	cfgRepo := &fakeSendConfigRepo{cfg: cfg}
	h := NewMessageSendHandler(msgRepo, cfgRepo, nil, nil, noopLogger{})
	h.whatsappSender = sender
	return h, msgRepo
}

func TestMessageSendHandler_SuccessMarksSent(t *testing.T) {
	cfg := &whatsappDomain.WhatsAppConfig{
		OrganizationID: 7,
		PhoneNumberID:  "phone-1",
		AccessToken:    "token",
		IsActive:       true,
	}
	h, msgRepo := newTestHandler(t, cfg, &fakeSender{msgID: "wamid.1"})

	send := events.NewMessageSend(7, 1, 42, "573001234567", "hello")
	if err := h.Handle(context.Background(), send); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if msgRepo.lastID != 42 || msgRepo.status != "sent" || msgRepo.msgID != "wamid.1" {
		t.Fatalf("expected message 42 marked sent with wamid.1, got id=%d status=%q msgID=%q", msgRepo.lastID, msgRepo.status, msgRepo.msgID)
	}
}

func TestMessageSendHandler_TransientSendFailureRetried(t *testing.T) {
	cfg := &whatsappDomain.WhatsAppConfig{
		OrganizationID: 7,
		PhoneNumberID:  "phone-1",
		AccessToken:    "token",
		IsActive:       true,
	}
	h, msgRepo := newTestHandler(t, cfg, &fakeSender{err: errors.New("timeout")})

	send := events.NewMessageSend(7, 1, 42, "573001234567", "hello")
	if err := h.Handle(context.Background(), send); err == nil {
		t.Fatal("expected error to trigger dispatcher retry")
	}
	if msgRepo.status != "failed" {
		t.Fatalf("expected message marked failed, got %q", msgRepo.status)
	}
}

func TestMessageSendHandler_InactiveConfigFailsMessage(t *testing.T) {
	cfg := &whatsappDomain.WhatsAppConfig{OrganizationID: 7, IsActive: false}
	h, msgRepo := newTestHandler(t, cfg, &fakeSender{})

	send := events.NewMessageSend(7, 1, 42, "573001234567", "hello")
	if err := h.Handle(context.Background(), send); err != nil {
		t.Fatalf("permanent failure should complete without error, got: %v", err)
	}
	if msgRepo.status != "failed" {
		t.Fatalf("expected message marked failed, got %q", msgRepo.status)
	}
}

func TestMessageSendHandler_ConfigResolutionErrorRetried(t *testing.T) {
	h, msgRepo := newTestHandler(t, nil, &fakeSender{})

	send := events.NewMessageSend(7, 1, 42, "573001234567", "hello")
	if err := h.Handle(context.Background(), send); err == nil {
		t.Fatal("expected error when config cannot be resolved")
	}
	if msgRepo.status != "" {
		t.Fatalf("unresolved config must not mark message, got %q", msgRepo.status)
	}
}
