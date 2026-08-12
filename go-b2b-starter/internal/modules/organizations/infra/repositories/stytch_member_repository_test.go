package repositories_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/moasq/go-b2b-starter/internal/modules/organizations/domain"
	"github.com/moasq/go-b2b-starter/internal/modules/organizations/infra/repositories"
	loggerDomain "github.com/moasq/go-b2b-starter/internal/platform/logger/domain"
	stytchcfg "github.com/moasq/go-b2b-starter/internal/platform/stytch"
)

type noopLogger struct{}

func (noopLogger) Debug(msg string, fields ...loggerDomain.Fields) {}
func (noopLogger) Info(msg string, fields ...loggerDomain.Fields)  {}
func (noopLogger) Warn(msg string, fields ...loggerDomain.Fields)  {}
func (noopLogger) Error(msg string, fields ...loggerDomain.Fields) {}
func (noopLogger) Fatal(msg string, fields ...loggerDomain.Fields) {}
func (noopLogger) WithFields(fields loggerDomain.Fields) loggerDomain.Logger {
	return noopLogger{}
}

// newRevoker spins up a stytchMemberRepository backed by an httptest server
// and a shared circuit-breaker client (threshold = maxFailures).
func newRevoker(t *testing.T, handler http.Handler, maxFailures int) (domain.SessionRevoker, *stytchcfg.Client) {
	t.Helper()

	server := httptest.NewServer(serveJWKS(handler))
	t.Cleanup(server.Close)

	cfg := stytchcfg.Config{
		ProjectID:                  "project-test-123",
		Secret:                     "secret-test-123",
		Env:                        stytchcfg.EnvTest,
		BaseURL:                    server.URL,
		APITimeout:                 5 * time.Second,
		CircuitBreakerThreshold:    maxFailures,
		CircuitBreakerTimeout:      time.Minute,
		CircuitBreakerHalfOpenProbes: 1,
	}
	client, err := stytchcfg.NewClient(cfg)
	if err != nil {
		t.Fatalf("failed to create stytch client: %v", err)
	}

	repo := repositories.NewStytchMemberRepository(client, cfg, noopLogger{})
	var revoker domain.SessionRevoker = repo
	return revoker, client
}

// serveJWKS short-circuits the JWKS bootstrap fetch the Stytch client makes at
// construction; Get/Revoke never touch it at runtime.
func serveJWKS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/sessions/jwks/") {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"keys":[]}`))
			return
		}
		next.ServeHTTP(w, r)
	})
}

func stytchError(status int, errorType string) []byte {
	body, _ := json.Marshal(map[string]any{
		"status_code":   status,
		"error_type":    errorType,
		"error_message": "boom",
		"request_id":    "req-test-1",
	})
	return body
}

func TestRevokeMemberSessionsHappyPath(t *testing.T) {
	var listCalls atomic.Int32
	var revokeCalls atomic.Int32

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/v1/b2b/sessions":
			listCalls.Add(1)
			// Assert org+member filter is passed through.
			if got := r.URL.Query().Get("organization_id"); got != "org-1" {
				t.Errorf("expected organization_id=org-1, got %q", got)
			}
			if got := r.URL.Query().Get("member_id"); got != "member-1" {
				t.Errorf("expected member_id=member-1, got %q", got)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"member_sessions":[
				{"member_session_id":"session-1"},
				{"member_session_id":"session-2"}
			]}`))
		case r.Method == http.MethodPost && r.URL.Path == "/v1/b2b/sessions/revoke":
			revokeCalls.Add(1)
			var body map[string]any
			_ = json.NewDecoder(r.Body).Decode(&body)
			sid, _ := body["member_session_id"].(string)
			if sid != "session-1" && sid != "session-2" {
				t.Errorf("unexpected member_session_id in revoke body: %q", sid)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"request_id":"req-1","status_code":200}`))
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	})

	revoker, _ := newRevoker(t, handler, 5)

	if err := revoker.RevokeMemberSessions(context.Background(), "org-1", "member-1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if listCalls.Load() != 1 {
		t.Fatalf("expected 1 list call, got %d", listCalls.Load())
	}
	if revokeCalls.Load() != 2 {
		t.Fatalf("expected 2 revoke calls, got %d", revokeCalls.Load())
	}
}

