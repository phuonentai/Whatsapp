//go:build integration

package integration

import (
	"context"
	"testing"
	"time"

	"github.com/moasq/go-b2b-starter/internal/db/helpers"
	sqlc "github.com/moasq/go-b2b-starter/internal/db/postgres/sqlc/gen"
)

func nowPtr() *time.Time {
	now := time.Now()
	return &now
}

// Cross-tenant inserts/updates must fail with FK violations (23503) at the
// database level, without any application check.

func TestCrossTenantAssignedToRejected(t *testing.T) {
	ctx := context.Background()
	q := testStore
	orgA, accA := createOrgWithAccount(t, ctx, q)
	_, accB := createOrgWithAccount(t, ctx, q)

	contact, err := q.UpsertContact(ctx, sqlc.UpsertContactParams{
		OrganizationID: orgA,
		PhoneNumber:    "+573001111001",
		DisplayName:    helpers.ToPgText("A"),
		LastMessageAt:  helpers.ToPgTimestampPtr(nil),
	})
	if err != nil {
		t.Fatalf("seed contact: %v", err)
	}

	_, err = q.UpdateContact(ctx, sqlc.UpdateContactParams{
		ID:             contact.ID,
		OrganizationID: orgA,
		AssignedTo:     helpers.ToPgInt4Ptr(&accB), // account belongs to another org
	})
	if !isPgError(err, "23503") {
		t.Fatalf("expected FK violation for cross-tenant assigned_to, got: %v", err)
	}
	_ = accA
}

func TestCrossTenantOwnerAccountRejected(t *testing.T) {
	ctx := context.Background()
	q := testStore
	orgA, _ := createOrgWithAccount(t, ctx, q)
	_, accB := createOrgWithAccount(t, ctx, q)

	_, err := q.CreateCompany(ctx, sqlc.CreateCompanyParams{
		OrganizationID: orgA,
		Name:           "Cross Owner Co",
		OwnerAccountID: helpers.ToPgInt4Ptr(&accB),
	})
	if !isPgError(err, "23503") {
		t.Fatalf("expected FK violation for cross-tenant owner_account_id, got: %v", err)
	}
}

func TestCrossTenantDealAssignedToRejected(t *testing.T) {
	ctx := context.Background()
	q := testStore
	orgA, _ := createOrgWithAccount(t, ctx, q)
	_, accB := createOrgWithAccount(t, ctx, q)

	pipeline, err := q.CreatePipeline(ctx, sqlc.CreatePipelineParams{
		OrganizationID: orgA,
		Nombre:         "P Cross",
		EsPredeterminado: false,
		Orden:           0,
	})
	if err != nil {
		t.Fatalf("seed pipeline: %v", err)
	}
	stage, err := q.CreatePipelineStage(ctx, sqlc.CreatePipelineStageParams{
		PipelineID: pipeline.ID,
		Nombre:     "Stage 1",
		Orden:      1,
	})
	if err != nil {
		t.Fatalf("seed stage: %v", err)
	}

	_, err = q.CreateDeal(ctx, sqlc.CreateDealParams{
		OrganizationID: orgA,
		Nombre:         "Deal Cross",
		PipelineID:     pipeline.ID,
		StageID:        helpers.ToPgInt4Ptr(&stage.ID),
		Estado:         "abierto",
		AssignedTo:     helpers.ToPgInt4Ptr(&accB),
	})
	if !isPgError(err, "23503") {
		t.Fatalf("expected FK violation for cross-tenant deal assigned_to, got: %v", err)
	}
}

func TestCrossTenantRealizadaPorRejected(t *testing.T) {
	ctx := context.Background()
	q := testStore
	orgA, _ := createOrgWithAccount(t, ctx, q)
	_, accB := createOrgWithAccount(t, ctx, q)

	contact, err := q.UpsertContact(ctx, sqlc.UpsertContactParams{
		OrganizationID: orgA,
		PhoneNumber:    "+573001111002",
		DisplayName:    helpers.ToPgText("B"),
		LastMessageAt:  helpers.ToPgTimestampPtr(nil),
	})
	if err != nil {
		t.Fatalf("seed contact: %v", err)
	}

	_, err = q.CreateActivity(ctx, sqlc.CreateActivityParams{
		OrganizationID: orgA,
		ContactID:      helpers.ToPgInt4Ptr(&contact.ID),
		Tipo:           "nota",
		Asunto:         helpers.ToPgText("x"),
		RealizadaPor:   helpers.ToPgInt4Ptr(&accB),
		RealizadaEn:    helpers.ToPgTimestamptzPtr(nowPtr()),
	})
	if !isPgError(err, "23503") {
		t.Fatalf("expected FK violation for cross-tenant realizada_por, got: %v", err)
	}
}

func TestCrossTenantParentLinksRejected(t *testing.T) {
	ctx := context.Background()
	q := testStore
	orgA, _ := createOrgWithAccount(t, ctx, q)
	orgB, _ := createOrgWithAccount(t, ctx, q)

	contactB, err := q.UpsertContact(ctx, sqlc.UpsertContactParams{
		OrganizationID: orgB,
		PhoneNumber:    "+573001111003",
		DisplayName:    helpers.ToPgText("C"),
		LastMessageAt:  helpers.ToPgTimestampPtr(nil),
	})
	if err != nil {
		t.Fatalf("seed contact B: %v", err)
	}
	companyB, err := q.CreateCompany(ctx, sqlc.CreateCompanyParams{
		OrganizationID: orgB,
		Name:           "Co B",
	})
	if err != nil {
		t.Fatalf("seed company B: %v", err)
	}
	convB, err := q.CreateConversation(ctx, sqlc.CreateConversationParams{
		OrganizationID: orgB,
		ContactID:      contactB.ID,
		Status:         "active",
	})
	if err != nil {
		t.Fatalf("seed conversation B: %v", err)
	}

	// contact in org A linked to company of org B
	contactA, err := q.UpsertContact(ctx, sqlc.UpsertContactParams{
		OrganizationID: orgA,
		PhoneNumber:    "+573001111004",
		DisplayName:    helpers.ToPgText("D"),
		LastMessageAt:  helpers.ToPgTimestampPtr(nil),
	})
	if err != nil {
		t.Fatalf("seed contact A: %v", err)
	}
	_, err = q.UpdateContact(ctx, sqlc.UpdateContactParams{
		ID:             contactA.ID,
		OrganizationID: orgA,
		CompanyID:      helpers.ToPgInt4Ptr(&companyB.ID),
	})
	if !isPgError(err, "23503") {
		t.Fatalf("expected FK violation for cross-tenant contacts.company_id, got: %v", err)
	}

	// conversation in org A for contact of org B
	_, err = q.CreateConversation(ctx, sqlc.CreateConversationParams{
		OrganizationID: orgA,
		ContactID:      contactB.ID,
		Status:         "active",
	})
	if !isPgError(err, "23503") {
		t.Fatalf("expected FK violation for cross-tenant conversations.contact_id, got: %v", err)
	}

	// message in org A in conversation of org B
	_, err = q.CreateMessage(ctx, sqlc.CreateMessageParams{
		OrganizationID:    orgA,
		ConversationID:    convB.ID,
		ContactID:         contactB.ID,
		WhatsappMessageID: helpers.ToPgText("wamid-cross-conv"),
		Direction:         "inbound",
		MessageType:       "text",
		Content:           helpers.ToPgText("hi"),
		Status:            "received",
	})
	if !isPgError(err, "23503") {
		t.Fatalf("expected FK violation for cross-tenant messages.conversation_id, got: %v", err)
	}
}
