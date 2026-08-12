package services

import (
	"context"
	"errors"
	"fmt"
	"sort"

	"github.com/moasq/go-b2b-starter/internal/modules/procurement/domain"
	billingServices "github.com/moasq/go-b2b-starter/internal/modules/billing/app/services"
	llmdomain "github.com/moasq/go-b2b-starter/internal/platform/llm/domain"
	logger "github.com/moasq/go-b2b-starter/internal/platform/logger/domain"
)

// ResponseWindowHours is the fixed lazy timeout window (D12 default).
const ResponseWindowHours = 24

// boardService implements the aggregation board: deterministic ranking (D5),
// the optional metered summary, and the lazy read-time timeout
// reconciliation (D12).
type boardService struct {
	runs    domain.InquiryRunRepository
	audit   domain.AuditRepository
	llm     llmdomain.LLMClient
	billing billingServices.BillingService
	metrics MetricsSink
	log     logger.Logger
}

// NewBoardService builds the board service.
func NewBoardService(
	runs domain.InquiryRunRepository,
	audit domain.AuditRepository,
	llm llmdomain.LLMClient,
	billing billingServices.BillingService,
	metrics MetricsSink,
	log logger.Logger,
) *boardService {
	if metrics == nil {
		metrics = noopMetrics{}
	}
	return &boardService{runs: runs, audit: audit, llm: llm, billing: billing, metrics: metrics, log: log}
}

// GetBoard reconciles expired recipients lazily (D12), loads the board feed,
// ranks deterministically (availability desc, unit price asc, lead time asc),
// and attaches an optional metered summary when responses exist and credits
// allow.
func (s *boardService) GetBoard(ctx context.Context, orgID, runID int32) (*domain.Board, error) {
	run, err := s.runs.GetRun(ctx, orgID, runID)
	if err != nil {
		return nil, err
	}

	if run.Status == domain.RunAwaitingResponses {
		if err := s.reconcileTimeouts(ctx, orgID, run); err != nil {
			return nil, err
		}
		run, err = s.runs.GetRun(ctx, orgID, runID)
		if err != nil {
			return nil, err
		}
	}

	rows, err := s.runs.RunBoardRows(ctx, orgID, runID)
	if err != nil {
		return nil, err
	}

	rankRows(rows)

	board := &domain.Board{Run: run, Rows: rows}

	hasResponses := false
	for _, r := range rows {
		if r.Response != nil {
			hasResponses = true
			break
		}
	}
	if hasResponses {
		summary, err := s.summarize(ctx, orgID, rows)
		if err == nil && summary != "" {
			board.Summary = &summary
		}
		// Summary is best-effort: credit exhaustion / LLM errors return the
		// board without the summary and never fail the read.
	}
	return board, nil
}

// reconcileTimeouts transitions sent recipients older than the response
// window to timed_out and re-evaluates the run's terminal transition
// (idempotent, transaction-isolated).
func (s *boardService) reconcileTimeouts(ctx context.Context, orgID int32, run *domain.InquiryRun) error {
	expired, err := s.runs.ListExpiredSentRecipients(ctx, orgID, run.ID, ResponseWindowHours)
	if err != nil {
		return err
	}
	for _, r := range expired {
		if _, err := s.runs.MarkRecipientTimedOut(ctx, orgID, r.ID); err != nil && !errors.Is(err, domain.ErrRecipientNotFound) {
			return err
		}
	}
	if len(expired) > 0 {
		_ = s.evaluateRunCompletion(ctx, orgID, run.ID)
	}
	return nil
}

// evaluateRunCompletion transitions an awaiting_responses run to completed or
// partially_answered based on recipient states (guarded, idempotent).
func (s *boardService) evaluateRunCompletion(ctx context.Context, orgID, runID int32) error {
	recipients, err := s.runs.ListRunRecipients(ctx, orgID, runID)
	if err != nil {
		return err
	}
	total := len(recipients)
	if total == 0 {
		return nil
	}
	answered := 0
	terminal := 0
	for _, r := range recipients {
		switch r.Status {
		case domain.RecipientAnswered:
			answered++
		case domain.RecipientTimedOut, domain.RecipientFailed:
			terminal++
		}
	}
	switch {
	case answered == total:
		_, err = s.runs.TransitionRun(ctx, orgID, runID, domain.RunAwaitingResponses, domain.RunCompleted)
	case answered+terminal == total && answered > 0:
		_, err = s.runs.TransitionRun(ctx, orgID, runID, domain.RunAwaitingResponses, domain.RunPartiallyAnswered)
	case terminal == total:
		_, err = s.runs.TransitionRun(ctx, orgID, runID, domain.RunAwaitingResponses, domain.RunPartiallyAnswered)
	default:
		err = nil // still awaiting
	}
	if err != nil && errors.Is(err, domain.ErrInvalidTransition) {
		return nil // already moved concurrently
	}
	return err
}

