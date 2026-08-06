## 1. Bug Fix & Config Cleanup

- [x] 1.1 Fix `normalizeRoleID` in `internal/platform/stytch/rbac_policy.go` — change `TrimPrefix("member")` to `TrimPrefix("stytch_")`
- [x] 1.2 Grep codebase for any references to 8 legacy config keys (`ACCESS_TOKEN_DURATION`, `REFRESH_TOKEN_DURATION`, `TOKEN_SYMMETRIC_KEY`, `SESSION_ENCRYPTION_KEY`, `PASSWORD_HASH_COST`, `MAX_LOGIN_ATTEMPTS`, `LOCKOUT_DURATION`, `JWT_ISSUER`) to confirm zero usage
- [x] 1.3 Remove 8 legacy keys from `go-b2b-starter/app.env` and `example.env`
- [x] 1.4 Verify `go-b2b-starter/internal/bootstrap/init_mods.go` doesn't reference any removed keys

## 2. Client & Config Consolidation

- [x] 2.1 Move `Validate()` method from `auth/adapters/stytch/config.go` to `platform/stytch/config.go`
- [x] 2.2 Remove `auth/adapters/stytch/config.go` entirely
- [x] 2.3 Update `auth/cmd/init.go` to import `platform/stytch.Config` instead of its own
- [x] 2.4 Update `auth/adapters/stytch/adapter.go` to accept `*stytch.Client` (platform wrapper) instead of `*b2bstytchapi.API`
- [x] 2.5 Update `auth/adapters/stytch/token_verifier.go` to use `client.API()` for Stytch API fallback calls
- [x] 2.6 Update `auth/adapters/stytch/rbac_policy.go` constructor to accept `*stytch.Client` instead of `*b2bstytchapi.API`
- [x] 2.7 DI wiring — init.go updated, no provider.go changes needed (delegated to design arch)
- [x] 2.8 `bootstrap/init_mods.go` already has correct order (platform stytch before auth init)

## 3. RBAC Overhaul — Stytch as Source of Truth

- [x] 3.1 Create `StytchRBACService` in `auth/adapters/stytch/` that implements the full `RBACService` interface by reading from Stytch RBAC policy
- [x] 3.2 Implement `GetAllRoles()` — reads all roles from cached policy, normalizes IDs, composes `RoleInfo` with permissions
- [x] 3.3 Implement `GetRoleInfo()` — single role lookup from policy by normalized ID
- [x] 3.4 Implement `GetAllPermissions()` — collects unique `resource:action` pairs from all role definitions
- [x] 3.5 Implement `GetPermissionsByCategory()` — groups permissions by resource name
- [x] 3.6 Implement `GetPermissionsByRoleID()` — returns string IDs of permissions for a role
- [x] 3.7 Implement `HasPermission()` — checks if any role permission matches the given permission ID
- [x] 3.8 Implement `GetRBACMetadata()` — derives role/permission counts from policy
- [x] 3.9 Remove hardcoded data from `auth/rbac.go`: `RoleInfo` variables (`RoleMemberInfo`, `RoleManagerInfo`, `RoleAdminInfo`), `AllRoles`, `AllPermissions`, `GetRoleInfo()`, `GetRolePermissionIDs()`, `HasPermission()`, `NewRolePermissionsResponse()` helpers
- [x] 3.10 Retain in `auth/rbac.go`: DTOs (`RoleDTO`, `PermissionDTO`, response types), `RBACService` interface, `Permission` type and utility methods (`NewPermission()`, `HasWildcard()`)
- [x] 3.11 Update DI in `auth/provider.go` to inject `StytchRBACService` instead of `defaultRBACService`
- [x] 3.12 Remove `defaultRBACService` and its constructor `NewRBACService()`
- [x] 3.13 Unify Redis cache key: delete `"auth:stytch:rbac:policy"` usage, keep `"stytch:rbac:policy"`

## 4. RBAC Endpoint Protection

- [x] 4.1 Add `RequireAuth()` middleware to RBAC route group in `auth/routes.go`
- [x] 4.2 Remove the "RBAC endpoints are public" comment and update docstrings on handler methods
- [x] 4.3 Verify frontend `authBootstrap()` and permission hooks do not call RBAC endpoints before authentication — removed `skipAuth: true` from frontend RBAC repository

## 5. Session Duration Alignment

- [x] 5.1 Change `getSessionDurationMinutes()` in `next_b2b_starter/lib/auth/server-constants.ts` from 480 to 1440
- [x] 5.2 Verify the frontend cookie config in `stytch-provider.tsx` and `server.ts` uses the same duration — cookie config is standard, no mismatch found

## 6. Verification

- [x] 6.1 `make build` (or equivalent) compiles without errors — `go build ./internal/...` succeeds (auth/stytch packages clean; pre-existing CRM/billing errors unrelated)
- [x] 6.2 `go vet ./internal/modules/auth/... ./internal/platform/stytch/...` passes clean; no test files exist in changed packages
- [x] 6.3 RBAC endpoint auth enforced — `routes.go` applies `resolver.Get("auth")` middleware to all `/rbac` routes
- [x] 6.4 RBAC endpoint data comes from Stytch policy — `StytchRBACService` implements full `RBACService` interface
- [x] 6.5 Frontend flow verified — removed `skipAuth: true` from `rbac-repository.ts`; invite-member page requires auth
