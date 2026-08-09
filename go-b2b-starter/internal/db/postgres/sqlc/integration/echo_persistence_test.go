//go:build integration

package integration

import (
	"context"
	"sync"
	"testing"

	"github.com/jackc/pgx/v5"

	"github.com/moasq/go-b2b-starter/internal/db/helpers"
	sqlc "github.com/moasq/go-b2b-starter/internal/db/postgres/sqlc/gen"
)

// Echo (coexistence mirror) persistence: outbound messages use the same
// InsertMessageIdempotent guard as inbound ones, so concurrent deliveries of
// the same echo yield exactly one crm.messages row.

func TestEchoMessageConcurrentDeliveriesYieldOneRow(t *testing.T) {
	ctx := context.Background()
	orgA, _ := createOrgWithAccount(t, ctx, testStore)
	contactID, convID := seedContactAndConversation(t, ctx, orgA, "+57300555666")

	params := func() sqlc.InsertMessageIdempotentParams {
		return sqlc.InsertMessageIdempotentParams{
			OrganizationID:    orgA,
			ConversationID:    convID,
			ContactID:         contactID,
			WhatsappMessageID: helpers.ToPgText("wamid-echo-1"),
			Direction:         "outbound",
			MessageType:       "text",
			Content:           helpers.ToPgText("sent from the phone app"),
			Status:            "received",
			MessageData:       helpers.ToJSONB(map[string]any{"origin": "echo"}),
		}
	}

	const workers = 8
	var wg sync.WaitGroup
	inserted := make([]bool, workers)
	errs := make([]error, workers)
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			row, err := testStore.InsertMessageIdempotent(ctx, params())
			if err != nil {
				errs[i] = err
				return
			}
			inserted[i] = row.MessageData != nil
		}(i)
	}
	wg.Wait()

	insertCount := 0
	for i, err := range errs {
		if err != nil {
			t.Fatalf("worker %d insert failed: %v", i, err)
		}
		if inserted[i] {
			insertCount++
		}
	}
	if insertCount != 1 {
		t.Fatalf("expected exactly 1 successful insert out of %d workers, got %d", workers, insertCount)
	}

	rows, err := testStore.ListMessagesByConversation(ctx, sqlc.ListMessagesByConversationParams{
		OrganizationID: orgA,
		ConversationID: convID,
	})
	if err != nil {
		t.Fatalf("list messages: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected exactly 1 message row, got %d", len(rows))
	}
	if rows[0].Direction != "outbound" {
		t.Fatalf("expected direction outbound, got %s", rows[0].Direction)
	}
	if rows[0].WhatsappMessageID.String != "wamid-echo-1" {
		t.Fatalf("expected wamid-echo-1, got %s", rows[0].WhatsappMessageID.String)
	}
}

func TestEchoMessageDuplicateSequentialReturnsExisting(t *testing.T) {
	ctx := context.Background()
	orgA, _ := createOrgWithAccount(t, ctx, testStore)
	contactID, convID := seedContactAndConversation(t, ctx, orgA, "+57300555667")

	params := func() sqlc.InsertMessageIdempotentParams {
		return sqlc.InsertMessageIdempotentParams{
			OrganizationID:    orgA,
			ConversationID:    convID,
			ContactID:         contactID,
			WhatsappMessageID: helpers.ToPgText("wamid-echo-2"),
			Direction:         "outbound",
			MessageType:       "text",
			Content:           helpers.ToPgText("dup echo"),
			Status:            "received",
		}
	}

	if _, err := testStore.InsertMessageIdempotent(ctx, params()); err != nil {
		t.Fatalf("first insert: %v", err)
	}

	// Second delivery: InsertMessageIdempotent returns no row, and the
	// repository fallback (GetMessageByWhatsAppID) fetches the existing row.
	_, err := testStore.InsertMessageIdempotent(ctx, params())
	if err == nil {
		t.Fatal("expected pgx.ErrNoRows on duplicate idempotent insert")
	}
	if err != pgx.ErrNoRows {
		t.Fatalf("expected pgx.ErrNoRows, got %v", err)
	}

	existing, err := testStore.GetMessageByWhatsAppID(ctx, sqlc.GetMessageByWhatsAppIDParams{
		OrganizationID:    orgA,
		WhatsappMessageID: helpers.ToPgText("wamid-echo-2"),
	})
	if err != nil {
		t.Fatalf("fetch existing: %v", err)
	}
	if existing.Direction != "outbound" || existing.Content.String != "dup echo" {
		t.Fatalf("unexpected existing row: %+v", existing)
	}
}