// rankRows sorts deterministically: availability desc, unit price asc, lead
// time asc (D5). Suppliers without a response or without a price/lead time
// sort after the ones that have them; ties break by recipient id.
func rankRows(rows []domain.BoardRow) {
	sort.SliceStable(rows, func(i, j int) bool {
		ai, aj := availabilityScore(rows[i]), availabilityScore(rows[j])
		if ai != aj {
			return ai > aj
		}
		pi, hasPi := minUnitPrice(rows[i])
		pj, hasPj := minUnitPrice(rows[j])
		if hasPi != hasPj {
			return hasPi // priced rows before unpriced
		}
		if hasPi && pi != pj {
			return pi < pj
		}
		li, hasLi := leadTime(rows[i])
		lj, hasLj := leadTime(rows[j])
		if hasLi != hasLj {
			return hasLi
		}
		if hasLi && li != lj {
			return li < lj
		}
		return rows[i].RecipientID < rows[j].RecipientID
	})
}

func availabilityScore(r domain.BoardRow) int {
	if r.Response == nil {
		return 0
	}
	score := 0
	for _, it := range r.Response.Items {
		if it.Disponible {
			score++
		}
	}
	return score
}

func minUnitPrice(r domain.BoardRow) (float64, bool) {
	if r.Response == nil {
		return 0, false
	}
	min := 0.0
	found := false
	for _, it := range r.Response.Items {
		if it.PrecioUnitario != nil {
			if !found || *it.PrecioUnitario < min {
				min = *it.PrecioUnitario
				found = true
			}
		}
	}
	return min, found
}

func leadTime(r domain.BoardRow) (int, bool) {
	if r.Response == nil {
		return 0, false
	}
	min := 0
	found := false
	for _, it := range r.Response.Items {
		if it.TiempoEntrega != nil && *it.TiempoEntrega != "" {
			n := extractLeadDays(*it.TiempoEntrega)
			if !found || n < min {
				min = n
				found = true
			}
		}
	}
	return min, found
}

// extractLeadDays parses "2-3 días" / "1 semana" style lead times; unknown
// formats score 0 (sorted first among equals is fine; ties break by id).
func extractLeadDays(text string) int {
	digits := 0
	for _, ch := range text {
		if ch >= '0' && ch <= '9' {
			digits = digits*10 + int(ch-'0')
		}
	}
	return digits
}

// summarize runs one optional metered LLM call over the answered rows. Credits
// exhausted or LLM errors yield "" (board returned without summary).
func (s *boardService) summarize(ctx context.Context, orgID int32, rows []domain.BoardRow) (string, error) {
	s.metrics.Inc(MetricSummaryAttempt, map[string]string{"org": itoa(orgID)})
	if creditsExhausted(ctx, s.billing, orgID) {
		return "", domain.ErrCreditsExhausted
	}

	prompt := "Resume en 3-4 líneas en español el estado de esta cotización comparativa de proveedores:\n"
	for _, r := range rows {
		line := fmt.Sprintf("- %s (estado %s)", r.DisplayName, r.RecipientStatus)
		if r.Response != nil {
			line += " → " + r.Response.Resumen
		}
		prompt += line + "\n"
	}
	prompt += "No menciones precios como confirmados; si hay ambigüedad indícalo."

	resp, err := s.llm.Complete(llmdomain.WithOrgID(ctx, orgID), llmdomain.CompletionRequest{Prompt: prompt})
	if err != nil {
		s.log.Warn("board summary failed", map[string]any{"run_org": orgID, "error": err.Error()})
		return "", err
	}
	return resp.Text, nil
}
