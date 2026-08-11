//go:build integration

package integration

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v5/pgtype"
	"sync"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/moasq/go-b2b-starter/internal/db/helpers"
	sqlc "github.com/moasq/go-b2b-starter/internal/db/postgres/sqlc/gen"
)

// Message idempotency: InsertMessageIdempotent inserts once; a duplicate insert
// returns no row; the fallback fetch returns the existing message; concurrent
// inserts with the same (organization_id, whatsapp_message_id) yield one row.

func seedContactAndConversation(t *testing.T, ctx context.Context, orgID int32, phone string) (int32, int32) {
	t.Helper()
	contact, err := testStore.UpsertContact(ctx, sqlc.UpsertContactParams{
		OrganizationID: orgID,
		PhoneNumber:    pgtype.Text{String: phone, Valid: true},
		DisplayName:    helpers.ToPgText("IT Contact"),
		LastMessageAt:  helpers.ToPgTimestampPtr(nil),
	})
	if err != nil {
		t.Fatalf("seed contact: %v", err)
	}
	conv, err := testStore.CreateConversation(ctx, sqlc.CreateConversationParams{
		OrganizationID: orgID,
		ContactID:      contact.ID,
		Channel:        "whatsapp",
		Status:         "active",
	})
	if err != nil {
		t.Fatalf("seed conversation: %v", err)
	}
	return contact.ID, conv.ID
}

func TestMessageIdempotentInsertSingleAndDuplicate(t *testing.T) {
	ctx := context.Background()
	orgA, _ := createOrgWithAccount(t, ctx, testStore)
	contactID, convID := seedContactAndConversation(t, ctx, orgA, "+573003333001")

	params := func() sqlc.InsertMessageIdempotentParams {
		return sqlc.InsertMessageIdempotentParams{
			OrganizationID:    orgA,
			ConversationID:    convID,
			ContactID:         contactID,
			Channel:           "whatsapp",
			ProviderMessageID: helpers.ToPgText("wamid-dup-1"),
			Direction:         "inbound",
			MessageType:       "text",
			Content:           helpers.ToPgText("hola"),
			Status:            "received",
		}
	}

	first, err := testStore.InsertMessageIdempotent(ctx, params())
	if err != nil {
		t.Fatalf("first idempotent insert: %v", err)
	}
	if first.ID == 0 {
		t.Fatal("expected inserted message")
	}

	second, err := testStore.InsertMessageIdempotent(ctx, params())
	if err == nil {
		t.Fatalf("expected pgx.ErrNoRows on duplicate insert, got row %d", second.ID)
	}
	if err != pgx.ErrNoRows {
		t.Fatalf("expected pgx.ErrNoRows, got: %v", err)
	}

	existing, err := testStore.GetMessageByProviderID(ctx, sqlc.GetMessageByProviderIDParams{
		OrganizationID:    orgA,
		Channel:           "whatsapp",
		ProviderMessageID: helpers.ToPgText("wamid-dup-1"),
	})
	if err != nil {
		t.Fatalf("fallback fetch: %v", err)
	}
	if existing.ID != first.ID {
		t.Fatalf("expected fallback to return the original row %d, got %d", first.ID, existing.ID)
	}

	count := countRows(t, ctx, fmt.Sprintf(
		"SELECT count(*) FROM crm.messages WHERE organization_id = %d AND provider_message_id = 'wamid-dup-1'", orgA))
	if count != 1 {
		t.Fatalf("expected exactly 1 row, got %d", count)
	}
}

func TestMessageIdempotentConcurrentInserts(t *testing.T) {
	ctx := context.Background()
	orgA, _ := createOrgWithAccount(t, ctx, testStore)
	contactID, convID := seedContactAndConversation(t, ctx, orgA, "+573003333002")

	const workers = 8
	var wg sync.WaitGroup
	errs := make(chan error, workers)
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := testStore.InsertMessageIdempotent(ctx, sqlc.InsertMessageIdempotentParams{
				OrganizationID:    orgA,
				ConversationID:    convID,
				ContactID:         contactID,
				Channel:           "whatsapp",
				ProviderMessageID: helpers.ToPgText("wamid-conc-1"),
				Direction:         "inbound",
				MessageType:       "text",
				Content:           helpers.ToPgText("hi"),
				Status:            "received",
			})
			if err != nil && err != pgx.ErrNoRows {
				errs <- err
			}
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Fatalf("concurrent insert error: %v", err)
	}

	count := countRows(t, ctx, fmt.Sprintf(
		"SELECT count(*) FROM crm.messages WHERE organization_id = %d AND provider_message_id = 'wamid-conc-1'", orgA))
	if count != 1 {
		t.Fatalf("expected exactly 1 row after concurrent inserts, got %d", count)
	}
}

