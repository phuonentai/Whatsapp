package whatsapp

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestSendTextMessage_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Errorf("expected Bearer test-token, got %s", r.Header.Get("Authorization"))
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("expected application/json, got %s", r.Header.Get("Content-Type"))
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"messaging_product":"whatsapp","contacts":[{"input":"+573001234567","wa_id":"573001234567"}],"messages":[{"id":"wamid.HBgNNTc"}]}`))
	}))
	defer server.Close()

	client := NewClient()
	msgID, err := client.SendTextMessage(
		context.Background(),
		"test-token",
		server.URL,
		"v21.0",
		"12345",
		"+573001234567",
		"Hello, world!",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if msgID != "wamid.HBgNNTc" {
		t.Errorf("expected wamid.HBgNNTc, got %s", msgID)
	}
}

func TestSendTextMessage_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":{"message":"(#100) Invalid parameter","type":"OAuthException","code":100}}`))
	}))
	defer server.Close()

	client := NewClient()
	_, err := client.SendTextMessage(
		context.Background(),
		"test-token",
		server.URL,
		"v21.0",
		"12345",
		"+573001234567",
		"Hello",
	)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestSendTextMessage_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	client := NewClient()
	_, err := client.SendTextMessage(
		context.Background(),
		"test-token",
		server.URL,
		"v21.0",
		"12345",
		"+573001234567",
		"Hello",
	)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestCircuitBreaker_OpenCloseTransitions(t *testing.T) {
	cb := NewCircuitBreaker(3, 50*time.Millisecond, 1)

	if !cb.Allow() {
		t.Error("expected circuit to be closed initially")
	}

	cb.Failure()
	cb.Failure()
	cb.Failure()

	if cb.Allow() {
		t.Error("expected circuit to be open after 3 failures")
	}

	time.Sleep(60 * time.Millisecond)

	if !cb.Allow() {
		t.Error("expected circuit to allow half-open probe after cooldown")
	}

	cb.Success()

	if !cb.Allow() {
		t.Error("expected circuit to be closed after success")
	}
}

func TestCircuitBreaker_StateAfterSuccess(t *testing.T) {
	cb := NewCircuitBreaker(2, time.Minute, 1)

	cb.Failure()
	if s := cb.State(); s != CircuitClosed {
		t.Errorf("expected closed state after 1 failure, got %d", s)
	}

	cb.Failure()
	if s := cb.State(); s != CircuitOpen {
		t.Errorf("expected open state after 2 failures, got %d", s)
	}

	cb.mu.Lock()
	cb.lastFailureAt = time.Now().Add(-2 * time.Minute)
	cb.mu.Unlock()

	if !cb.Allow() {
		t.Error("expected to transition to half-open after cooldown")
	}

	cb.Success()
	if s := cb.State(); s != CircuitClosed {
		t.Errorf("expected closed state after success, got %d", s)
	}
}

func TestCircuitBreaker_HalfOpenLimitsProbes(t *testing.T) {
	cb := NewCircuitBreaker(1, time.Millisecond, 2)

	cb.Failure()
	time.Sleep(2 * time.Millisecond)

	if !cb.Allow() {
		t.Error("expected first half-open probe to be allowed")
	}
	if !cb.Allow() {
		t.Error("expected second half-open probe to be allowed")
	}
	if cb.Allow() {
		t.Error("expected third half-open probe to be blocked")
	}
}

func TestClientWithBreaker_BlocksWhenOpen(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := NewClientWithBreaker(2, time.Minute, 1)

	_, err := client.SendTextMessage(context.Background(), "tok", server.URL, "v21.0", "123", "+573001234567", "hi")
	if err == nil {
		t.Fatal("expected error from API, got nil")
	}

	_, err = client.SendTextMessage(context.Background(), "tok", server.URL, "v21.0", "123", "+573001234567", "hi")
	if err == nil {
		t.Fatal("expected error from API, got nil")
	}

	_, err = client.SendTextMessage(context.Background(), "tok", server.URL, "v21.0", "123", "+573001234567", "hi")
	if err == nil {
		t.Fatal("expected circuit breaker to block request")
	}
}
