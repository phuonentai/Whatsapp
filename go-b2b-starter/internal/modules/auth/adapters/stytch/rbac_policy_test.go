package stytch

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/moasq/go-b2b-starter/internal/platform/logger"
	platformstytch "github.com/moasq/go-b2b-starter/internal/platform/stytch"
	"github.com/stytchauth/stytch-go/v18/stytch/b2b/rbac"
)

// noopRedis implements redis.Client with no storage: the policy cache always
// misses so every GetAllRoles call hits the fetch path (or the breaker).
type noopRedis struct{}

func (noopRedis) Set(ctx context.Context, key string, value any, ttl time.Duration) error {
	return nil
}
func (noopRedis) Get(ctx context.Context, key string) (string, error) { return "", nil }
func (noopRedis) Delete(ctx context.Context, key string) error         { return nil }
func (noopRedis) Exists(ctx context.Context, key string) (bool, error) { return false, nil }
func (noopRedis) Ping(ctx context.Context) error                       { return nil }

// cachedRedis serves a fixed JSON-encoded policy from Get (cache-hit path),
// so tests exercise GetAllRoles mapping without network access.
type cachedRedis struct {
	payload string
}

func (c *cachedRedis) Set(ctx context.Context, key string, value any, ttl time.Duration) error {
	return nil
}
func (c *cachedRedis) Get(ctx context.Context, key string) (string, error) {
	return c.payload, nil
}
func (c *cachedRedis) Delete(ctx context.Context, key string) error         { return nil }
func (c *cachedRedis) Exists(ctx context.Context, key string) (bool, error) { return true, nil }
func (c *cachedRedis) Ping(ctx context.Context) error                       { return nil }

// TestGetAllRolesBreakerOpenReturnsEmpty simulates a Stytch RBAC policy API
// outage: the policy endpoint returns 500, consecutive failures trip the
// two-tier circuit breaker, and GetAllRoles must degrade to an empty list
// (policy unavailable) — never a hard error that would surface as a 500 to the
// UI. The 4th call is blocked by the open breaker without hitting the API.
func TestGetAllRolesBreakerOpenReturnsEmpty(t *testing.T) {
	var policyHits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// The SDK fetches JWKS at client construction; serve an empty key set.
		if strings.Contains(r.URL.Path, "/sessions/jwks/") {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"keys": []}`)
			return
		}
		policyHits.Add(1)
		http.Error(w, "stytch API unavailable", http.StatusInternalServerError)
	}))
	defer srv.Close()

	cfg := platformstytch.Config{
		ProjectID:                    "test-project-id",
		Secret:                       "test-secret",
		BaseURL:                      srv.URL,
		APITimeout:                   2 * time.Second,
		CircuitBreakerThreshold:      3,
		CircuitBreakerTimeout:        10 * time.Second,
		CircuitBreakerHalfOpenProbes: 2,
	}

	client, err := platformstytch.NewClient(cfg)
	if err != nil {
		t.Fatalf("failed to create breaker client: %v", err)
	}

	policyService := NewRBACPolicyServiceWithBreaker(
		client.API(),
		client,
		noopRedis{},
		logger.New(logger.WithLevel(logger.FatalLevel)), // silence expected error logs
	)
	service := NewStytchRBACService(policyService)

	// Threshold is 3: iterations 1-3 fail at the API (500), iteration 4 is
	// blocked by the open breaker. Every call must return an empty role list
	// (no error propagates out of GetAllRoles).
	for i := 0; i < 4; i++ {
		roles := service.GetAllRoles()
		if len(roles) != 0 {
			t.Fatalf("iteration %d: expected empty roles (policy unavailable), got %d roles", i+1, len(roles))
		}
	}

	if state := client.BreakerState(); state != platformstytch.CircuitOpen {
		t.Fatalf("expected circuit breaker to be open after repeated failures, got %v", state)
	}
	if hits := policyHits.Load(); hits != 3 {
		t.Errorf("expected exactly 3 policy API hits before the breaker opened, got %d", hits)
	}
}

// TestGetAllRolesPropagatesPolicyDescription ensures the Stytch policy role
// description reaches RoleInfo.Description (single source of truth) and that
// wildcard permissions expand to the resource's declared actions.
func TestGetAllRolesPropagatesPolicyDescription(t *testing.T) {
	policy := rbac.Policy{
		Resources: []rbac.PolicyResource{
			{ResourceID: "contact", Actions: []string{"view", "create", "edit"}},
		},
		Roles: []rbac.PolicyRole{
			{
				RoleID:      "stytch_admin",
				Description: "Control total de la organización.",
				Permissions: []rbac.PolicyRolePermission{
					{ResourceID: "contact", Actions: []string{"*"}},
				},
			},
		},
	}
	raw, err := json.Marshal(policy)
	if err != nil {
		t.Fatalf("failed to marshal policy: %v", err)
	}

	policyService := NewRBACPolicyService(
		nil, // raw client unused on the cache-hit path
		&cachedRedis{payload: string(raw)},
		logger.New(logger.WithLevel(logger.FatalLevel)),
	)
	service := NewStytchRBACService(policyService)

	roles := service.GetAllRoles()
	if len(roles) != 1 {
		t.Fatalf("expected 1 role from policy, got %d", len(roles))
	}

	role := roles[0]
	if role.ID != "admin" {
		t.Errorf("expected normalized role ID 'admin', got %q", role.ID)
	}
	if role.Description != "Control total de la organización." {
		t.Errorf("expected policy description to propagate, got %q", role.Description)
	}

	// Wildcard expansion: contact:* → contact:view, contact:create, contact:edit
	expected := []string{"contact:view", "contact:create", "contact:edit"}
	if len(role.Permissions) != len(expected) {
		t.Fatalf("expected %d expanded permissions, got %d: %v", len(expected), len(role.Permissions), role.Permissions)
	}
	for i, perm := range role.Permissions {
		if string(perm) != expected[i] {
			t.Errorf("permission %d: expected %q, got %q", i, expected[i], string(perm))
		}
	}
}
