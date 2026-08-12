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
	stytchcfg "github.com/moasq/go-b2b-starter/internal/platform/stytch"
)

// fakeLocalOrgRepo satisfies domain.OrganizationRepository for the Stytch
// organization repository constructor. UpdateMfaPolicy never touches it.
type fakeLocalOrgRepo struct{}

func (f *fakeLocalOrgRepo) Create(ctx context.Context, org *domain.Organization) (*domain.Organization, error) {
	return org, nil
}
func (f *fakeLocalOrgRepo) GetByID(ctx context.Context, id int32) (*domain.Organization, error) {
	return nil, domain.ErrOrganizationNotFound
}
func (f *fakeLocalOrgRepo) GetBySlug(ctx context.Context, slug string) (*domain.Organization, error) {
	return nil, domain.ErrOrganizationNotFound
}
func (f *fakeLocalOrgRepo) GetByStytchID(ctx context.Context, stytchOrgID string) (*domain.Organization, error) {
	return nil, domain.ErrOrganizationNotFound
}
func (f *fakeLocalOrgRepo) GetByUserEmail(ctx context.Context, email string) (*domain.Organization, error) {
	return nil, domain.ErrOrganizationNotFound
}
func (f *fakeLocalOrgRepo) Update(ctx context.Context, org *domain.Organization) (*domain.Organization, error) {
	return org, nil
}
func (f *fakeLocalOrgRepo) UpdateStytchInfo(ctx context.Context, id int32, stytchOrgID, stytchConnectionID, stytchConnectionName string) (*domain.Organization, error) {
	return nil, nil
}
func (f *fakeLocalOrgRepo) Delete(ctx context.Context, id int32) error {
	return nil
}
func (f *fakeLocalOrgRepo) List(ctx context.Context, limit, offset int32) ([]*domain.Organization, error) {
	return nil, nil
}
func (f *fakeLocalOrgRepo) GetStats(ctx context.Context, id int32) (*domain.OrganizationStats, error) {
	return nil, nil
}

// newMfaPolicyUpdater spins up a Stytch organization repository backed by an
// httptest server and a shared circuit-breaker client (threshold = maxFailures).
func newMfaPolicyUpdater(t *testing.T, handler http.Handler, maxFailures int) (domain.MfaPolicyUpdater, *stytchcfg.Client) {
	t.Helper()

	server := httptest.NewServer(serveJWKS(handler))
	t.Cleanup(server.Close)

	cfg := stytchcfg.Config{
		ProjectID:                    "project-test-123",
		Secret:                       "secret-test-123",
		Env:                          stytchcfg.EnvTest,
		BaseURL:                      server.URL,
		APITimeout:                   5 * time.Second,
		CircuitBreakerThreshold:      maxFailures,
		CircuitBreakerTimeout:        time.Minute,
		CircuitBreakerHalfOpenProbes: 1,
	}
	client, err := stytchcfg.NewClient(cfg)
	if err != nil {
		t.Fatalf("failed to create stytch client: %v", err)
	}

	repo := repositories.NewStytchOrganizationRepository(client, noopLogger{}, &fakeLocalOrgRepo{})
	var updater domain.MfaPolicyUpdater = repo
	return updater, client
}

