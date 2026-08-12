package tickets

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/moasq/go-b2b-starter/internal/modules/auth"
	"github.com/moasq/go-b2b-starter/internal/platform/authcontext"
	billingServices "github.com/moasq/go-b2b-starter/internal/modules/billing/app/services"
	billingDomain "github.com/moasq/go-b2b-starter/internal/modules/billing/domain"
	ticketsServices "github.com/moasq/go-b2b-starter/internal/modules/tickets/app/services"
	"github.com/moasq/go-b2b-starter/internal/modules/tickets/domain"
	llmdomain "github.com/moasq/go-b2b-starter/internal/platform/llm/domain"
)

// ---------- mocks ----------

type handlerMockLLM struct {
	llmdomain.LLMClient
	text string
	err  error
}

func (m *handlerMockLLM) Complete(ctx context.Context, req llmdomain.CompletionRequest) (*llmdomain.CompletionResponse, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &llmdomain.CompletionResponse{Text: m.text, TokensUsed: 5, Model: "test-model"}, nil
}

type handlerMockBilling struct {
	billingServices.BillingService
	status *billingDomain.AiUsageStatus
	err    error
}

func (m *handlerMockBilling) GetAiUsageStatus(ctx context.Context, orgID int32) (*billingDomain.AiUsageStatus, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.status, nil
}

type handlerMockRepo struct {
	domain.TicketRepository
	tickets map[int32]*domain.Ticket
}

func (m *handlerMockRepo) GetByID(ctx context.Context, orgID, ticketID int32) (*domain.Ticket, error) {
	t, ok := m.tickets[ticketID]
	if !ok || t.OrganizationID != orgID {
		return nil, domain.ErrTicketNotFound
	}
	return t, nil
}

// ---------- harness ----------

func newTriageHandler(llm llmdomain.LLMClient, billing billingServices.BillingService) *Handler {
	repo := &handlerMockRepo{tickets: map[int32]*domain.Ticket{
		1: {ID: 1, OrganizationID: 1, Title: "Problema con factura", Description: "No llegó la factura"},
		2: {ID: 2, OrganizationID: 2, Title: "Ajeno", Description: "otra org"},
	}}
	return NewHandler(nil, ticketsServices.NewAITriageService(llm, billing, repo))
}

func newTriageTestRouter(h *Handler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	api := r.Group("/api")
	api.Use(func(c *gin.Context) {
		authcontext.SetIdentity(c, &authcontext.Identity{
			UserID: "member-1",
			Permissions: []authcontext.Permission{
				authcontext.NewPermission("ticket", "view"),
			},
		})
		authcontext.SetRequestContext(c, &authcontext.RequestContext{OrganizationID: 1})
	})
	api.POST("/tickets/:id/ai-triage", auth.RequirePermissionFunc("ticket", "view"), h.AiTriage)
	return r
}

func doTriageRequest(router *gin.Engine, id string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/tickets/"+id+"/ai-triage", nil)
	router.ServeHTTP(w, req)
	return w
}

// ---------- tests ----------

func TestAiTriageHandler_UnauthenticatedReturns401(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/api/tickets/:id/ai-triage", auth.RequirePermissionFunc("ticket", "view"), newTriageHandler(
		&handlerMockLLM{text: `{"note":"n","priority":"alta"}`},
		&handlerMockBilling{status: &billingDomain.AiUsageStatus{CreditsMax: 100, CreditsRemaining: 50}},
	).AiTriage)

	w := doTriageRequest(r, "1")
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without identity, got %d (body: %s)", w.Code, w.Body.String())
	}
}

func TestAiTriageHandler_MissingOrForeignTicketReturns404(t *testing.T) {
	router := newTriageTestRouter(newTriageHandler(
		&handlerMockLLM{text: `{"note":"n","priority":"alta"}`},
		&handlerMockBilling{status: &billingDomain.AiUsageStatus{CreditsMax: 100, CreditsRemaining: 50}},
	))

	// Missing ticket id.
	w := doTriageRequest(router, "999")
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for missing ticket, got %d (body: %s)", w.Code, w.Body.String())
	}

	// Foreign-org ticket id (ticket 2 belongs to org 2; caller is org 1).
	w = doTriageRequest(router, "2")
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for foreign-org ticket, got %d (body: %s)", w.Code, w.Body.String())
	}
}

func TestAiTriageHandler_InvalidIDReturns400(t *testing.T) {
	router := newTriageTestRouter(newTriageHandler(
		&handlerMockLLM{text: `{"note":"n","priority":"alta"}`},
		&handlerMockBilling{status: &billingDomain.AiUsageStatus{CreditsMax: 100, CreditsRemaining: 50}},
	))

	w := doTriageRequest(router, "abc")
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid id, got %d (body: %s)", w.Code, w.Body.String())
	}
}

func TestAiTriageHandler_CreditsExhaustedReturns402(t *testing.T) {
	router := newTriageTestRouter(newTriageHandler(
		&handlerMockLLM{text: `{"note":"no debe usarse","priority":"alta"}`},
		&handlerMockBilling{status: &billingDomain.AiUsageStatus{CreditsMax: 100, CreditsRemaining: 0}},
	))

	w := doTriageRequest(router, "1")
	if w.Code != http.StatusPaymentRequired {
		t.Fatalf("expected 402, got %d (body: %s)", w.Code, w.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("unparseable body: %v", err)
	}
	if body["code"] != "ai_credits_exhausted" {
		t.Fatalf("expected machine-readable ai_credits_exhausted code, got %v", body["code"])
	}
}

func TestAiTriageHandler_HappyPathReturns200Envelope(t *testing.T) {
	router := newTriageTestRouter(newTriageHandler(
		&handlerMockLLM{text: `{"note":"El cliente no recibió su factura.","priority":"alta"}`},
		&handlerMockBilling{status: &billingDomain.AiUsageStatus{CreditsMax: 100, CreditsRemaining: 50}},
	))

	w := doTriageRequest(router, "1")
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", w.Code, w.Body.String())
	}
	var body struct {
		Success bool `json:"success"`
		Data    struct {
			Note     string  `json:"note"`
			Priority *string `json:"priority"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("unparseable body: %v", err)
	}
	if !body.Success {
		t.Fatal("expected success: true")
	}
	if body.Data.Note != "El cliente no recibió su factura." {
		t.Fatalf("unexpected note: %q", body.Data.Note)
	}
	if body.Data.Priority == nil || *body.Data.Priority != "high" {
		t.Fatalf("expected priority high, got %v", body.Data.Priority)
	}
}

func TestAiTriageHandler_InvalidPriorityIsDroppedInEnvelope(t *testing.T) {
	router := newTriageTestRouter(newTriageHandler(
		&handlerMockLLM{text: `{"note":"Solo nota.","priority":"urgente"}`},
		&handlerMockBilling{status: &billingDomain.AiUsageStatus{CreditsMax: 100, CreditsRemaining: 50}},
	))

	w := doTriageRequest(router, "1")
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"priority":null`) {
		t.Fatalf("expected priority null in body: %s", w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "Solo nota.") {
		t.Fatalf("expected note preserved in body: %s", w.Body.String())
	}
}

func TestAiTriageHandler_LLMFailureReturns500(t *testing.T) {
	router := newTriageTestRouter(newTriageHandler(
		&handlerMockLLM{err: errors.New("provider down")},
		&handlerMockBilling{status: &billingDomain.AiUsageStatus{CreditsMax: 100, CreditsRemaining: 50}},
	))

	w := doTriageRequest(router, "1")
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d (body: %s)", w.Code, w.Body.String())
	}
}
