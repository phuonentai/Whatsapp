## Context

The Stytch integration has two near-identical code paths that handle Stytch communication:

- **Platform layer** (`internal/platform/stytch/`): thin `Client` wrapper, full `Config`, and an `RBACPolicyService` that returns `[]string` permissions. Has the `normalizeRoleID` bug.
- **Auth adapter layer** (`internal/modules/auth/adapters/stytch/`): its own separate `Config`, its own Stytch SDK client (`*b2bstytchapi.API`), and a slightly different `RBACPolicyService` that returns `[]auth.Permission`. Has the correct `normalizeRoleID`.

The auth module also ships a hardcoded `RBACService` (`auth/rbac.go`) with ~120 lines of role→permission maps, DTOs, and helper functions. This is consulted *first* on every request; the Stytch RBAC policy is only a fallback. RBAC API endpoints are completely unauthenticated.

Additionally, `app.env` carries 8 legacy auth keys from a pre-Stytch era that serve no purpose.

## Goals / Non-Goals

**Goals:**
- Single Stytch client instance shared across platform and auth modules
- Single `Config` struct used by all layers
- Fix `normalizeRoleID` bug in `platform/stytch/rbac_policy.go`
- Remove dead legacy config keys from `app.env`
- Align session durations between backend (1440 min) and frontend (480 min)
- Make Stytch RBAC the sole source of truth for authorization
- Require authentication on all RBAC API endpoints
- Unify duplicate Redis cache keys into one

**Non-Goals:**
- Adding webhook integration (separate change)
- Adding SSO/SAML/OIDC (separate change)
- Rewriting the frontend auth context or permission hook
- Changing the magic link authentication flow
- Adding JWT refresh tokens

## Decisions

### D1: Single Stytch Client — Platform Client as Singleton

**Decision:** The platform `*Client` wrapper becomes the single Stytch SDK instance. The auth adapter removes its own `*b2bstytchapi.API` creation and receives `*Client` via DI.

**Rationale:** The platform client is already initialized in the DI container (`stytchCmd.ProvideStytchDependencies()`). The auth adapter creates a separate connection for no reason — it increases connection count, duplicates configuration, and could lead to inconsistent state.

**Alternatives considered:**
- Keep both and document the split — rejected because it adds cognitive overhead for unclear benefit
- Move everything into the adapter — rejected because the platform layer needs its own access for org repositories

**Impact:**
- `auth/adapters/stytch/adapter.go`: constructor takes `*stytch.Client` instead of `*b2bstytchapi.API`
- `auth/adapters/stytch/token_verifier.go`: uses `client.API()` for Stytch API fallback calls
- `auth/adapters/stytch/rbac_policy.go`: constructor takes `*stytch.Client` instead of `*b2bstytchapi.API`
- `bootstrap/init_mods.go`: DI order already has platform client before auth init — no reordering needed

### D2: Config Consolidation — Keep Platform, Remove Adapter

**Decision:** Keep `platform/stytch/config.go`. Remove `auth/adapters/stytch/config.go`. Point auth module init at the platform config.

**Rationale:** The structs are functionally identical (same fields, same mapstructure tags, same default logic). The adapter version has only one extra method: `Validate()` (easily moved to platform) and `NewConfigFromExisting()` (a migration bridge — no longer needed).

**Impact:**
- Move `Validate()` method to platform config
- Remove `auth/adapters/stytch/config.go`
- Update `auth/cmd/init.go` to import `platform/stytch.Config` instead of its own

### D3: Stytch RBAC as Single Source of Truth

**Decision:** Remove all hardcoded role→permission maps from `auth/rbac.go` (the `RoleInfo` variables, `GetRoleInfo()`, `HasPermission()`, `AllRoles`, `AllPermissions`). Replace the `defaultRBACService` with a new `StytchRBACService` that reads everything from the Stytch RBAC policy.

**Implementation:**
1. Create a new `StytchRBACService` in `auth/adapters/stytch/` implementing the `RBACService` interface
2. It wraps the existing `RBACPolicyService` for `GetRolePermissions()` and derives all other methods from the cached policy structure
3. Keep the DTOs (`RoleDTO`, `PermissionDTO`, response types) and the `RBACService` interface in `auth/rbac.go` — only remove the hardcoded *data*
4. Keep `Permission`, `NewPermission()`, wildcard matching (`HasWildcard()`) — these are utility types, not hardcoded data

