package crm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/moasq/go-b2b-starter/internal/modules/crm/domain"
	whatsappDomain "github.com/moasq/go-b2b-starter/internal/modules/whatsapp/domain"
	"github.com/moasq/go-b2b-starter/internal/platform/authcontext"
)

func errFrom(msg string) error { return errors.New(msg) }

// fakeOutboundService records template-send calls and returns scripted errors.
type fakeOutboundService struct {
	sendTemplateFn func(ctx context.Context, orgID, convID int32, templateID int64, params []string) (*domain.Message, error)
}

func (f *fakeOutboundService) SendMessage(ctx context.Context, orgID, convID int32, content string) (*domain.Message, error) {
	return nil, nil
}

func (f *fakeOutboundService) SendTemplateMessage(ctx context.Context, orgID, convID int32, templateID int64, params []string) (*domain.Message, error) {
	return f.sendTemplateFn(ctx, orgID, convID, templateID, params)
}

func performTemplateSend(t *testing.T, h *CRMHandler, orgID int32, body string) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/crm/conversaciones/:id/mensajes/template", func(c *gin.Context) {
		authcontext.SetRequestContext(c, &authcontext.RequestContext{
			Identity:       &authcontext.Identity{},
			OrganizationID: orgID,
			AccountID:      7,
		})
		c.Next()
	}, h.HandleSendTemplateMessage)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/crm/conversaciones/42/mensajes/template", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	return w
}

func TestHandleSendTemplateMessage_Success(t *testing.T) {
	msg := &domain.Message{
		OrganizationID:    42,
		ConversationID:    42,
		ProviderMessageID: "wamid.HBgNNTc",
		Direction:         domain.MessageDirectionOutbound,
		Status:            "sent",
	}
	h := &CRMHandler{outboundService: &fakeOutboundService{sendTemplateFn: func(ctx context.Context, orgID, convID int32, templateID int64, params []string) (*domain.Message, error) {
		if orgID != 42 || convID != 42 || templateID != 7 {
			t.Errorf("unexpected args: org=%d conv=%d template=%d", orgID, convID, templateID)
		}
		if len(params) != 2 || params[0] != "María" || params[1] != "Pedido #1234" {
			t.Errorf("unexpected params: %v", params)
		}
		return msg, nil
	}}}

	w := performTemplateSend(t, h, 42, `{"template_id": 7, "params": ["María", "Pedido #1234"]}`)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var env struct {
		Success bool            `json:"success"`
		Data    json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &env); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if !env.Success {
		t.Fatal("expected success envelope")
	}
	var got domain.Message
	if err := json.Unmarshal(env.Data, &got); err != nil {
		t.Fatalf("failed to decode message: %v", err)
	}
	if got.ProviderMessageID != "wamid.HBgNNTc" || got.Status != "sent" {
		t.Fatalf("unexpected message: %+v", got)
	}
}

func TestHandleSendTemplateMessage_TemplateNotFound(t *testing.T) {
	h := &CRMHandler{outboundService: &fakeOutboundService{sendTemplateFn: func(ctx context.Context, orgID, convID int32, templateID int64, params []string) (*domain.Message, error) {
		return nil, whatsappDomain.ErrTemplateNotFound
	}}}

	w := performTemplateSend(t, h, 42, `{"template_id": 999, "params": []}`)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
	var env struct {
		Code string `json:"code"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &env); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	if env.Code != "template_not_found" {
		t.Fatalf("expected template_not_found, got %s", env.Code)
	}
}

func TestHandleSendTemplateMessage_NotApproved(t *testing.T) {
	h := &CRMHandler{outboundService: &fakeOutboundService{sendTemplateFn: func(ctx context.Context, orgID, convID int32, templateID int64, params []string) (*domain.Message, error) {
		return nil, whatsappDomain.ErrTemplateNotApproved
	}}}

	w := performTemplateSend(t, h, 42, `{"template_id": 7, "params": []}`)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
	var env struct {
		Code string `json:"code"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &env)
	if env.Code != "template_not_approved" {
		t.Fatalf("expected template_not_approved, got %s", env.Code)
	}
}

func TestHandleSendTemplateMessage_ParamCountMismatch(t *testing.T) {
	h := &CRMHandler{outboundService: &fakeOutboundService{sendTemplateFn: func(ctx context.Context, orgID, convID int32, templateID int64, params []string) (*domain.Message, error) {
		return nil, whatsappDomain.ErrTemplateParamCountMismatch
	}}}

	w := performTemplateSend(t, h, 42, `{"template_id": 7, "params": ["solo"]}`)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
	var env struct {
		Code string `json:"code"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &env)
	if env.Code != "template_param_count_mismatch" {
		t.Fatalf("expected template_param_count_mismatch, got %s", env.Code)
	}
}

func TestHandleSendTemplateMessage_WhatsAppAPIError(t *testing.T) {
	h := &CRMHandler{outboundService: &fakeOutboundService{sendTemplateFn: func(ctx context.Context, orgID, convID int32, templateID int64, params []string) (*domain.Message, error) {
		return nil, errFrom("whatsapp_api_error: graph api error (code 100)")
	}}}

	w := performTemplateSend(t, h, 42, `{"template_id": 7, "params": ["a", "b"]}`)
	if w.Code != http.StatusBadGateway {
		t.Fatalf("expected 502, got %d: %s", w.Code, w.Body.String())
	}
}

func TestHandleSendTemplateMessage_RateLimit(t *testing.T) {
	h := &CRMHandler{outboundService: &fakeOutboundService{sendTemplateFn: func(ctx context.Context, orgID, convID int32, templateID int64, params []string) (*domain.Message, error) {
		return nil, errFrom("rate_limit: exceeded 10 messages per 10 seconds")
	}}}

	w := performTemplateSend(t, h, 42, `{"template_id": 7, "params": ["a", "b"]}`)
	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429, got %d: %s", w.Code, w.Body.String())
	}
	var env struct {
		Code string `json:"code"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &env)
	if env.Code != "rate_limit" {
		t.Fatalf("expected rate_limit, got %s", env.Code)
	}
}
