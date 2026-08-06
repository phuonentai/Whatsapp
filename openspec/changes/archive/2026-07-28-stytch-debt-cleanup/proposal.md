## Why

The Stytch integration has accumulated technical debt from a partial migration: a `normalizeRoleID` bug that silently corrupts role data, duplicate Stytch client instances, dead legacy auth config, two near-identical Config structs, and a dual RBAC system where hardcoded Go maps and Stytch's RBAC policy compete as sources of truth. Meanwhile, Stytch B2B's core features remain unused, and the absence of webhooks means local DB state drifts from Stytch whenever roles or memberships change externally.

This change cleans up the debt, consolidates the integration, and makes Stytch RBAC the sole authorization authority so permission changes in the Stytch dashboard take effect without deployments.

## What Changes

- **Fix `normalizeRoleID` bug** — `TrimPrefix("member")` → `TrimPrefix("stytch_")` in `platform/stytch/rbac_policy.go`
- **Consolidate to a single Stytch client** — remove the duplicate client in the auth adapter; inject the platform client instead
- **Consolidate Config structs** — merge `platform/stytch/config.go` and `auth/adapters/stytch/config.go` into one
- **Remove dead legacy auth config** — `ACCESS_TOKEN_DURATION`, `REFRESH_TOKEN_DURATION`, `TOKEN_SYMMETRIC_KEY`, `SESSION_ENCRYPTION_KEY`, `PASSWORD_HASH_COST`, `MAX_LOGIN_ATTEMPTS`, `LOCKOUT_DURATION`, `JWT_ISSUER` from `app.env`
- **Align session durations** — fix frontend/backend mismatch (1440 min vs 480 min)
- **Replace hardcoded RBAC maps with Stytch RBAC policy** — remove `auth/rbac.go` role->permission constants; read all permissions from Stytch RBAC API (Redis-cached)
- **Require auth on RBAC API endpoints** — `GET /api/rbac/roles`, `GET /api/rbac/permissions` etc. require a valid session
- **Document the full Stytch API surface** — endpoints used, token format, session strategy, webhook event model

## Capabilities

### New Capabilities

- `stytch-authorization`: Stytch RBAC as single source of truth — permission resolution, role normalization, protected RBAC API endpoints

### Modified Capabilities

_(None — no existing specs cover authentication or authorization)_

## Impact

**Backend (`go-b2b-starter/`):**
- `internal/platform/stytch/rbac_policy.go` — bug fix
- `internal/platform/stytch/config.go` + `internal/modules/auth/adapters/stytch/config.go` — merge
- `internal/platform/stytch/client.go` + `internal/modules/auth/adapters/stytch/adapter.go` — single client
- `internal/modules/auth/rbac.go` — remove hardcoded role->permission maps
- `internal/modules/auth/rbac.go` — remove fallback in `HasRolePermission`
- `internal/modules/auth/middleware.go` — simplify `hasPermission()` (no fallback)
- `internal/modules/auth/handler.go` + `internal/modules/auth/routes.go` — auth on RBAC endpoints
- `internal/modules/auth/adapters/stytch/rbac_policy.go` — promote to primary (was fallback)
- `internal/modules/auth/adapters/stytch/token_verifier.go` — update to use shared client
- `internal/bootstrap/init_mods.go` — adjust DI order after client consolidation
- `app.env` — remove 8 legacy keys

**Frontend (`next_b2b_starter/`):**
- `lib/auth/server-constants.ts` — fix session duration to match backend
- `lib/auth/constants.ts` — verify cookie config alignment

**Database:** None
**Dependencies:** None added; `stytch` + `@stytch/nextjs` already present
