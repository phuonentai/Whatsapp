package stytch

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/moasq/go-b2b-starter/internal/modules/auth"
	"github.com/moasq/go-b2b-starter/internal/platform/logger"
	"github.com/moasq/go-b2b-starter/internal/platform/redis"
	platformstytch "github.com/moasq/go-b2b-starter/internal/platform/stytch"
	"github.com/stytchauth/stytch-go/v18/stytch/b2b/b2bstytchapi"
	"github.com/stytchauth/stytch-go/v18/stytch/b2b/rbac"
)

const (
	// Redis cache key for RBAC policy.
	// Versioned: v2 when the `export` action was added; v3 when the inbox
	// scope permissions (`inbox:view_all`, `inbox:view_unassigned`,
	// `inbox:reassign`) were added (conversation-row-scoping) so they apply
	// without waiting for the previous cache TTL to expire.
	rbacPolicyCacheKey = "auth:stytch:rbac:policy:v3"
	// Cache TTL matches Stytch SDK default (5 minutes)
	rbacPolicyCacheTTL = 5 * time.Minute
)

// RBACPolicyService fetches and caches the Stytch RBAC policy.
//
// It retrieves the role-permission mappings from Stytch and caches them
// in Redis to avoid API calls on every request.
//
// The policy fetch SHALL pass through the two-tier circuit breaker
// (`platform/stytch.Client.Run`: threshold 5, timeout 10s, half-open 2) when a
// breaker client is provided — the consolidated service served via DI always
// provides one, so a Stytch outage degrades to cached policy (or empty with a
// log) instead of a hard failure per request.
type RBACPolicyService struct {
	client  *b2bstytchapi.API // raw Stytch B2B API client (policy endpoint)
	breaker *platformstytch.Client // optional breaker wrapper; when set, fetches run via Client.Run
	redis   redis.Client
	logger  logger.Logger
}

// NewRBACPolicyService creates a policy service whose fetch is NOT guarded by
// the circuit breaker. Prefer NewRBACPolicyServiceWithBreaker for any
// production-served instance.
func NewRBACPolicyService(client *b2bstytchapi.API, redisClient redis.Client, logger logger.Logger) *RBACPolicyService {
	return &RBACPolicyService{
		client: client,
		redis:  redisClient,
		logger: logger,
	}
}

// NewRBACPolicyServiceWithBreaker creates a policy service whose policy fetch
// is guarded by the shared two-tier circuit breaker (`Client.Run`).
func NewRBACPolicyServiceWithBreaker(client *b2bstytchapi.API, breaker *platformstytch.Client, redisClient redis.Client, logger logger.Logger) *RBACPolicyService {
	return &RBACPolicyService{
		client:  client,
		breaker: breaker,
		redis:   redisClient,
		logger:  logger,
	}
}

// GetRolePermissions returns all permissions for a given role from Stytch RBAC policy.
//
// Returns permissions in "resource:action" format (e.g., "invoice:create").
func (s *RBACPolicyService) GetRolePermissions(ctx context.Context, roleID string) ([]auth.Permission, error) {
	// Normalize role ID
	normalizedRoleID := normalizeRoleID(roleID)

	// Get policy from cache or Stytch
	policy, err := s.getPolicy(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get RBAC policy: %w", err)
	}

	// Find role in policy
	for _, role := range policy.Roles {
		if strings.EqualFold(role.RoleID, normalizedRoleID) {
			return s.convertPermissions(role.Permissions, policy), nil
		}
	}

	// Role not found in policy
	s.logger.Debug("role not found in Stytch RBAC policy", logger.Fields{
		"role_id":    roleID,
		"normalized": normalizedRoleID,
	})
	return nil, nil
}

// getPolicy fetches policy from Redis cache or Stytch API.
func (s *RBACPolicyService) getPolicy(ctx context.Context) (*rbac.Policy, error) {
	// Try cache first
	cached, err := s.redis.Get(ctx, rbacPolicyCacheKey)
	if err == nil && cached != "" {
		var policy rbac.Policy
		if unmarshalErr := json.Unmarshal([]byte(cached), &policy); unmarshalErr == nil {
			s.logger.Debug("RBAC policy fetched from cache", logger.Fields{})
			return &policy, nil
		} else {
			s.logger.Warn("failed to unmarshal cached RBAC policy", logger.Fields{
				"error": unmarshalErr.Error(),
			})
		}
	}

	// Cache miss - fetch from Stytch
	policy, err := s.fetchPolicyFromStytch(ctx)
	if err != nil {
		return nil, err
	}

	// Cache the policy
	s.cachePolicy(ctx, policy)

	return policy, nil
}

