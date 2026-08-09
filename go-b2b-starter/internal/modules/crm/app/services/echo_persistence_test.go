package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/moasq/go-b2b-starter/internal/modules/crm/domain"
	whatsappEvents "github.com/moasq/go-b2b-starter/internal/modules/whatsapp/domain/events"
	"github.com/moasq/go-b2b-starter/internal/platform/features"
	loggerdomain "github.com/moasq/go-b2b-starter/internal/platform/logger/domain"
)

type noopLogger struct{}

func (noopLogger) Debug(string, ...loggerdomain.Fields) {}
func (noopLogger) Info(string, ...loggerdomain.Fields)  {}
func (noopLogger) Warn(string, ...loggerdomain.Fields)  {}
func (noopLogger) Error(string, ...loggerdomain.Fields) {}
func (noopLogger) Fatal(string, ...loggerdomain.Fields) {}
func (noopLogger) WithFields(fields loggerdomain.Fields) loggerdomain.Logger {
	return noopLogger{}
}

type fakeEchoContactRepo struct {
	domain.ContactRepository
	upserted *domain.Contact
}

func (f *fakeEchoContactRepo) UpsertByPhone(_ context.Context, c *domain.Contact) (*domain.Contact, error) {
	f.upserted = c
	c.ID = 1
	return c, nil
}

type fakeEchoConvRepo struct {
	domain.ConversationRepository
	conv    *domain.Conversation
	updated bool
}

func (f *fakeEchoConvRepo) EnsureActive(_ context.Context, c *domain.Conversation) (*domain.Conversation, error) {
	c.ID = 1
	f.conv = c
	return c, nil
}

func (f *fakeEchoConvRepo) UpdateLastMessageAt(_ context.Context, _ int32, _ int32, _ *time.Time) (*domain.Conversation, error) {
	f.updated = true
	return f.conv, nil
}

type fakeEchoMsgRepo struct {
	domain.MessageRepository
	inserted  bool
	lastMsg   *domain.Message
	insertCalls int
}

func (f *fakeEchoMsgRepo) InsertIdempotent(_ context.Context, m *domain.Message) (*domain.Message, bool, error) {
	f.insertCalls++
	f.lastMsg = m
	if !f.inserted {
		f.inserted = true
		m.ID = 1
		return m, true, nil
	}
	return m, false, nil
}

type fakeEchoActivityRepo struct {
	domain.ActivityRepository
	created []*domain.Activity
}

func (f *fakeEchoActivityRepo) Create(_ context.Context, a *domain.Activity) (*domain.Activity, error) {
	f.created = append(f.created, a)
	return a, nil
}

type noEntitlement struct{}

func (noEntitlement) GetEntitlement(context.Context, int32) (*features.Entitlement, error) {
	return nil, nil
}

func newEchoCRMService(msgRepo *fakeEchoMsgRepo, actRepo *fakeEchoActivityRepo) CRMService {
	return NewCRMService(
		&fakeEchoContactRepo{},
		&fakeEchoConvRepo{},
		msgRepo,
		actRepo,
		noEntitlement{},
		noopLogger{},
	)
}

func TestProcessEchoMessage_PersistsOutbound(t *testing.T) {
	msgRepo := &fakeEchoMsgRepo{}
	actRepo := &fakeEchoActivityRepo{}
	svc := newEchoCRMService(msgRepo, actRepo)

	ts := time.Unix(1723000000, 0)
	err := svc.ProcessEchoMessage(context.Background(), &whatsappEvents.MessageEcho{
		OrganizationID:    7,
		MessageSID:        "wamid.echo1",
		From:              "573001234567",
		To:                "573009999999",
		MessageType:       "text",
		Content:           "sent from the phone app",
		WhatsAppTimestamp: ts,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if msgRepo.lastMsg == nil {
		t.Fatal("expected message insert")
	}
	if msgRepo.lastMsg.Direction != domain.MessageDirectionOutbound {
		t.Fatalf("expected outbound direction, got %s", msgRepo.lastMsg.Direction)
	}
	if msgRepo.lastMsg.OrganizationID != 7 || msgRepo.lastMsg.ContactID != 1 || msgRepo.lastMsg.WhatsAppMessageID != "wamid.echo1" {
		t.Fatalf("unexpected message: %+v", msgRepo.lastMsg)
	}
	if msgRepo.lastMsg.MessageData["origin"] != "echo" {
		t.Fatalf("expected origin echo in message data, got %v", msgRepo.lastMsg.MessageData)
	}
	if len(actRepo.created) != 0 {
		t.Fatalf("expected no activity without crm_activities entitlement, got %d", len(actRepo.created))
	}
}

func TestProcessEchoMessage_DuplicateIsNoop(t *testing.T) {
	msgRepo := &fakeEchoMsgRepo{}
	actRepo := &fakeEchoActivityRepo{}
	svc := newEchoCRMService(msgRepo, actRepo)

	ev := &whatsappEvents.MessageEcho{
		OrganizationID:    7,
		MessageSID:        "wamid.echo1",
		From:              "573001234567",
		MessageType:       "text",
		Content:           "dup",
		WhatsAppTimestamp: time.Now(),
	}

	if err := svc.ProcessEchoMessage(context.Background(), ev); err != nil {
		t.Fatalf("first delivery: %v", err)
	}
	if err := svc.ProcessEchoMessage(context.Background(), ev); err != nil {
		t.Fatalf("duplicate delivery must be a no-op, got: %v", err)
	}
	if msgRepo.insertCalls != 2 {
		t.Fatalf("expected 2 idempotent insert calls, got %d", msgRepo.insertCalls)
	}
}

func TestEchoListener_ErrorPropagated(t *testing.T) {
	failing := &fakeEchoMsgRepo{}
	// Force failure by making the message invalid (no conversation).
	svc := NewCRMService(
		&fakeEchoContactRepo{},
		&fakeEchoConvRepo{},
		&failingMsgRepo{failing},
		&fakeEchoActivityRepo{},
		noEntitlement{},
		noopLogger{},
	)
	listener := NewEchoListener(svc, noopLogger{})

	err := listener.HandleMessageEcho(context.Background(), &whatsappEvents.MessageEcho{
		OrganizationID:    7,
		MessageSID:        "wamid.echo9",
		From:              "573001234567",
		MessageType:       "text",
		WhatsAppTimestamp: time.Now(),
	})
	if err == nil {
		t.Fatal("expected error to propagate from listener")
	}
}

type failingMsgRepo struct {
	*fakeEchoMsgRepo
}

func (f *failingMsgRepo) InsertIdempotent(ctx context.Context, m *domain.Message) (*domain.Message, bool, error) {
	return nil, false, errors.New("insert failed")
}