func TestRevokeMemberSessionsAlreadyRevokedIsNoOp(t *testing.T) {
	var revokeCalls atomic.Int32

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/v1/b2b/sessions":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"member_sessions":[{"member_session_id":"session-1"}]}`))
		case r.Method == http.MethodPost && r.URL.Path == "/v1/b2b/sessions/revoke":
			revokeCalls.Add(1)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write(stytchError(http.StatusNotFound, "session_not_found"))
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	})

	revoker, _ := newRevoker(t, handler, 5)

	// Revoking an already-revoked session must be treated as a no-op success.
	if err := revoker.RevokeMemberSessions(context.Background(), "org-1", "member-1"); err != nil {
		t.Fatalf("expected idempotent no-op success, got error: %v", err)
	}
	if revokeCalls.Load() != 1 {
		t.Fatalf("expected 1 revoke attempt, got %d", revokeCalls.Load())
	}
}

func TestRevokeMemberSessionsNoSessionsIsNoOp(t *testing.T) {
	var revokeCalls atomic.Int32

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/v1/b2b/sessions":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"member_sessions":[]}`))
		case r.Method == http.MethodPost && r.URL.Path == "/v1/b2b/sessions/revoke":
			revokeCalls.Add(1)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"request_id":"req-1","status_code":200}`))
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	})

	revoker, _ := newRevoker(t, handler, 5)

	if err := revoker.RevokeMemberSessions(context.Background(), "org-1", "member-1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if revokeCalls.Load() != 0 {
		t.Fatalf("expected no revoke calls, got %d", revokeCalls.Load())
	}
}

func TestRevokeMemberSessionsBreakerOpenDefers(t *testing.T) {
	var apiCalls atomic.Int32

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		apiCalls.Add(1)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write(stytchError(http.StatusInternalServerError, "internal_server_error"))
	})

	// Threshold 2: two failures trip the breaker open.
	revoker, _ := newRevoker(t, handler, 2)

	ctx := context.Background()
	if err := revoker.RevokeMemberSessions(ctx, "org-1", "member-1"); err == nil {
		t.Fatal("expected error from failing list call")
	}
	if err := revoker.RevokeMemberSessions(ctx, "org-1", "member-1"); err == nil {
		t.Fatal("expected error from failing list call")
	}

	// Breaker is now open: the call fails fast WITHOUT reaching the API.
	err := revoker.RevokeMemberSessions(ctx, "org-1", "member-1")
	if err == nil {
		t.Fatal("expected circuit-open error")
	}
	if !errors.Is(err, stytchcfg.ErrCircuitOpen) {
		t.Fatalf("expected ErrCircuitOpen, got: %v", err)
	}
	if apiCalls.Load() != 2 {
		t.Fatalf("expected API calls to stop after breaker tripped; got %d", apiCalls.Load())
	}
}

func TestRevokeMemberSessionsRevokeFailureSurfacesError(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/v1/b2b/sessions":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"member_sessions":[{"member_session_id":"session-1"}]}`))
		case r.Method == http.MethodPost && r.URL.Path == "/v1/b2b/sessions/revoke":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write(stytchError(http.StatusInternalServerError, "internal_server_error"))
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	})

	revoker, _ := newRevoker(t, handler, 5)

	err := revoker.RevokeMemberSessions(context.Background(), "org-1", "member-1")
	if err == nil {
		t.Fatal("expected revoke failure to surface")
	}
	if !strings.Contains(err.Error(), "revoke member session") {
		t.Fatalf("expected revoke context in error, got: %v", err)
	}
}

func TestRevokeMemberSessionsValidation(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})
	revoker, _ := newRevoker(t, handler, 5)

	if err := revoker.RevokeMemberSessions(context.Background(), "", "member-1"); !errors.Is(err, domain.ErrAuthOrganizationIDRequired) {
		t.Fatalf("expected ErrAuthOrganizationIDRequired, got: %v", err)
	}
	if err := revoker.RevokeMemberSessions(context.Background(), "org-1", ""); !errors.Is(err, domain.ErrAuthMemberIDRequired) {
		t.Fatalf("expected ErrAuthMemberIDRequired, got: %v", err)
	}

	// Sanity: the repo short-circuits before any API call on empty IDs.
	if err := revoker.RevokeMemberSessions(context.Background(), "", "member-1"); err == nil {
		t.Fatal("expected validation error")
	}
}