**Rationale:** Eliminates the dual-source problem. Role/permission changes in the Stytch dashboard take effect within 5 minutes (the Redis cache TTL) instead of requiring a deployment. The DTOs stay because they're the API contract — the *values* come from Stytch now.

**Alternatives considered:**
- Keep hardcoded as primary, Stytch as fallback — rejected because it makes Stytch dashboard changes invisible to the system
- Drop Stytch RBAC entirely, keep hardcoded — rejected because it defeats the purpose of using Stytch B2B

### D4: Auth-Protected RBAC Endpoints

**Decision:** Add `RequireAuth()` middleware to the RBAC route group. Any authenticated session is sufficient.

**Rationale:** Exposing the full role/permission structure to unauthenticated callers is unnecessary information disclosure. The frontend already has an authenticated session before it needs to call these endpoints (it uses them for UI rendering after login).

**Impact:**
- `routes.go`: add `resolver.Resolve("auth")` before the `/rbac` group or add it to each route
- Frontend: verify no RBAC endpoint is called before authentication in the user flow

### D5: Session Duration Alignment

**Decision:** Set backend and frontend to the same duration. The backend default is 1440 min (24h); change frontend from 480 min to match.

**Rationale:** Mismatched durations create inconsistent expiry behavior. 1440 min is a reasonable session lifetime for a B2B SaaS.

**Impact:**
- `next_b2b_starter/lib/auth/server-constants.ts`: change `getSessionDurationMinutes()` from 480 to 1440

### D6: Cache Key Unification

**Decision:** Use `"stytch:rbac:policy"` (the platform key) as the single Redis cache key for RBAC policy. Remove `"auth:stytch:rbac:policy"`.

**Rationale:** After client consolidation, both layers share the same Redis instance. Two different cache keys for the same data is wasteful.

### D7: normalizeRoleID Fix

**Decision:** Change `strings.TrimPrefix(roleID, "member")` to `strings.TrimPrefix(roleID, "stytch_")` in `platform/stytch/rbac_policy.go:221`.

### D8: Legacy Config Removal

**Decision:** Remove 8 keys from `app.env`: `ACCESS_TOKEN_DURATION`, `REFRESH_TOKEN_DURATION`, `TOKEN_SYMMETRIC_KEY`, `SESSION_ENCRYPTION_KEY`, `PASSWORD_HASH_COST`, `MAX_LOGIN_ATTEMPTS`, `LOCKOUT_DURATION`, `JWT_ISSUER`.

**Risk:** If any code still references these values (e.g., via viper or os.Getenv), it will break. A grep must confirm zero references before removal.

## Risks / Trade-offs

| Risk | Likelihood | Mitigation |
|------|-----------|------------|
| Stytch RBAC API is unreachable and cache is empty → RBAC resolution fails | Low | Redis cache with 5-min TTL covers most scenarios. Add a fallback returning cached data even if stale. |
| Frontend calls RBAC endpoints before auth → 401 error | Medium | Audit frontend bootstrap flow. The RBAC endpoints are called during `authBootstrap()` which runs after session validation. |
| Legacy config keys still referenced somewhere | Low | Grep the entire codebase before removal. |
| `StytchRBACService` has higher latency than hardcoded maps | Medium | All reads go through the Redis cache (sub-ms). Only the first request after cache expiry hits Stytch API (~200-500ms). |
| RBAC endpoint auth breaks the `/metadata` endpoint if called unauthenticated | Low | Metadata is used by frontend for display only — it's already behind auth in the normal flow. |

## Open Questions

1. **Frontend bootstrap timing:** Does the frontend ever call `GET /api/rbac/roles` before the user is authenticated? If so, adding auth to these endpoints will break the login page. **Resolution:** Audit `authBootstrap()` and `use-permissions.ts` to confirm they only run post-auth.
2. **Stytch RBAC policy format:** Are there custom resource/action definitions in the Stytch dashboard that don't match what the Go code expects? **Resolution:** Verify Stytch RBAC policy configuration before deploying the RBAC change.