// Conversation idempotency: one active conversation per contact; concurrent
// creation yields one active row; closing permits a new active conversation.

func TestConversationOneActivePerContact(t *testing.T) {
	ctx := context.Background()
	orgA, _ := createOrgWithAccount(t, ctx, testStore)
	contactID, _ := seedContactAndConversation(t, ctx, orgA, "+573003333003")

	// Second active conversation for the same contact must conflict.
	_, err := testStore.InsertActiveConversationIdempotent(ctx, sqlc.InsertActiveConversationIdempotentParams{
		OrganizationID: orgA,
		ContactID:      contactID,
		Channel:           "whatsapp",
		LastMessageAt:  helpers.ToPgTimestampPtr(nil),
	})
	if err != pgx.ErrNoRows {
		t.Fatalf("expected pgx.ErrNoRows for duplicate active conversation, got: %v", err)
	}

	// Closing the active conversation permits a new one.
	active, err := testStore.GetActiveConversationByContact(ctx, sqlc.GetActiveConversationByContactParams{
		ContactID:      contactID,
		OrganizationID: orgA,
		Channel:        "whatsapp",
	})
	if err != nil {
		t.Fatalf("get active conversation: %v", err)
	}
	if _, err := testStore.UpdateConversationStatus(ctx, sqlc.UpdateConversationStatusParams{
		ID:             active.ID,
		OrganizationID: orgA,
		Status:         "closed",
	}); err != nil {
		t.Fatalf("close conversation: %v", err)
	}

	second, err := testStore.InsertActiveConversationIdempotent(ctx, sqlc.InsertActiveConversationIdempotentParams{
		OrganizationID: orgA,
		ContactID:      contactID,
		Channel:           "whatsapp",
		LastMessageAt:  helpers.ToPgTimestampPtr(nil),
	})
	if err != nil {
		t.Fatalf("expected new active conversation after close, got: %v", err)
	}
	if second.ID == 0 || second.ID == active.ID {
		t.Fatalf("expected a new active conversation, got %d", second.ID)
	}
}

func TestConversationConcurrentEnsureActive(t *testing.T) {
	ctx := context.Background()
	orgA, _ := createOrgWithAccount(t, ctx, testStore)

	contact, err := testStore.UpsertContact(ctx, sqlc.UpsertContactParams{
		OrganizationID: orgA,
		PhoneNumber:    pgtype.Text{String: "+573003333004", Valid: true},
		DisplayName:    helpers.ToPgText("Conc Contact"),
		LastMessageAt:  helpers.ToPgTimestampPtr(nil),
	})
	if err != nil {
		t.Fatalf("seed contact: %v", err)
	}

	const workers = 8
	var wg sync.WaitGroup
	errs := make(chan error, workers)
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := testStore.InsertActiveConversationIdempotent(ctx, sqlc.InsertActiveConversationIdempotentParams{
				OrganizationID: orgA,
				ContactID:      contact.ID,
		Channel:           "whatsapp",
				LastMessageAt:  helpers.ToPgTimestampPtr(nil),
			})
			if err != nil && err != pgx.ErrNoRows {
				errs <- err
			}
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Fatalf("concurrent ensure-active error: %v", err)
	}

	count := countRows(t, ctx, fmt.Sprintf(
		"SELECT count(*) FROM crm.conversations WHERE organization_id = %d AND contact_id = %d AND status = 'active'", orgA, contact.ID))
	if count != 1 {
		t.Fatalf("expected exactly 1 active conversation, got %d", count)
	}
}

func countRows(t *testing.T, ctx context.Context, query string) int {
	t.Helper()
	var count int
	if err := testPool.QueryRow(ctx, query).Scan(&count); err != nil {
		t.Fatalf("count query: %v", err)
	}
	return count
}
