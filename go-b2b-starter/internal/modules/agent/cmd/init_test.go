package cmd

import (
	"context"
	"errors"
	"testing"

	whatsappEvents "github.com/moasq/go-b2b-starter/internal/modules/whatsapp/domain/events"
)

// fakeChecker implements agent.ActiveInquiryChecker for the skip-path tests.
type fakeChecker struct {
	active bool
	err    error
}

func (f *fakeChecker) IsActiveRecipientByPhone(ctx context.Context, orgID int32, phone string) (bool, error) {
	if f.err != nil {
		return false, f.err
	}
	return f.active, nil
}

func TestShouldSkipInquiry(t *testing.T) {
	ctx := context.Background()
	checker := &fakeChecker{active: true}

	ev := &whatsappEvents.MessageReceived{
		OrganizationID: 42,
		From:           "+573001234567",
		MessageType:    "text",
		Content:        "Disponible",
		MessageSID:     "wamid.inbound.1",
	}
	skip, err := shouldSkipInquiry(ctx, checker, ev)
	if err != nil {
		t.Fatalf("shouldSkipInquiry: %v", err)
	}
	if !skip {
		t.Fatalf("expected skip for an active inquiry-run recipient")
	}

	checker.active = false
	skip, err = shouldSkipInquiry(ctx, checker, ev)
	if err != nil {
		t.Fatalf("shouldSkipInquiry: %v", err)
	}
	if skip {
		t.Fatalf("expected no skip for a non-recipient")
	}

	// Non-text messages never trigger the skip path (and never skip).
	nonText := &whatsappEvents.MessageReceived{OrganizationID: 42, From: "+573001234567", MessageType: "image"}
	skip, err = shouldSkipInquiry(ctx, checker, nonText)
	if err != nil {
		t.Fatalf("shouldSkipInquiry non-text: %v", err)
	}
	if skip {
		t.Fatalf("expected no skip for non-text messages")
	}

	// Lookup errors return (false, err) so the caller fails safe (no skip).
	checker.err = errors.New("db down")
	_, err = shouldSkipInquiry(ctx, checker, ev)
	if err == nil {
		t.Fatalf("expected lookup error to propagate")
	}
}
