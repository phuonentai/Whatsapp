package stytch

import (
	"context"
	"fmt"
	"strings"

	"github.com/moasq/go-b2b-starter/internal/modules/auth"
	"github.com/stytchauth/stytch-go/v18/stytch/b2b/rbac"
)

// StytchRBACService implements auth.RBACService backed by the Stytch RBAC policy.
//
// All role-permission mappings come exclusively from the Stytch RBAC policy,
// cached in Redis with a 5-minute TTL. No hardcoded role→permission maps are used.
type StytchRBACService struct {
	policyService *RBACPolicyService
}

// ensure StytchRBACService implements auth.RBACService
var _ auth.RBACService = (*StytchRBACService)(nil)

func NewStytchRBACService(policyService *RBACPolicyService) *StytchRBACService {
	return &StytchRBACService{
		policyService: policyService,
	}
}

// getPolicy fetches the Stytch RBAC policy (from cache or API).
func (s *StytchRBACService) getPolicy() (*rbac.Policy, error) {
	return s.policyService.getPolicy(context.Background())
}

// GetAllRoles returns all roles from the Stytch RBAC policy.
func (s *StytchRBACService) GetAllRoles() []auth.RoleInfo {
	policy, err := s.getPolicy()
	if err != nil || policy == nil {
		return nil
	}

	roles := make([]auth.RoleInfo, 0, len(policy.Roles))
	for _, role := range policy.Roles {
		permissions := s.policyService.convertPermissions(role.Permissions, policy)
		roles = append(roles, auth.RoleInfo{
			ID:   normalizeRoleID(role.RoleID),
			Name: roleName(role.RoleID),
			Permissions: permissions,
		})
	}
	return roles
}

// GetRoleInfo returns role information by normalized role ID.
func (s *StytchRBACService) GetRoleInfo(roleID string) *auth.RoleInfo {
	normalized := normalizeRoleID(roleID)

	policy, err := s.getPolicy()
	if err != nil || policy == nil {
		return nil
	}

	for _, role := range policy.Roles {
		if strings.EqualFold(normalizeRoleID(role.RoleID), normalized) {
			permissions := s.policyService.convertPermissions(role.Permissions, policy)
			return &auth.RoleInfo{
				ID:   normalized,
				Name: roleName(role.RoleID),
				Permissions: permissions,
			}
		}
	}
	return nil
}

// GetAllPermissions returns all unique permissions from all role definitions.
func (s *StytchRBACService) GetAllPermissions() []auth.Permission {
	policy, err := s.getPolicy()
	if err != nil || policy == nil {
		return nil
	}

	seen := make(map[auth.Permission]struct{})
	for _, role := range policy.Roles {
		perms := s.policyService.convertPermissions(role.Permissions, policy)
		for _, p := range perms {
			seen[p] = struct{}{}
		}
	}

	result := make([]auth.Permission, 0, len(seen))
	for p := range seen {
		result = append(result, p)
	}
	return result
}

// GetRolePermissions returns permissions for a given role, delegating to RBACPolicyService.
func (s *StytchRBACService) GetRolePermissions(roleID string) []auth.Permission {
	perms, err := s.policyService.GetRolePermissions(context.Background(), roleID)
	if err != nil {
		return nil
	}
	return perms
}

// GetPermissionsByCategory groups permissions by their resource (category).
func (s *StytchRBACService) GetPermissionsByCategory() map[string][]auth.Permission {
	policy, err := s.getPolicy()
	if err != nil || policy == nil {
		return nil
	}

	categories := make(map[string][]auth.Permission)
	for _, resource := range policy.Resources {
		resourceID := strings.ToLower(resource.ResourceID)
		perms := make([]auth.Permission, 0, len(resource.Actions))
		for _, action := range resource.Actions {
			perms = append(perms, auth.NewPermission(resourceID, strings.ToLower(action)))
		}
		if len(perms) > 0 {
			categories[resourceID] = perms
		}
	}

	// Fallback: if no resources defined, derive from roles
	if len(categories) == 0 {
		for _, role := range policy.Roles {
			for _, perm := range role.Permissions {
				resourceID := strings.ToLower(perm.ResourceID)
				resourceActions := categories[resourceID]
				if resourceActions == nil {
					resourceActions = make([]auth.Permission, 0)
				}
				for _, action := range perm.Actions {
					if action == "*" {
						continue
					}
					p := auth.NewPermission(resourceID, strings.ToLower(action))
					found := false
					for _, existing := range resourceActions {
						if existing == p {
							found = true
							break
						}
					}
					if !found {
						resourceActions = append(resourceActions, p)
					}
				}
				categories[resourceID] = resourceActions
			}
		}
	}

	return categories
}

// GetPermissionsByRoleID returns permission string IDs for a given role.
func (s *StytchRBACService) GetPermissionsByRoleID(roleID string) []string {
	perms := s.GetRolePermissions(roleID)
	if len(perms) == 0 {
		return nil
	}
	ids := make([]string, len(perms))
	for i, p := range perms {
		ids[i] = string(p)
	}
	return ids
}

// HasPermission checks if a role has a specific permission by ID.
func (s *StytchRBACService) HasPermission(roleID string, permissionID string) bool {
	perms := s.GetRolePermissions(roleID)
	target := auth.Permission(permissionID)
	for _, p := range perms {
		if p == target {
			return true
		}
	}
	return false
}

// GetRBACMetadata derives role/permission counts from the Stytch policy.
func (s *StytchRBACService) GetRBACMetadata() auth.RBACMetadata {
	policy, err := s.getPolicy()
	if err != nil || policy == nil {
		return auth.RBACMetadata{}
	}

	allPerms := s.GetAllPermissions()
	permsByRole := make(map[string]int)
	for _, role := range policy.Roles {
		normalizedID := normalizeRoleID(role.RoleID)
		permsByRole[normalizedID] = len(role.Permissions)
	}

	return auth.RBACMetadata{
		TotalRoles:        len(policy.Roles),
		TotalPermissions:  len(allPerms),
		PermissionsByRole: permsByRole,
		Description:       fmt.Sprintf("RBAC policy with %d roles defined in Stytch", len(policy.Roles)),
	}
}

// roleName generates a display name from a role ID.
func roleName(roleID string) string {
	normalized := normalizeRoleID(roleID)
	if len(normalized) == 0 {
		return ""
	}
	return strings.ToUpper(normalized[:1]) + normalized[1:]
}
