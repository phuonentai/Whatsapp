package services

import (
	"context"
	"testing"
	"time"

	"github.com/moasq/go-b2b-starter/internal/modules/procurement/domain"
)

func TestBoardDeterministicRanking(t *testing.T) {
	rows := []domain.BoardRow{
		{RecipientID: 1, DisplayName: "Sin respuesta"},
		{RecipientID: 2, DisplayName: "Caro y lento", Response: &domain.InquiryResponse{Items: []domain.ResponseItem{
			{ProductName: "A", Disponible: true, PrecioUnitario: f64(15000), TiempoEntrega: sPtr("3 días")},
		}}},
		{RecipientID: 3, DisplayName: "Barato y rápido", Response: &domain.InquiryResponse{Items: []domain.ResponseItem{
			{ProductName: "A", Disponible: true, PrecioUnitario: f64(10000), TiempoEntrega: sPtr("1 día")},
		}}},
		{RecipientID: 4, DisplayName: "Sin precio", Response: &domain.InquiryResponse{Items: []domain.ResponseItem{
			{ProductName: "A", Disponible: true},
		}}},
	}
	rankRows(rows)

	// availability desc first (all available), then unit price asc, then lead
	// time asc; no-response rows sort after available ones.
	if rows[0].RecipientID != 3 {
		t.Fatalf("expected cheapest+fastest first, got %d", rows[0].RecipientID)
	}
	if rows[1].RecipientID != 2 {
		t.Fatalf("expected second cheapest first-availability, got %d", rows[1].RecipientID)
	}
	// deterministic tie-break: no-price row must come before no-response row
	if rows[2].RecipientID != 4 || rows[3].RecipientID != 1 {
		t.Fatalf("expected [4,1] tail, got [%d,%d]", rows[2].RecipientID, rows[3].RecipientID)
	}

	// stability: re-rank an equal state produces the same order
	before := rows[2].RecipientID
	rankRows(rows)
	if rows[2].RecipientID != before {
		t.Fatalf("ranking must be deterministic across refreshes")
	}
}

func TestBoardLazyTimeoutReconcilesAndCompletes(t *testing.T) {
	ctx := context.Background()
	runs, _, audit, llm, _, metrics := newTestDeps()
	llm.addResponse(`{"message":"hola"}`)
	runs.seedSupplier(1, 101, "900111", "ProvA")
	runs.seedSupplier(2, 102, "900222", "ProvB")

	svc := NewRunService(runs, nil, newProductsRepo(), audit, NewDraftingService(llm, &mockBilling{}, audit, metrics, stubLogger{}), metrics, stubLogger{})
	board := NewBoardService(runs, audit, llm, &mockBilling{}, metrics, stubLogger{})

	run, err := svc.CreateRun(ctx, 42, "m", CreateRunInput{SupplierIDs: []int32{1, 2}, Products: []RunProduct{{ProductID: 10, Quantity: 1}}})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if _, err := svc.SendRun(ctx, 42, run.ID); err != nil {
		t.Fatalf("send: %v", err)
	}
	// all recipients sent; run awaiting responses (mock afterSend is not
	// triggered — the mock does not run afterSend, so move manually)
	for _, r := range runs.recipients {
		if r.RunID == run.ID {
			r.Status = domain.RecipientSent
			now := time.Now().Add(-48 * time.Hour)
			r.SentAt = &now
		}
	}
	runs.runs[run.ID].Status = domain.RunAwaitingResponses

	b, err := board.GetBoard(ctx, 42, run.ID)
	if err != nil {
		t.Fatalf("board: %v", err)
	}
	// both expired → timed_out → partially_answered (answered=0, terminal=2)
	if b.Run.Status != domain.RunPartiallyAnswered {
		t.Fatalf("expected partially_answered after lazy timeout, got %s", b.Run.Status)
	}
	for _, r := range runs.recipients {
		if r.RunID == run.ID && r.Status != domain.RecipientTimedOut {
			t.Fatalf("expected timed_out, got %s", r.Status)
		}
	}
}

func TestBoardSummarySkippedWhenCreditsExhausted(t *testing.T) {
	ctx := context.Background()
	runs, _, audit, llm, _, metrics := newTestDeps()
	runs.seedSupplier(1, 101, "900111", "ProvA")

	// a run with an answered response
	run, _ := runs.CreateRun(ctx, 42, strPtr(""), "m")
	rec, _ := runs.CreateRecipient(ctx, 42, run.ID, 1, 101, strPtr("draft"))
	rec.Status = domain.RecipientAnswered
	resp := &domain.InquiryResponse{OrganizationID: 42, RecipientID: rec.ID, RawMessageID: "m1", Items: []domain.ResponseItem{{ProductName: "A", Disponible: true}}, Resumen: "Disponible"}
	_, _ = runs.SaveResponse(ctx, resp)
	run.Status = domain.RunAwaitingResponses

	board := NewBoardService(runs, audit, llm, &mockBilling{exhausted: true}, metrics, stubLogger{})
	b, err := board.GetBoard(ctx, 42, run.ID)
	if err != nil {
		t.Fatalf("board: %v", err)
	}
	if b.Summary != nil {
		t.Fatalf("expected no summary when credits exhausted")
	}
	if llm.calls != 0 {
		t.Fatalf("expected no LLM call when credits exhausted")
	}
}

func f64(v float64) *float64 { return &v }
func sPtr(s string) *string  { return &s }
