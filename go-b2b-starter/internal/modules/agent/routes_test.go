package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/moasq/go-b2b-starter/internal/modules/agent/app/services"
	"github.com/moasq/go-b2b-starter/internal/platform/authcontext"
)

// stubAgentService implements AgentService for route tests by embedding the
// interface and overriding only RephraseText.
type stubAgentService struct {
	services.AgentService
	rephraseText string
	rephraseErr  error
}

func (s *stubAgentService) RephraseText(ctx context.Context, orgID int32, text, mode string) (string, error) {
	if s.rephraseErr != nil {
		return "", s.rephraseErr
	}
	return s.rephraseText, nil
}

// stubResolver provides the named middleware the agent routes require.
// authed=false simulates an unauthenticated request (401 before handlers run).
type stubResolver struct {
	authed bool
}

func (r *stubResolver) Get(name string) gin.HandlerFunc {
	switch name {
	case "auth":
		return func(c *gin.Context) {
			if !r.authed {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"code": "unauthorized", "message": "authentication required"})
				return
			}
			authcontext.SetIdentity(c, &authcontext.Identity{
				UserID:      "stytch_user_1",
				Roles:       []authcontext.Role{authcontext.RoleAdmin},
				Permissions: []authcontext.Permission{authcontext.NewPermission("org", "view"), authcontext.NewPermission("org", "manage")},
			})
			c.Next()
		}
	case "org_context":
		return func(c *gin.Context) {
			authcontext.SetRequestContext(c, &authcontext.RequestContext{
				Identity:       authcontext.GetIdentity(c),
				OrganizationID: 42,
				AccountID:      7,
				ProviderOrgID:  "org-uuid",
			})
			c.Next()
		}
	case "subscription":
		return func(c *gin.Context) {
			c.Next()
		}
	default:
		return func(c *gin.Context) { c.Next() }
	}
}

func setupRephraseRouter(svc services.AgentService, authed bool) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	routes := NewRoutes(NewHandler(svc, nil, nil, nil))
	routes.RegisterRoutes(router.Group("/"), &stubResolver{authed: authed})
	return router
}

func postRephrase(router *gin.Engine, body string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/agent/rephrase", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)
	return rec
}

func TestHandleRephraseUnauthenticatedReturns401(t *testing.T) {
	router := setupRephraseRouter(&stubAgentService{rephraseText: "texto"}, false)

	rec := postRephrase(router, `{"text":"hola","mode":"rephrase"}`)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestHandleRephraseSuccessEnvelope(t *testing.T) {
	router := setupRephraseRouter(&stubAgentService{rephraseText: "Hola, ¿cuándo llega mi pedido?"}, true)

	rec := postRephrase(router, `{"text":"hola cuando llega mi pedido","mode":"formal"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	var envelope struct {
		Data struct {
			Text string `json:"text"`
		} `json:"data"`
		Success bool `json:"success"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("unmarshal envelope: %v", err)
	}
	if !envelope.Success || envelope.Data.Text != "Hola, ¿cuándo llega mi pedido?" {
		t.Fatalf("unexpected envelope: %+v", envelope)
	}
}

func TestHandleRephraseInvalidModeReturns400(t *testing.T) {
	router := setupRephraseRouter(&stubAgentService{rephraseText: "x"}, true)

	rec := postRephrase(router, `{"text":"hola","mode":"poetic"}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", rec.Code, rec.Body.String())
	}
	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal body: %v", err)
	}
	if body["code"] != "invalid_mode" {
		t.Fatalf("expected invalid_mode code, got %v", body)
	}
}

func TestHandleRephraseCreditsExhaustedReturns402(t *testing.T) {
	router := setupRephraseRouter(&stubAgentService{rephraseErr: services.ErrAICreditsExhausted}, true)

	rec := postRephrase(router, `{"text":"hola","mode":"rephrase"}`)
	if rec.Code != http.StatusPaymentRequired {
		t.Fatalf("expected 402, got %d body=%s", rec.Code, rec.Body.String())
	}
	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal body: %v", err)
	}
	if body["code"] != "ai_credits_exhausted" {
		t.Fatalf("expected ai_credits_exhausted code, got %v", body)
	}
}
