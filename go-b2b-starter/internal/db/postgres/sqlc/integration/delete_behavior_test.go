//go:build integration

package integration

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/moasq/go-b2b-starter/internal/db/helpers"
	sqlc "github.com/moasq/go-b2b-starter/internal/db/postgres/sqlc/gen"
)

// Delete behavior must be preserved: account deletion nulls assignment columns
// only; contact deletion cascades conversations and messages; company deletion
// nulls company_id on contacts and deals; stage deletion nulls deals.stage_id
// and preserves pipeline_id.

func TestAccountDeletionNullsAssignments(t *testing.T) {
	ctx := context.Background()
	q := testStore
	orgA, accA := createOrgWithAccount(t, ctx, q)

	contact, err := q.UpsertContact(ctx, sqlc.UpsertContactParams{
		OrganizationID: orgA,
		PhoneNumber:    "+573002222001",
		DisplayName:    helpers.ToPgText("C1"),
		LastMessageAt:  helpers.ToPgTimestampPtr(nil),
	})
	if err != nil {
		t.Fatalf("seed contact: %v", err)
	}
	if _, err := q.UpdateContact(ctx, sqlc.UpdateContactParams{
		ID:             contact.ID,
		OrganizationID: orgA,
		AssignedTo:     helpers.ToPgInt4Ptr(&accA),
	}); err != nil {
		t.Fatalf("assign contact: %v", err)
	}

	pipeline, err := q.CreatePipeline(ctx, sqlc.CreatePipelineParams{
		OrganizationID:   orgA,
		Nombre:           "P Del",
		EsPredeterminado: false,
		Orden:            0,
	})
	if err != nil {
		t.Fatalf("seed pipeline: %v", err)
	}
	stage, err := q.CreatePipelineStage(ctx, sqlc.CreatePipelineStageParams{
		PipelineID: pipeline.ID,
		Nombre:     "S1",
		Orden:      1,
	})
	if err != nil {
		t.Fatalf("seed stage: %v", err)
	}
	deal, err := q.CreateDeal(ctx, sqlc.CreateDealParams{
		OrganizationID: orgA,
		Nombre:         "D1",
		PipelineID:     pipeline.ID,
		StageID:        helpers.ToPgInt4Ptr(&stage.ID),
		Estado:         "abierto",
		AssignedTo:     helpers.ToPgInt4Ptr(&accA),
	})
	if err != nil {
		t.Fatalf("seed deal: %v", err)
	}

	company, err := q.CreateCompany(ctx, sqlc.CreateCompanyParams{
		OrganizationID: orgA,
		Name:           "Co1",
		OwnerAccountID: helpers.ToPgInt4Ptr(&accA),
	})
	if err != nil {
		t.Fatalf("seed company: %v", err)
	}

	// NOTE: the app's DeleteAccount query is a SOFT delete (status='inactive'),
	// so it must NOT null assignments. This test exercises the FK's hard-delete
	// semantics directly, since the composite FK carries ON DELETE SET NULL
	// (assigned_to).
	if _, err := testPool.Exec(ctx, "DELETE FROM organizations.accounts WHERE id = $1", accA); err != nil {
		t.Fatalf("hard delete account: %v", err)
	}

	got, err := q.GetContactByID(ctx, sqlc.GetContactByIDParams{ID: contact.ID, OrganizationID: orgA})
	if err != nil {
		t.Fatalf("get contact: %v", err)
	}
	if got.AssignedTo.Valid {
		t.Fatalf("expected contacts.assigned_to NULL after account deletion, got %+v", got.AssignedTo)
	}

	gotDeal, err := q.GetDealByID(ctx, sqlc.GetDealByIDParams{ID: deal.ID, OrganizationID: orgA})
	if err != nil {
		t.Fatalf("get deal: %v", err)
	}
	if gotDeal.AssignedTo.Valid {
		t.Fatalf("expected deals.assigned_to NULL after account deletion, got %+v", gotDeal.AssignedTo)
	}

	gotCo, err := q.GetCompanyByID(ctx, sqlc.GetCompanyByIDParams{ID: company.ID, OrganizationID: orgA})
	if err != nil {
		t.Fatalf("get company: %v", err)
	}
	if gotCo.OwnerAccountID.Valid {
		t.Fatalf("expected companies.owner_account_id NULL after account deletion, got %+v", gotCo.OwnerAccountID)
	}
}