func TestUpdateMfaPolicyHappyPath(t *testing.T) {
	var updateCalls atomic.Int32

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPut && r.URL.Path == "/v1/b2b/organizations/org-1":
			updateCalls.Add(1)
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Errorf("failed to decode update body: %v", err)
			}
			if got := body["mfa_policy"]; got != "REQUIRED_FOR_ALL" {
				t.Errorf("expected mfa_policy=REQUIRED_FOR_ALL, got %v", got)
			}
			if got := body["mfa_methods"]; got != "RESTRICTED" {
				t.Errorf("expected mfa_methods=RESTRICTED, got %v", got)
			}
			allowed, ok := body["allowed_mfa_methods"].([]any)
			if !ok || len(allowed) != 1 || allowed[0] != "totp" {
				t.Errorf("expected allowed_mfa_methods=[totp], got %v", body["allowed_mfa_methods"])
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"organization_id":"org-1","status_code":200}`))
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	})

	updater, _ := newMfaPolicyUpdater(t, handler, 5)

	err := updater.UpdateMfaPolicy(
		context.Background(),
		"org-1",
		domain.MfaPolicyRequiredForAll,
		domain.MfaMethodsRestricted,
		[]domain.MfaMethod{domain.MfaMethodTOTP},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updateCalls.Load() != 1 {
		t.Fatalf("expected 1 update call, got %d", updateCalls.Load())
	}
}

func TestUpdateMfaPolicyStytch4xxSurfacesError(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write(stytchError(http.StatusBadRequest, "invalid_request"))
	})

	updater, _ := newMfaPolicyUpdater(t, handler, 5)

	err := updater.UpdateMfaPolicy(
		context.Background(),
		"org-1",
		domain.MfaPolicyRequiredForAll,
		domain.MfaMethodsRestricted,
		[]domain.MfaMethod{domain.MfaMethodTOTP},
	)
	if err == nil {
		t.Fatal("expected 4xx to surface an error")
	}
	// A client 4xx must NOT be treated as an availability problem (503).
	if errors.Is(err, domain.ErrMfaPolicyUnavailable) {
		t.Fatalf("4xx must not map to ErrMfaPolicyUnavailable, got: %v", err)
	}
	if !strings.Contains(err.Error(), "bad request") {
		t.Fatalf("expected wrapped Stytch error detail, got: %v", err)
	}
}

func TestUpdateMfaPolicyStytch5xxMapsUnavailable(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write(stytchError(http.StatusInternalServerError, "internal_server_error"))
	})

	updater, _ := newMfaPolicyUpdater(t, handler, 5)

	err := updater.UpdateMfaPolicy(
		context.Background(),
		"org-1",
		domain.MfaPolicyRequiredForAll,
		domain.MfaMethodsRestricted,
		[]domain.MfaMethod{domain.MfaMethodTOTP},
	)
	if err == nil {
		t.Fatal("expected 5xx to surface an error")
	}
	if !errors.Is(err, domain.ErrMfaPolicyUnavailable) {
		t.Fatalf("expected ErrMfaPolicyUnavailable for 5xx, got: %v", err)
	}
}

func TestUpdateMfaPolicyBreakerOpenMapsUnavailable(t *testing.T) {
	var apiCalls atomic.Int32

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		apiCalls.Add(1)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write(stytchError(http.StatusInternalServerError, "internal_server_error"))
	})

	// Threshold 2: two failures trip the breaker open.
	updater, _ := newMfaPolicyUpdater(t, handler, 2)

	ctx := context.Background()
	args := []domain.MfaMethod{domain.MfaMethodTOTP}

	for i := 0; i < 2; i++ {
		if err := updater.UpdateMfaPolicy(ctx, "org-1", domain.MfaPolicyRequiredForAll, domain.MfaMethodsRestricted, args); err == nil {
			t.Fatal("expected error from failing update call")
		}
	}

	// Breaker is now open: fails fast WITHOUT reaching the API, mapped to the
	// 503 domain sentinel.
	err := updater.UpdateMfaPolicy(ctx, "org-1", domain.MfaPolicyRequiredForAll, domain.MfaMethodsRestricted, args)
	if err == nil {
		t.Fatal("expected circuit-open error")
	}
	if !errors.Is(err, domain.ErrMfaPolicyUnavailable) {
		t.Fatalf("expected ErrMfaPolicyUnavailable (breaker open), got: %v", err)
	}
	if apiCalls.Load() != 2 {
		t.Fatalf("expected API calls to stop after breaker tripped; got %d", apiCalls.Load())
	}
}

func TestUpdateMfaPolicyValidation(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})
	updater, _ := newMfaPolicyUpdater(t, handler, 5)

	ctx := context.Background()

	// Empty org ID short-circuits before any API call.
	if err := updater.UpdateMfaPolicy(ctx, "", domain.MfaPolicyOptional, domain.MfaMethodsAllAllowed, nil); !errors.Is(err, domain.ErrAuthOrganizationIDRequired) {
		t.Fatalf("expected ErrAuthOrganizationIDRequired, got: %v", err)
	}

	// Invalid policy value.
	if err := updater.UpdateMfaPolicy(ctx, "org-1", domain.MfaPolicy("NEVER"), domain.MfaMethodsAllAllowed, nil); !errors.Is(err, domain.ErrInvalidMfaPolicy) {
		t.Fatalf("expected ErrInvalidMfaPolicy, got: %v", err)
	}

	// Invalid method value.
	if err := updater.UpdateMfaPolicy(ctx, "org-1", domain.MfaPolicyOptional, domain.MfaMethods("MAYBE"), nil); !errors.Is(err, domain.ErrInvalidMfaMethods) {
		t.Fatalf("expected ErrInvalidMfaMethods, got: %v", err)
	}

	// Restricted methods require a non-empty allowlist.
	if err := updater.UpdateMfaPolicy(ctx, "org-1", domain.MfaPolicyOptional, domain.MfaMethodsRestricted, nil); err == nil {
		t.Fatal("expected error for restricted methods without allowlist")
	}

	// Unsupported allowed method.
	if err := updater.UpdateMfaPolicy(ctx, "org-1", domain.MfaPolicyOptional, domain.MfaMethodsRestricted, []domain.MfaMethod{"email"}); !errors.Is(err, domain.ErrInvalidMfaMethod) {
		t.Fatalf("expected ErrInvalidMfaMethod, got: %v", err)
	}
}

// newAuthPolicyUpdater spins up a Stytch organization repository backed by an
// httptest server and a shared circuit-breaker client (threshold = maxFailures)
// exposed through the OrgAuthPolicyUpdater domain contract.
func newAuthPolicyUpdater(t *testing.T, handler http.Handler, maxFailures int) (domain.OrgAuthPolicyUpdater, *stytchcfg.Client) {
	t.Helper()

	server := httptest.NewServer(serveJWKS(handler))
	t.Cleanup(server.Close)

	cfg := stytchcfg.Config{
		ProjectID:                    "project-test-123",
		Secret:                       "secret-test-123",
		Env:                          stytchcfg.EnvTest,
		BaseURL:                      server.URL,
		APITimeout:                   5 * time.Second,
		CircuitBreakerThreshold:      maxFailures,
		CircuitBreakerTimeout:        time.Minute,
		CircuitBreakerHalfOpenProbes: 1,
	}
	client, err := stytchcfg.NewClient(cfg)
	if err != nil {
		t.Fatalf("failed to create stytch client: %v", err)
	}

	repo := repositories.NewStytchOrganizationRepository(client, noopLogger{}, &fakeLocalOrgRepo{})
	var updater domain.OrgAuthPolicyUpdater = repo
	return updater, client
}

// orgResponse renders a minimal Stytch organization object JSON.
func orgResponse(fields map[string]any) []byte {
	org := map[string]any{"organization_id": "org-1"}
	for k, v := range fields {
		org[k] = v
	}
	body, _ := json.Marshal(map[string]any{"organization": org})
	return body
}

func TestGetAuthPolicyHappyPath(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/v1/b2b/organizations/org-1" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(orgResponse(map[string]any{
			"email_jit_provisioning":                   "RESTRICTED",
			"email_allowed_domains":                    []string{"acme.com"},
			"auth_methods":                             "RESTRICTED",
			"allowed_auth_methods":                     []string{"magic_link", "email_otp"},
			"sso_jit_provisioning":                     "RESTRICTED",
			"sso_jit_provisioning_allowed_connections": []string{"conn-1"},
			"sso_default_connection_id":                "conn-1",
			"sso_active_connections":                   []map[string]any{{"connection_id": "conn-1"}, {"connection_id": "conn-2"}},
		}))
	})

	updater, _ := newAuthPolicyUpdater(t, handler, 5)

	policy, err := updater.GetAuthPolicy(context.Background(), "org-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if policy.EmailJITProvisioning != domain.JitPolicyDomainRestricted {
		t.Errorf("expected DOMAIN_RESTRICTED email JIT, got %q", policy.EmailJITProvisioning)
	}
	if len(policy.EmailAllowedDomains) != 1 || policy.EmailAllowedDomains[0] != "acme.com" {
		t.Errorf("expected allowed domains [acme.com], got %v", policy.EmailAllowedDomains)
	}
	if !policy.AuthMethodsRestricted {
		t.Error("expected auth_methods_restricted=true")
	}
	if len(policy.AllowedAuthMethods) != 2 || policy.AllowedAuthMethods[0] != domain.AuthMethodMagicLink {
		t.Errorf("expected allowed auth methods [magic_link email_otp], got %v", policy.AllowedAuthMethods)
	}
	if policy.SSOJITProvisioning != domain.SsoJitPolicyConnectionRestricted {
		t.Errorf("expected CONNECTION_RESTRICTED SSO JIT, got %q", policy.SSOJITProvisioning)
	}
	if len(policy.SSOJITProvisioningAllowedConnections) != 1 || policy.SSOJITProvisioningAllowedConnections[0] != "conn-1" {
		t.Errorf("expected sso allowed connections [conn-1], got %v", policy.SSOJITProvisioningAllowedConnections)
	}
	if policy.SSODefaultConnectionID != "conn-1" {
		t.Errorf("expected sso_default_connection_id=conn-1, got %q", policy.SSODefaultConnectionID)
	}
	if len(policy.SSOActiveConnectionIDs) != 2 {
		t.Errorf("expected 2 active connection ids, got %v", policy.SSOActiveConnectionIDs)
	}
}

func TestGetAuthPolicyCollapsesProviderDefaults(t *testing.T) {
	// An org that never saved a policy: provider defaults (email JIT
	// NOT_ALLOWED, SSO JIT ALL_ALLOWED per the SDK docs, auth_methods
	// ALL_ALLOWED) must collapse to the safe disabled mirror values.
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(orgResponse(map[string]any{
			"email_jit_provisioning": "NOT_ALLOWED",
			"sso_jit_provisioning":   "ALL_ALLOWED",
			"auth_methods":           "ALL_ALLOWED",
		}))
	})

	updater, _ := newAuthPolicyUpdater(t, handler, 5)

	policy, err := updater.GetAuthPolicy(context.Background(), "org-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if policy.EmailJITProvisioning != domain.JitPolicyDisabled {
		t.Errorf("expected DISABLED email JIT for NOT_ALLOWED, got %q", policy.EmailJITProvisioning)
	}
	// Org-wide ALL_ALLOWED SSO JIT is never written by the platform; the
	// mirror renders it as "not governed" (disabled).
	if policy.SSOJITProvisioning != domain.SsoJitPolicyDisabled {
		t.Errorf("expected DISABLED SSO JIT for ALL_ALLOWED, got %q", policy.SSOJITProvisioning)
	}
	if policy.AuthMethodsRestricted {
		t.Error("expected auth_methods_restricted=false for ALL_ALLOWED org")
	}
}

func TestGetAuthPolicyStytch5xxMapsUnavailable(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write(stytchError(http.StatusInternalServerError, "internal_server_error"))
	})

	updater, _ := newAuthPolicyUpdater(t, handler, 5)

	_, err := updater.GetAuthPolicy(context.Background(), "org-1")
	if err == nil {
		t.Fatal("expected 5xx to surface an error")
	}
	if !errors.Is(err, domain.ErrAuthPolicyUnavailable) {
		t.Fatalf("expected ErrAuthPolicyUnavailable for 5xx, got: %v", err)
	}
}

func TestGetAuthPolicyBreakerOpenMapsUnavailable(t *testing.T) {
	var apiCalls atomic.Int32

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		apiCalls.Add(1)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write(stytchError(http.StatusInternalServerError, "internal_server_error"))
	})

	// Threshold 2: two failures trip the breaker open.
	updater, _ := newAuthPolicyUpdater(t, handler, 2)

	ctx := context.Background()
	for i := 0; i < 2; i++ {
		if _, err := updater.GetAuthPolicy(ctx, "org-1"); err == nil {
			t.Fatal("expected error from failing read call")
		}
	}

	// Breaker is now open: the read fails fast WITHOUT reaching the API,
	// mapped to the 503 domain sentinel.
	_, err := updater.GetAuthPolicy(ctx, "org-1")
	if err == nil {
		t.Fatal("expected circuit-open error")
	}
	if !errors.Is(err, domain.ErrAuthPolicyUnavailable) {
		t.Fatalf("expected ErrAuthPolicyUnavailable (breaker open), got: %v", err)
	}
	if apiCalls.Load() != 2 {
		t.Fatalf("expected API calls to stop after breaker tripped; got %d", apiCalls.Load())
	}
}

func TestUpdateAuthPolicyHappyPath(t *testing.T) {
	var putCalls atomic.Int32

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/v1/b2b/organizations/org-1":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write(orgResponse(map[string]any{
				"auth_methods":           "RESTRICTED",
				"allowed_auth_methods":   []string{"magic_link"},
				"sso_active_connections": []map[string]any{{"connection_id": "conn-1"}},
			}))
		case r.Method == http.MethodPut && r.URL.Path == "/v1/b2b/organizations/org-1":
			putCalls.Add(1)
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Errorf("failed to decode update body: %v", err)
			}
			if got := body["email_jit_provisioning"]; got != "RESTRICTED" {
				t.Errorf("expected email_jit_provisioning=RESTRICTED, got %v", got)
			}
			domains, ok := body["email_allowed_domains"].([]any)
			if !ok || len(domains) != 1 || domains[0] != "acme.com" {
				t.Errorf("expected email_allowed_domains=[acme.com], got %v", body["email_allowed_domains"])
			}
			if got := body["auth_methods"]; got != "RESTRICTED" {
				t.Errorf("expected auth_methods=RESTRICTED (enforced-list mode), got %v", got)
			}
			allowed, ok := body["allowed_auth_methods"].([]any)
			if !ok || len(allowed) != 2 || allowed[0] != "magic_link" || allowed[1] != "email_otp" {
				t.Errorf("expected allowed_auth_methods=[magic_link email_otp], got %v", body["allowed_auth_methods"])
			}
			if got := body["sso_jit_provisioning"]; got != "NOT_ALLOWED" {
				t.Errorf("expected sso_jit_provisioning=NOT_ALLOWED, got %v", got)
			}
			if got := body["sso_default_connection_id"]; got != "conn-1" {
				t.Errorf("expected sso_default_connection_id=conn-1, got %v", got)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"organization_id":"org-1","status_code":200}`))
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	})

	updater, _ := newAuthPolicyUpdater(t, handler, 5)

	// Org owns conn-1 (GET returns it as active) so the default connection id
	// passes the ownership check.
	handlerForGet := http.HandlerFunc(handler.ServeHTTP)
	_ = handlerForGet

	err := updater.UpdateAuthPolicy(
		context.Background(),
		"org-1",
		domain.JitPolicyDomainRestricted,
		[]string{"acme.com"},
		[]domain.AllowedAuthMethod{domain.AuthMethodEmailOTP},
		domain.SsoJitPolicyDisabled,
		nil,
		"conn-1",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if putCalls.Load() != 1 {
		t.Fatalf("expected 1 update call, got %d", putCalls.Load())
	}
}

