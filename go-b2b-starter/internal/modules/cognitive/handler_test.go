package cognitive

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/moasq/go-b2b-starter/internal/platform/authcontext"
	"github.com/moasq/go-b2b-starter/internal/modules/cognitive/app/services"
	"github.com/moasq/go-b2b-starter/internal/modules/cognitive/domain"
	"github.com/moasq/go-b2b-starter/internal/platform/features"
)

// mockRAGService embeds services.RAGService so only Chat/ChatStream are stubbed.
type mockRAGService struct {
	services.RAGService
	chatResponse *domain.ChatResponse
	chatErr      error
	streamEvents []domain.StreamEvent
	streamResp   *domain.ChatResponse
	streamErr    error
}

func (m *mockRAGService) Chat(ctx context.Context, orgID, accountID int32, req *domain.ChatRequest) (*domain.ChatResponse, error) {
	return m.chatResponse, m.chatErr
}

func (m *mockRAGService) ChatStream(ctx context.Context, orgID, accountID int32, req *domain.ChatRequest, emit func(domain.StreamEvent) error) (*domain.ChatResponse, error) {
	for _, ev := range m.streamEvents {
		if err := emit(ev); err != nil {
			return nil, err
		}
	}
	return m.streamResp, m.streamErr
}

func newChatHandlerTestContext(body string, accept string) (*gin.Context, *httptest.ResponseRecorder) {
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/example_cognitive/chat", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	if accept != "" {
		c.Request.Header.Set("Accept", accept)
	}
	authcontext.SetRequestContext(c, &authcontext.RequestContext{OrganizationID: 1, AccountID: 1})
	features.SetEntitlement(c, &features.Entitlement{
		Features: map[string]bool{"ai_assistant": true},
	})
	return c, rec
}

func TestChatStreamingSSEFraming(t *testing.T) {
	handler := NewHandler(&mockRAGService{
		streamEvents: []domain.StreamEvent{
			{Content: "Hola ", Done: false},
			{Content: "mundo", Done: false},
			{Done: true},
		},
		streamResp: &domain.ChatResponse{
			SessionID: 7,
			Message:   &domain.ChatMessage{ID: 11, SessionID: 7, Role: domain.ChatRoleAssistant, Content: "Hola mundo"},
			TokensUsed: 42,
		},
	}, nil)

	c, rec := newChatHandlerTestContext(`{"message":"hola","stream":true}`, "")
	handler.Chat(c)

	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "text/event-stream", rec.Header().Get("Content-Type"))

	body := rec.Body.String()
	assert.Contains(t, body, `data: {"token":"Hola "}`)
	assert.Contains(t, body, `data: {"token":"mundo"}`)
	assert.Contains(t, body, `"done":true`)
	assert.Contains(t, body, `"session_id":7`)
	assert.Contains(t, body, `"message_id":11`)
}

func TestChatStreamingViaAcceptHeader(t *testing.T) {
	handler := NewHandler(&mockRAGService{
		streamEvents: []domain.StreamEvent{
			{Content: "respuesta", Done: false},
			{Done: true},
		},
		streamResp: &domain.ChatResponse{
			SessionID: 3,
			Message:   &domain.ChatMessage{ID: 9, SessionID: 3, Role: domain.ChatRoleAssistant, Content: "respuesta"},
		},
	}, nil)

	c, rec := newChatHandlerTestContext(`{"message":"hola"}`, "text/event-stream")
	handler.Chat(c)

	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "text/event-stream", rec.Header().Get("Content-Type"))
	assert.Contains(t, rec.Body.String(), `data: {"token":"respuesta"}`)
}

func TestChatNonStreamingReturnsJSON(t *testing.T) {
	handler := NewHandler(&mockRAGService{
		chatResponse: &domain.ChatResponse{
			SessionID: 1,
			Message:   &domain.ChatMessage{ID: 1, SessionID: 1, Role: domain.ChatRoleAssistant, Content: "respuesta"},
			TokensUsed: 10,
		},
	}, nil)

	c, rec := newChatHandlerTestContext(`{"message":"hola"}`, "")
	handler.Chat(c)

	require.Equal(t, http.StatusOK, rec.Code)
	assert.NotEqual(t, "text/event-stream", rec.Header().Get("Content-Type"))

	var resp domain.ChatResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, "respuesta", resp.Message.Content)
	assert.Equal(t, int32(10), resp.TokensUsed)
}

func TestChatStreamingErrorEmitsErrorEvent(t *testing.T) {
	handler := NewHandler(&mockRAGService{
		streamErr: context.DeadlineExceeded,
	}, nil)

	c, rec := newChatHandlerTestContext(`{"message":"hola","stream":true}`, "")
	handler.Chat(c)

	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "text/event-stream", rec.Header().Get("Content-Type"))
	assert.Contains(t, rec.Body.String(), "event: error")
}
