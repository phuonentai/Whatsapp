## ADDED Requirements

### Requirement: Permission resolution from Stytch RBAC policy

The system SHALL resolve all role-to-permission mappings exclusively from the Stytch RBAC policy API. Hardcoded role→permission maps in Go code MUST NOT be used as a source of truth for authorization decisions.

The Stytch RBAC policy MUST be cached in Redis with a 5-minute TTL. On cache hit, the cached policy MUST be used without calling the Stytch API. On cache miss, the policy MUST be fetched from Stytch, cached, and then used.

If the Stytch RBAC API is unreachable and the cache is empty, the system MUST return a 503 Service Unavailable error for any authorization check.

#### Scenario: Permission check with cached policy
- **WHEN** a request requires permission verification
- **AND** the Stytch RBAC policy is in the Redis cache
- **THEN** the system reads permissions from the cached policy
- **AND** the system does NOT call the Stytch API

#### Scenario: Permission check with cache miss
- **WHEN** a request requires permission verification
- **AND** the Stytch RBAC policy is NOT in the Redis cache
- **THEN** the system fetches the policy from the Stytch RBAC API
- **AND** caches it in Redis with a 5-minute TTL
- **AND** resolves the permission from the fetched policy

#### Scenario: Permission check with Stytch API unavailable
- **WHEN** a request requires permission verification
- **AND** the Stytch RBAC policy is NOT in the Redis cache
- **AND** the Stytch RBAC API is unreachable
- **THEN** the system returns a 503 Service Unavailable error

#### Scenario: Stytch dashboard change propagation
- **WHEN** an admin updates role permissions in the Stytch dashboard
- **THEN** the updated permissions MUST take effect within 5 minutes (the Redis cache TTL)
- **AND** no deployment or code change is required

#### Scenario: Wildcard permission expansion
- **WHEN** a Stytch RBAC policy role has a permission with wildcard action (`*`)
- **THEN** the system expands the wildcard to all actions defined for that resource in the Stytch policy
- **AND** each expanded action produces a separate `resource:action` permission entry

### Requirement: Role normalization

The system SHALL normalize Stytch role IDs by removing the `stytch_` prefix. For example, `stytch_member` SHALL be normalized to `member`, `stytch_admin` to `admin`.

The normalization MUST use `strings.TrimPrefix(roleID, "stytch_")` — NOT string matching on partial substrings like `"member"`.

#### Scenario: Standard role normalization
- **WHEN** a role ID `stytch_member` is received
- **THEN** it is normalized to `member`

#### Scenario: Role without prefix passes through
- **WHEN** a role ID `custom_role` is received
- **THEN** it passes through unchanged as `custom_role`

#### Scenario: Whitespace handling
- **WHEN** a role ID has leading or trailing whitespace
- **THEN** the whitespace is trimmed before prefix removal

### Requirement: RBAC API endpoint authentication

All RBAC API endpoints (roles, permissions, by-category, role details, check-permission, metadata) SHALL require a valid authenticated session. Unauthenticated requests MUST receive a 401 Unauthorized response.

#### Scenario: Authenticated RBAC request
- **WHEN** an authenticated user sends a GET request to `/api/rbac/roles`
- **THEN** the system returns a 200 response with the roles and their permissions

#### Scenario: Unauthenticated RBAC request
- **WHEN** an unauthenticated user sends a GET request to `/api/rbac/roles`
- **THEN** the system returns a 401 Unauthorized response

### Requirement: RBACService implementation backed by Stytch policy

The `RBACService` interface SHALL have a single implementation (`StytchRBACService`) that derives all methods from the Stytch RBAC policy. The following methods MUST be supported:

| Method | Derivation |
|--------|-----------|
| `GetAllRoles()` | All roles from policy, each with `RoleInfo` including normalized ID and permissions |
| `GetRoleInfo(roleID)` | Single role lookup by normalized ID |
| `GetAllPermissions()` | All unique `resource:action` pairs from all role definitions |
| `GetRolePermissions(roleID)` | Delegates to `RBACPolicyService.GetRolePermissions()` |
| `GetPermissionsByCategory()` | Permissions grouped by resource (category = resource name) |
| `GetPermissionsByRoleID(roleID)` | String IDs of all permissions for a role |
| `HasPermission(roleID, permissionId)` | True if any role permission matches the permission ID |
| `GetRBACMetadata()` | Counts derived from policy (total roles, total permissions, per-role counts) |

#### Scenario: GetAllRoles returns roles from policy
- **WHEN** `GetAllRoles()` is called
- **THEN** it returns all roles defined in the Stytch RBAC policy
- **AND** each role has a normalized ID, display name, and resolved permissions

#### Scenario: GetRoleInfo for existing role
- **WHEN** `GetRoleInfo("admin")` is called
- **AND** the Stytch policy defines an `admin` role
- **THEN** it returns the `RoleInfo` with the correct permissions

#### Scenario: GetRoleInfo for non-existent role
- **WHEN** `GetRoleInfo("nonexistent_role")` is called
- **AND** the Stytch policy does NOT define this role
- **THEN** it returns nil

### Requirement: DTOs retained as API contract

The `RoleDTO`, `PermissionDTO`, `RolesResponse`, `PermissionsResponse`, and other API response types in `auth/rbac.go` SHALL be retained. Only the hardcoded *data* (`RoleInfo` variables, `AllRoles`, `AllPermissions`, `GetRoleInfo()`, `HasPermission()`) SHALL be removed.

#### Scenario: API response format unchanged
- **WHEN** a client calls `GET /api/rbac/roles`
- **THEN** the JSON response format SHALL be identical to the pre-change format
- **AND** the values (role names, permission lists) SHALL come from Stytch RBAC policy, not hardcoded constants