func TestUpdateAuthPolicyFirstWritePreservesEffectiveMethods(t *testing.T) {
	// An org still on auth_methods=ALL_ALLOWED with an active SSO connection:
	// the first write must persist RESTRICTED + the full effective method set
	// (magic_link, email_otp, google_oauth, microsoft_oauth, sso) + the
	// requested addition (email_otp is already in the preserved set → no dup).
	var putBody map[string]any

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/v1/b2b/organizations/org-1":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write(orgResponse(map[string]any{
				"auth_methods":           "ALL_ALLOWED",
				"sso_active_connections": []map[string]any{{"connection_id": "conn-1"}},
			}))
		case r.Method == http.MethodPut && r.URL.Path == "/v1/b2b/organizations/org-1":
			_ = json.NewDecoder(r.Body).Decode(&putBody)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"organization_id":"org-1","status_code":200}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})

	updater, _ := newAuthPolicyUpdater(t, handler, 5)

	err := updater.UpdateAuthPolicy(
		context.Background(),
		"org-1",
		domain.JitPolicyDomainRestricted,
		[]string{"acme.com"},
		[]domain.AllowedAuthMethod{domain.AuthMethodMagicLink},
		domain.SsoJitPolicyConnectionRestricted,
		[]string{"conn-1"},
		"",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got := putBody["auth_methods"]; got != "RESTRICTED" {
		t.Errorf("expected auth_methods=RESTRICTED, got %v", got)
	}
	allowed, ok := putBody["allowed_auth_methods"].([]any)
	if !ok {
		t.Fatalf("expected allowed_auth_methods list, got %v", putBody["allowed_auth_methods"])
	}
	want := []string{"magic_link", "email_otp", "google_oauth", "microsoft_oauth", "sso"}
	if len(allowed) != len(want) {
		t.Fatalf("expected preserved effective set %v, got %v", want, allowed)
	}
	for i, m := range allowed {
		if m != want[i] {
			t.Fatalf("expected %v, got %v", want, allowed)
		}
	}
	if got := putBody["sso_jit_provisioning"]; got != "RESTRICTED" {
		t.Errorf("expected sso_jit_provisioning=RESTRICTED, got %v", got)
	}
	conns, ok := putBody["sso_jit_provisioning_allowed_connections"].([]any)
	if !ok || len(conns) != 1 || conns[0] != "conn-1" {
		t.Errorf("expected sso_jit_provisioning_allowed_connections=[conn-1], got %v", putBody["sso_jit_provisioning_allowed_connections"])
	}
}