// fetchPolicyFromStytch fetches RBAC policy from Stytch API.
//
// When a breaker client is wired, the call runs through the two-tier circuit
// breaker (`Client.Run`); a breaker-open failure returns ErrCircuitOpen and is
// treated as policy unavailable (logged; cache is preferred, otherwise empty).
func (s *RBACPolicyService) fetchPolicyFromStytch(ctx context.Context) (*rbac.Policy, error) {
	s.logger.Info("fetching RBAC policy from Stytch", logger.Fields{})

	var policy *rbac.Policy

	fetch := func() error {
		resp, err := s.client.RBAC.Policy(ctx, &rbac.PolicyParams{})
		if err != nil {
			return fmt.Errorf("stytch RBAC policy API call failed: %w", err)
		}

		if resp.Policy == nil {
			return fmt.Errorf("stytch returned empty policy")
		}

		policy = resp.Policy
		return nil
	}

	if s.breaker != nil {
		if err := s.breaker.Run(ctx, fetch); err != nil {
			s.logPolicyFetchFailure(err)
			return nil, err
		}
	} else {
		if err := fetch(); err != nil {
			s.logPolicyFetchFailure(err)
			return nil, err
		}
	}

	s.logger.Info("successfully fetched RBAC policy", logger.Fields{
		"roles_count":     len(policy.Roles),
		"resources_count": len(policy.Resources),
	})

	return policy, nil
}

// logPolicyFetchFailure logs a policy fetch failure. Breaker-open failures are
// logged distinctly so operators can tell "blocked locally" from "API error".
func (s *RBACPolicyService) logPolicyFetchFailure(err error) {
	if errors.Is(err, platformstytch.ErrCircuitOpen) {
		s.logger.Warn("RBAC policy fetch blocked by circuit breaker (policy unavailable)", logger.Fields{
			"error": err.Error(),
		})
		return
	}
	s.logger.Error("failed to fetch RBAC policy from Stytch (policy unavailable)", logger.Fields{
		"error": err.Error(),
	})
}

// cachePolicy stores policy in Redis.
func (s *RBACPolicyService) cachePolicy(ctx context.Context, policy *rbac.Policy) {
	data, err := json.Marshal(policy)
	if err != nil {
		s.logger.Warn("failed to marshal RBAC policy for caching", logger.Fields{
			"error": err.Error(),
		})
		return
	}

	if err := s.redis.Set(ctx, rbacPolicyCacheKey, string(data), rbacPolicyCacheTTL); err != nil {
		s.logger.Warn("failed to cache RBAC policy in Redis", logger.Fields{
			"error": err.Error(),
		})
	}
}

// convertPermissions converts Stytch permission format to auth.Permission slice.
//
// Handles wildcard expansion:
//
//	Input: []PolicyRolePermission{{ResourceID: "invoice", Actions: ["view", "create"]}}
//	Output: [Permission("invoice:view"), Permission("invoice:create")]
func (s *RBACPolicyService) convertPermissions(permissions []rbac.PolicyRolePermission, policy *rbac.Policy) []auth.Permission {
	if len(permissions) == 0 {
		return nil
	}

	result := make([]auth.Permission, 0, len(permissions)*5)

	for _, perm := range permissions {
		resourceID := strings.ToLower(perm.ResourceID)
		if resourceID == "" {
			continue
		}

		// Expand wildcard actions
		expandedActions := s.expandWildcardActions(perm.ResourceID, perm.Actions, policy)

		// Convert each action to Permission
		for _, action := range expandedActions {
			if action == "" {
				continue
			}
			result = append(result, auth.NewPermission(resourceID, strings.ToLower(action)))
		}
	}

	return result
}

// expandWildcardActions expands wildcard (*) to all resource actions from policy.
func (s *RBACPolicyService) expandWildcardActions(resourceID string, actions []string, policy *rbac.Policy) []string {
	// Check if actions contain wildcard
	hasWildcard := false
	for _, action := range actions {
		if action == "*" {
			hasWildcard = true
			break
		}
	}

	if !hasWildcard {
		return actions
	}

	// Find resource definition to get all actions
	for _, resource := range policy.Resources {
		if strings.EqualFold(resource.ResourceID, resourceID) {
			if len(resource.Actions) > 0 {
				s.logger.Debug("expanded wildcard permission", logger.Fields{
					"resource":      resourceID,
					"actions_count": len(resource.Actions),
				})
				return resource.Actions
			}
		}
	}

	// Resource not found in policy, keep wildcard as-is
	// (documented residual: UI shows the literal wildcard with a
	// "broad permission" note rather than inventing an expansion).
	return actions
}

// normalizeRoleID removes common prefixes from role IDs.
func normalizeRoleID(roleID string) string {
	roleID = strings.TrimSpace(roleID)
	roleID = strings.TrimPrefix(roleID, "stytch_")
	return roleID
}
