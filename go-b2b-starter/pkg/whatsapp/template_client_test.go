package whatsapp

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestSendTemplateMessage_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Errorf("expected Bearer test-token, got %s", r.Header.Get("Authorization"))
		}
		if !strings.HasSuffix(r.URL.Path, "/v21.0/12345/messages") {
			t.Errorf("expected messages path, got %s", r.URL.Path)
		}

		var payload TemplateMessageRequest
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Errorf("failed to decode request: %v", err)
			return
		}
		if payload.Type != "template" {
			t.Errorf("expected type template, got %s", payload.Type)
		}
		if payload.Template.Name != "confirmacion_pedido" {
			t.Errorf("expected template name confirmacion_pedido, got %s", payload.Template.Name)
		}
		if payload.Template.Language.Policy != "deterministic" || payload.Template.Language.Code != "es" {
			t.Errorf("unexpected language block: %+v", payload.Template.Language)
		}
		if len(payload.Template.Components) != 1 || payload.Template.Components[0].Type != "body" {
			t.Fatalf("expected single body component, got %+v", payload.Template.Components)
		}
		params := payload.Template.Components[0].Parameters
		if len(params) != 2 || params[0].Text != "María" || params[1].Text != "Pedido #1234" {
			t.Errorf("unexpected parameters: %+v", params)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"messaging_product":"whatsapp","contacts":[{"input":"+573001234567","wa_id":"573001234567"}],"messages":[{"id":"wamid.HBgNNTc"}]}`))
	}))
	defer server.Close()

	client := NewClientWithBreaker(5, 10*time.Second, 2)
	msgID, err := client.SendTemplateMessage(
		context.Background(),
		"test-token",
		server.URL,
		"v21.0",
		"12345",
		"+573001234567",
		"confirmacion_pedido",
		"es",
		[]string{"María", "Pedido #1234"},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if msgID != "wamid.HBgNNTc" {
		t.Errorf("expected wamid.HBgNNTc, got %s", msgID)
	}
}

func TestSendTemplateMessage_OpensSharedCircuitBreaker(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error":{"message":"boom","type":"Server","code":1}}`))
	}))
	defer server.Close()

	client := NewClientWithBreaker(5, 10*time.Second, 2)
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		if _, err := client.SendTemplateMessage(ctx, "t", server.URL, "v21.0", "12345", "+573001234567", "n", "es", []string{"x"}); err == nil {
			t.Fatalf("iteration %d: expected 5xx error", i)
		}
	}

	// Template and text sends both fail fast once the shared breaker is open.
	if _, err := client.SendTemplateMessage(ctx, "t", server.URL, "v21.0", "12345", "+573001234567", "n", "es", []string{"x"}); err == nil || !strings.Contains(err.Error(), "circuit breaker open") {
		t.Fatalf("expected template call blocked by open breaker, got %v", err)
	}
	if _, err := client.SendTextMessage(ctx, "t", server.URL, "v21.0", "12345", "+573001234567", "hello"); err == nil || !strings.Contains(err.Error(), "circuit breaker open") {
		t.Fatalf("expected text call blocked by open breaker, got %v", err)
	}
}

func TestSendTemplateMessage_APIFailureReturnsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":{"message":"(#100) Invalid template","type":"GraphMethodException","code":100}}`))
	}))
	defer server.Close()

	client := NewClient()
	_, err := client.sendTemplateMessage(
		context.Background(),
		"test-token",
		server.URL,
		"v21.0",
		"12345",
		"+573001234567",
		"n",
		"es",
		[]string{"x"},
	)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "whatsapp api error (code 100)") {
		t.Errorf("expected api error with code 100, got %v", err)
	}
}