func TestUpdateAuthPolicyRestrictedOrgKeepsCurrentList(t *testing.T) {
	// An org already on RESTRICTED keeps its current allowlist; the requested
	// addition is unioned in (sso dropped — org has no active connections).
	var putBody map[string]any

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/v1/b2b/organizations/org-1":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write(orgResponse(map[string]any{
				"auth_methods":         "RESTRICTED",
				"allowed_auth_methods": []string{"magic_link"},
			}))
		case r.Method == http.MethodPut && r.URL.Path == "/v1/b2b/organizations/org-1":
			_ = json.NewDecoder(r.Body).Decode(&putBody)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"organization_id":"org-1","status_code":200}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})

	updater, _ := newAuthPolicyUpdater(t, handler, 5)

	err := updater.UpdateAuthPolicy(
		context.Background(),
		"org-1",
		domain.JitPolicyDisabled,
		nil,
		[]domain.AllowedAuthMethod{domain.AuthMethodEmailOTP},
		domain.SsoJitPolicyDisabled,
		nil,
		"",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	allowed, ok := putBody["allowed_auth_methods"].([]any)
	if !ok || len(allowed) != 2 || allowed[0] != "magic_link" || allowed[1] != "email_otp" {
		t.Fatalf("expected [magic_link email_otp], got %v", putBody["allowed_auth_methods"])
	}
}

