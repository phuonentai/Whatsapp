//go:build integration

package integration

import (
	"context"
	"testing"

	"github.com/moasq/go-b2b-starter/internal/db/helpers"
	sqlc "github.com/moasq/go-b2b-starter/internal/db/postgres/sqlc/gen"
)

// Stage/pipeline consistency: a deal cannot hold a stage from another pipeline;
// stage_id updates normalize pipeline_id via the trigger; deleting a stage nulls
// deals.stage_id while preserving pipeline_id.

func TestCrossPipelineStageRejected(t *testing.T) {
	ctx := context.Background()
	q := testStore
	orgA, _ := createOrgWithAccount(t, ctx, q)

	p1, err := q.CreatePipeline(ctx, sqlc.CreatePipelineParams{
		OrganizationID:   orgA,
		Nombre:           "P1",
		EsPredeterminado: false,
		Orden:            0,
	})
	if err != nil {
		t.Fatalf("seed pipeline 1: %v", err)
	}
	p2, err := q.CreatePipeline(ctx, sqlc.CreatePipelineParams{
		OrganizationID:   orgA,
		Nombre:           "P2",
		EsPredeterminado: false,
		Orden:            1,
	})
	if err != nil {
		t.Fatalf("seed pipeline 2: %v", err)
	}
	stageP1, err := q.CreatePipelineStage(ctx, sqlc.CreatePipelineStageParams{
		PipelineID: p1.ID,
		Nombre:     "S P1",
		Orden:      1,
	})
	if err != nil {
		t.Fatalf("seed stage P1: %v", err)
	}
	stageP2, err := q.CreatePipelineStage(ctx, sqlc.CreatePipelineStageParams{
		PipelineID: p2.ID,
		Nombre:     "S P2",
		Orden:      1,
	})
	if err != nil {
		t.Fatalf("seed stage P2: %v", err)
	}

	// Insert with mismatched pipeline_id/stage_id: the trigger normalizes
	// pipeline_id from stage_id, so this must succeed with pipeline_id = p1.
	deal, err := q.CreateDeal(ctx, sqlc.CreateDealParams{
		OrganizationID: orgA,
		Nombre:         "D Normalize",
		PipelineID:     p2.ID, // intentionally wrong; stage belongs to p1
		StageID:        helpers.ToPgInt4Ptr(&stageP1.ID),
		Estado:         "abierto",
	})
	if err != nil {
		t.Fatalf("create deal (should normalize): %v", err)
	}
	if deal.PipelineID != p1.ID {
		t.Fatalf("expected pipeline_id normalized to %d, got %d", p1.ID, deal.PipelineID)
	}

	// Update stage_id to a stage of another pipeline must normalize too.
	updated, err := q.UpdateDealStage(ctx, sqlc.UpdateDealStageParams{
		ID:             deal.ID,
		OrganizationID: orgA,
		StageID:        helpers.ToPgInt4Ptr(&stageP2.ID),
	})
	if err != nil {
		t.Fatalf("update deal stage: %v", err)
	}
	if updated.PipelineID != p2.ID {
		t.Fatalf("expected pipeline_id %d after stage update, got %d", p2.ID, updated.PipelineID)
	}
	if !updated.StageID.Valid || updated.StageID.Int32 != stageP2.ID {
		t.Fatalf("expected stage_id %d, got %+v", stageP2.ID, updated.StageID)
	}
}

func TestUnknownStageRejected(t *testing.T) {
	ctx := context.Background()
	q := testStore
	orgA, _ := createOrgWithAccount(t, ctx, q)

	p1, err := q.CreatePipeline(ctx, sqlc.CreatePipelineParams{
		OrganizationID:   orgA,
		Nombre:           "P Unknown",
		EsPredeterminado: false,
		Orden:            0,
	})
	if err != nil {
		t.Fatalf("seed pipeline: %v", err)
	}

	_, err = q.CreateDeal(ctx, sqlc.CreateDealParams{
		OrganizationID: orgA,
		Nombre:         "D Bad",
		PipelineID:     p1.ID,
		StageID:        helpers.ToPgInt4Ptr(int32Ptr(99999)),
		Estado:         "abierto",
	})
	if err == nil {
		t.Fatal("expected error for unknown stage, got nil")
	}
}

func int32Ptr(v int32) *int32 { return &v }

func TestStageDeletionNullsStagePreservesPipeline(t *testing.T) {
	ctx := context.Background()
	q := testStore
	orgA, _ := createOrgWithAccount(t, ctx, q)

	p1, err := q.CreatePipeline(ctx, sqlc.CreatePipelineParams{
		OrganizationID:   orgA,
		Nombre:           "P Del Stage",
		EsPredeterminado: false,
		Orden:            0,
	})
	if err != nil {
		t.Fatalf("seed pipeline: %v", err)
	}
	stage, err := q.CreatePipelineStage(ctx, sqlc.CreatePipelineStageParams{
		PipelineID: p1.ID,
		Nombre:     "S1",
		Orden:      1,
	})
	if err != nil {
		t.Fatalf("seed stage: %v", err)
	}
	deal, err := q.CreateDeal(ctx, sqlc.CreateDealParams{
		OrganizationID: orgA,
		Nombre:         "D1",
		PipelineID:     p1.ID,
		StageID:        helpers.ToPgInt4Ptr(&stage.ID),
		Estado:         "abierto",
	})
	if err != nil {
		t.Fatalf("seed deal: %v", err)
	}

	if err := q.DeletePipelineStage(ctx, sqlc.DeletePipelineStageParams{ID: stage.ID, PipelineID: p1.ID}); err != nil {
		t.Fatalf("delete stage: %v", err)
	}

	got, err := q.GetDealByID(ctx, sqlc.GetDealByIDParams{ID: deal.ID, OrganizationID: orgA})
	if err != nil {
		t.Fatalf("get deal: %v", err)
	}
	if got.StageID.Valid {
		t.Fatalf("expected deals.stage_id NULL after stage deletion, got %+v", got.StageID)
	}
	if got.PipelineID != p1.ID {
		t.Fatalf("expected pipeline_id preserved as %d, got %d", p1.ID, got.PipelineID)
	}
}

func intPtr(v int32) *int32 { return &v }
