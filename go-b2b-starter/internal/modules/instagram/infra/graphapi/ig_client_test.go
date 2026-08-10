package graphapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/moasq/go-b2b-starter/pkg/whatsapp"
)

func newTestServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return srv
}

func TestSendTextMessage_CorrectRequest(t *testing.T) {
	var gotPath, gotAuth, gotCT string
	var gotBody map[string]any

	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("Authorization")
		gotCT = r.Header.Get("Content-Type")
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"recipient_id": "customer-1", "message_id": "mid.sent.1"}`))
	})

	client := NewIGClient(ClientConfig{}, srv.Client())
	msgID, err := client.SendTextMessage(context.Background(), "tok-1", srv.URL, "v21.0", "business-1", "customer-1", "Hola!")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if msgID != "mid.sent.1" {
		t.Fatalf("expected mid.sent.1, got %s", msgID)
	}
	if gotPath != "/v21.0/business-1/messages" {
		t.Fatalf("unexpected path %s", gotPath)
	}
	if gotAuth != "Bearer tok-1" {
		t.Fatalf("unexpected auth header %s", gotAuth)
	}
	if gotCT != "application/json" {
		t.Fatalf("unexpected content type %s", gotCT)
	}
	recipient := gotBody["recipient"].(map[string]any)
	if recipient["id"] != "customer-1" {
		t.Fatalf("unexpected recipient %v", recipient)
	}
	message := gotBody["message"].(map[string]any)
	if message["text"] != "Hola!" {
		t.Fatalf("unexpected message %v", message)
	}
}

func TestGetIGUser_ResolvesProfile(t *testing.T) {
	var gotPath string
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path + "?" + r.URL.RawQuery
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id": "customer-1", "username": "cliente.ig", "profile_picture_url": "https://cdn.ig/pic.jpg"}`))
	})

	client := NewIGClient(ClientConfig{}, srv.Client())
	user, err := client.GetIGUser(context.Background(), "tok-1", srv.URL, "v21.0", "customer-1")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if user.Username != "cliente.ig" || user.ProfilePictureURL != "https://cdn.ig/pic.jpg" {
		t.Fatalf("unexpected user %+v", user)
	}
	if gotPath != "/v21.0/customer-1?fields=username%2Cprofile_picture_url" && gotPath != "/v21.0/customer-1?fields=username,profile_picture_url" {
		t.Fatalf("unexpected request %s", gotPath)
	}
}

func TestRefreshToken_ExchangesAndReturnsExpiry(t *testing.T) {
	var gotPath string
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path + "?" + r.URL.RawQuery
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"access_token": "new-token", "token_type": "bearer", "expires_in": 5184000}`))
	})

	client := NewIGClient(ClientConfig{}, srv.Client())
	token, expiry, err := client.RefreshToken(context.Background(), "app-1", "secret-1", srv.URL, "v21.0", "old-token")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if token != "new-token" {
		t.Fatalf("expected new-token, got %s", token)
	}
	if expiry == nil || expiry.Before(time.Now().Add(50*24*time.Hour)) {
		t.Fatalf("expected expiry ~60 days out, got %v", expiry)
	}
	if gotPath == "" || len(gotPath) < 10 {
		t.Fatalf("unexpected request %s", gotPath)
	}
}

func TestRefreshToken_APIError(t *testing.T) {
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error": {"message": "invalid token", "type": "OAuthException", "code": 190}}`))
	})

	client := NewIGClient(ClientConfig{}, srv.Client())
	_, _, err := client.RefreshToken(context.Background(), "app-1", "secret-1", srv.URL, "v21.0", "bad-token")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var gErr *GraphError
	if !errors.As(err, &gErr) || gErr.Code != 190 {
		t.Fatalf("expected GraphError code 190, got %v", err)
	}
}

func TestCircuitBreaker_OpensAfterFailures(t *testing.T) {
	var calls atomic.Int32
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error": {"message": "boom", "code": 1}}`))
	})

	breaker := whatsapp.NewCircuitBreaker(2, 30*time.Second, 1)
	client := newIGClient(ClientConfig{}, srv.Client(), breaker)

	for i := 0; i < 2; i++ {
		_, err := client.SendTextMessage(context.Background(), "tok", srv.URL, "v21.0", "b1", "c1", "hi")
		if err == nil {
			t.Fatal("expected error")
		}
	}

	// Third call must fail fast without hitting the server.
	_, err := client.SendTextMessage(context.Background(), "tok", srv.URL, "v21.0", "b1", "c1", "hi")
	if err == nil {
		t.Fatal("expected breaker-open error")
	}
	if calls.Load() != 2 {
		t.Fatalf("expected 2 server calls, got %d", calls.Load())
	}
}