func TestUpdateAuthPolicyStytch4xxSurfacesError(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write(orgResponse(map[string]any{"auth_methods": "ALL_ALLOWED"}))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write(stytchError(http.StatusBadRequest, "invalid_request"))
	})

	updater, _ := newAuthPolicyUpdater(t, handler, 5)

	err := updater.UpdateAuthPolicy(
		context.Background(),
		"org-1",
		domain.JitPolicyDisabled,
		nil,
		[]domain.AllowedAuthMethod{domain.AuthMethodMagicLink},
		domain.SsoJitPolicyDisabled,
		nil,
		"",
	)
	if err == nil {
		t.Fatal("expected 4xx to surface an error")
	}
	if errors.Is(err, domain.ErrAuthPolicyUnavailable) {
		t.Fatalf("4xx must not map to ErrAuthPolicyUnavailable, got: %v", err)
	}
	if !strings.Contains(err.Error(), "bad request") {
		t.Fatalf("expected wrapped Stytch error detail, got: %v", err)
	}
}

func TestUpdateAuthPolicyStytch5xxMapsUnavailable(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write(stytchError(http.StatusInternalServerError, "internal_server_error"))
	})

	updater, _ := newAuthPolicyUpdater(t, handler, 5)

	err := updater.UpdateAuthPolicy(
		context.Background(),
		"org-1",
		domain.JitPolicyDisabled,
		nil,
		[]domain.AllowedAuthMethod{domain.AuthMethodMagicLink},
		domain.SsoJitPolicyDisabled,
		nil,
		"",
	)
	if err == nil {
		t.Fatal("expected 5xx to surface an error")
	}
	if !errors.Is(err, domain.ErrAuthPolicyUnavailable) {
		t.Fatalf("expected ErrAuthPolicyUnavailable for 5xx, got: %v", err)
	}
}