func TestContactDeletionCascadesConversationsAndMessages(t *testing.T) {
	ctx := context.Background()
	q := testStore
	orgA, _ := createOrgWithAccount(t, ctx, q)

	contact, err := q.UpsertContact(ctx, sqlc.UpsertContactParams{
		OrganizationID: orgA,
		PhoneNumber:    "+573002222002",
		DisplayName:    helpers.ToPgText("C2"),
		LastMessageAt:  helpers.ToPgTimestampPtr(nil),
	})
	if err != nil {
		t.Fatalf("seed contact: %v", err)
	}
	conv, err := q.CreateConversation(ctx, sqlc.CreateConversationParams{
		OrganizationID: orgA,
		ContactID:      contact.ID,
		Status:         "active",
	})
	if err != nil {
		t.Fatalf("seed conversation: %v", err)
	}
	if _, err := q.CreateMessage(ctx, sqlc.CreateMessageParams{
		OrganizationID:    orgA,
		ConversationID:    conv.ID,
		ContactID:         contact.ID,
		WhatsappMessageID: helpers.ToPgText("wamid-del-1"),
		Direction:         "inbound",
		MessageType:       "text",
		Content:           helpers.ToPgText("hi"),
		Status:            "received",
	}); err != nil {
		t.Fatalf("seed message: %v", err)
	}

	if err := q.DeleteContact(ctx, sqlc.DeleteContactParams{ID: contact.ID, OrganizationID: orgA}); err != nil {
		t.Fatalf("delete contact: %v", err)
	}

	_, err = testStore.GetConversationByID(ctx, sqlc.GetConversationByIDParams{ID: conv.ID, OrganizationID: orgA})
	if !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("expected conversation gone after contact deletion, got: %v", err)
	}
	msgs, err := q.ListMessagesByConversation(ctx, sqlc.ListMessagesByConversationParams{
		ConversationID: conv.ID,
		OrganizationID: orgA,
		Limit:          10,
		Offset:         0,
	})
	if err != nil {
		t.Fatalf("list messages: %v", err)
	}
	if len(msgs) != 0 {
		t.Fatalf("expected messages cascade-deleted, got %d", len(msgs))
	}
}

func TestCompanyDeletionNullsCompanyID(t *testing.T) {
	ctx := context.Background()
	q := testStore
	orgA, _ := createOrgWithAccount(t, ctx, q)

	company, err := q.CreateCompany(ctx, sqlc.CreateCompanyParams{
		OrganizationID: orgA,
		Name:           "Co2",
	})
	if err != nil {
		t.Fatalf("seed company: %v", err)
	}

	contact, err := q.UpsertContact(ctx, sqlc.UpsertContactParams{
		OrganizationID: orgA,
		PhoneNumber:    "+573002222003",
		DisplayName:    helpers.ToPgText("C3"),
		LastMessageAt:  helpers.ToPgTimestampPtr(nil),
	})
	if err != nil {
		t.Fatalf("seed contact: %v", err)
	}
	if _, err := q.UpdateContact(ctx, sqlc.UpdateContactParams{
		ID:             contact.ID,
		OrganizationID: orgA,
		CompanyID:      helpers.ToPgInt4Ptr(&company.ID),
	}); err != nil {
		t.Fatalf("assign contact to company: %v", err)
	}

	pipeline, err := q.CreatePipeline(ctx, sqlc.CreatePipelineParams{
		OrganizationID:   orgA,
		Nombre:           "P CoDel",
		EsPredeterminado: false,
		Orden:            0,
	})
	if err != nil {
		t.Fatalf("seed pipeline: %v", err)
	}
	stage, err := q.CreatePipelineStage(ctx, sqlc.CreatePipelineStageParams{
		PipelineID: pipeline.ID,
		Nombre:     "S1",
		Orden:      1,
	})
	if err != nil {
		t.Fatalf("seed stage: %v", err)
	}
	deal, err := q.CreateDeal(ctx, sqlc.CreateDealParams{
		OrganizationID: orgA,
		Nombre:         "D Co",
		PipelineID:     pipeline.ID,
		StageID:        helpers.ToPgInt4Ptr(&stage.ID),
		CompanyID:      helpers.ToPgInt4Ptr(&company.ID),
		Estado:         "abierto",
	})
	if err != nil {
		t.Fatalf("seed deal: %v", err)
	}

	if err := q.DeleteCompany(ctx, sqlc.DeleteCompanyParams{ID: company.ID, OrganizationID: orgA}); err != nil {
		t.Fatalf("delete company: %v", err)
	}

	gotContact, err := q.GetContactByID(ctx, sqlc.GetContactByIDParams{ID: contact.ID, OrganizationID: orgA})
	if err != nil {
		t.Fatalf("get contact: %v", err)
	}
	if gotContact.CompanyID.Valid {
		t.Fatalf("expected contacts.company_id NULL after company deletion, got %+v", gotContact.CompanyID)
	}

	gotDeal, err := q.GetDealByID(ctx, sqlc.GetDealByIDParams{ID: deal.ID, OrganizationID: orgA})
	if err != nil {
		t.Fatalf("get deal: %v", err)
	}
	if gotDeal.CompanyID.Valid {
		t.Fatalf("expected deals.company_id NULL after company deletion, got %+v", gotDeal.CompanyID)
	}
}