func TestUpdateAuthPolicyBreakerOpenMapsUnavailable(t *testing.T) {
	var apiCalls atomic.Int32

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		apiCalls.Add(1)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write(stytchError(http.StatusInternalServerError, "internal_server_error"))
	})

	// Threshold 2: two failures trip the breaker open (the read counts).
	updater, _ := newAuthPolicyUpdater(t, handler, 2)

	ctx := context.Background()
	for i := 0; i < 2; i++ {
		if err := updater.UpdateAuthPolicy(
			ctx, "org-1",
			domain.JitPolicyDisabled, nil,
			[]domain.AllowedAuthMethod{domain.AuthMethodMagicLink},
			domain.SsoJitPolicyDisabled, nil, "",
		); err == nil {
			t.Fatal("expected error from failing update call")
		}
	}

	// Breaker is now open: the write fails fast WITHOUT reaching the API,
	// mapped to the 503 domain sentinel; the org policy is unchanged (no local
	// state is touched).
	err := updater.UpdateAuthPolicy(
		ctx, "org-1",
		domain.JitPolicyDisabled, nil,
		[]domain.AllowedAuthMethod{domain.AuthMethodMagicLink},
		domain.SsoJitPolicyDisabled, nil, "",
	)
	if err == nil {
		t.Fatal("expected circuit-open error")
	}
	if !errors.Is(err, domain.ErrAuthPolicyUnavailable) {
		t.Fatalf("expected ErrAuthPolicyUnavailable (breaker open), got: %v", err)
	}
	if apiCalls.Load() != 2 {
		t.Fatalf("expected API calls to stop after breaker tripped; got %d", apiCalls.Load())
	}
}

func TestUpdateAuthPolicyValidation(t *testing.T) {
	// GET returns an org with one active connection (conn-1) so ownership
	// checks exercise the happy path; the PUT must never be reached.
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(orgResponse(map[string]any{
			"auth_methods":           "ALL_ALLOWED",
			"sso_active_connections": []map[string]any{{"connection_id": "conn-1"}},
		}))
	})
	updater, _ := newAuthPolicyUpdater(t, handler, 5)

	ctx := context.Background()
	valid := func() error {
		return updater.UpdateAuthPolicy(
			ctx, "org-1",
			domain.JitPolicyDisabled, nil,
			[]domain.AllowedAuthMethod{domain.AuthMethodMagicLink},
			domain.SsoJitPolicyDisabled, nil, "",
		)
	}

	// Empty org ID short-circuits before any API call.
	if err := updater.UpdateAuthPolicy(
		ctx, "",
		domain.JitPolicyDisabled, nil,
		[]domain.AllowedAuthMethod{domain.AuthMethodMagicLink},
		domain.SsoJitPolicyDisabled, nil, "",
	); !errors.Is(err, domain.ErrAuthOrganizationIDRequired) {
		t.Fatalf("expected ErrAuthOrganizationIDRequired, got: %v", err)
	}

	// Invalid email JIT policy value.
	if err := updater.UpdateAuthPolicy(
		ctx, "org-1",
		domain.JitPolicy("MAYBE"), nil,
		[]domain.AllowedAuthMethod{domain.AuthMethodMagicLink},
		domain.SsoJitPolicyDisabled, nil, "",
	); !errors.Is(err, domain.ErrInvalidAuthPolicy) {
		t.Fatalf("expected ErrInvalidAuthPolicy, got: %v", err)
	}

	// Invalid SSO JIT policy value.
	if err := updater.UpdateAuthPolicy(
		ctx, "org-1",
		domain.JitPolicyDisabled, nil,
		[]domain.AllowedAuthMethod{domain.AuthMethodMagicLink},
		domain.SsoJitPolicy("MAYBE"), nil, "",
	); !errors.Is(err, domain.ErrInvalidAuthPolicy) {
		t.Fatalf("expected ErrInvalidAuthPolicy, got: %v", err)
	}

	// Domain-restricted email JIT requires a non-empty domain allowlist.
	if err := updater.UpdateAuthPolicy(
		ctx, "org-1",
		domain.JitPolicyDomainRestricted, nil,
		[]domain.AllowedAuthMethod{domain.AuthMethodMagicLink},
		domain.SsoJitPolicyDisabled, nil, "",
	); err == nil {
		t.Fatal("expected error for domain-restricted JIT without domains")
	}

	// At least one primary method.
	if err := updater.UpdateAuthPolicy(
		ctx, "org-1",
		domain.JitPolicyDisabled, nil, nil,
		domain.SsoJitPolicyDisabled, nil, "",
	); err == nil {
		t.Fatal("expected error for empty allowed auth methods")
	}

	// Unsupported allowed method.
	if err := updater.UpdateAuthPolicy(
		ctx, "org-1",
		domain.JitPolicyDisabled, nil,
		[]domain.AllowedAuthMethod{"passkeys"},
		domain.SsoJitPolicyDisabled, nil, "",
	); !errors.Is(err, domain.ErrInvalidAuthPolicy) {
		t.Fatalf("expected ErrInvalidAuthPolicy for unsupported method, got: %v", err)
	}

	// Connection-restricted SSO JIT requires allowed connections.
	if err := updater.UpdateAuthPolicy(
		ctx, "org-1",
		domain.JitPolicyDisabled, nil,
		[]domain.AllowedAuthMethod{domain.AuthMethodMagicLink},
		domain.SsoJitPolicyConnectionRestricted, nil, "",
	); err == nil {
		t.Fatal("expected error for connection-restricted SSO JIT without connections")
	}

	// sso_default_connection_id must reference an org-owned connection.
	if err := updater.UpdateAuthPolicy(
		ctx, "org-1",
		domain.JitPolicyDisabled, nil,
		[]domain.AllowedAuthMethod{domain.AuthMethodMagicLink},
		domain.SsoJitPolicyDisabled, nil, "conn-foreign",
	); !errors.Is(err, domain.ErrInvalidAuthPolicy) {
		t.Fatalf("expected ErrInvalidAuthPolicy for foreign default connection, got: %v", err)
	}

	// A valid update passes validation.
	if err := valid(); err != nil {
		t.Fatalf("expected valid update to pass, got: %v", err)
	}
}
