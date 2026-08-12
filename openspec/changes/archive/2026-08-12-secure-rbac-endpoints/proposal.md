# Proposal: secure-rbac-endpoints

## Why

The RBAC API (`/api/rbac/*`) is the only unauthenticated surface on the backend: `internal/modules/auth/routes.go:21` registers the group with the comment "RBAC endpoints are public and do NOT require authentication", so the full role/permission model — including `org:manage`, `contact:export`, and other grants — is readable by any anonymous caller. This violates the living spec (`openspec/specs/stytch-authorization/spec.md` — "RBAC API endpoint authentication": *"All RBAC API endpoints (roles, permissions, by-category, role details, check-permission, metadata) SHALL require a valid authenticated session. Unauthenticated requests MUST receive a 401 Unauthorized response."*).

## What Changes

- **Backend** (`go-b2b-starter/internal/modules/auth/routes.go`): add the `auth` middleware (`resolver.Get("auth")`) to the `/rbac` group, following the existing pattern used by the whatsapp mgmt group (`internal/modules/whatsapp/routes.go:25-28`). Update the "public" comment. No route handlers, DTOs, or response shapes change — this is middleware-only.
- **Frontend** (`next_b2b_starter/lib/api/api/repositories/rbac-repository.ts:39-41`): drop `skipAuth: true` from `getRoles()` so the request carries the session (client auto-attaches the access token when `skipAuth` is unset, `api-client.ts:138`, and 401s route to the existing `handleUnauthorizedResponse`). The only consumer is `invite-member.tsx:86` (settings page, post-login).
- **Tests**: unit test asserting 401 for unauthenticated `/api/rbac/roles` and 200 for an authenticated session (mock auth), matching the spec scenarios.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `stytch-authorization`: delta tightens the existing RBAC-endpoint-auth requirement into per-endpoint 401 scenarios and adds the frontend no-bypass contract (RBAC repository SHALL NOT use `skipAuth`). The base requirement already exists in the living spec — this change makes the code conform to it and the delta makes the contract testable.

## Non-Goals

- **No local credential storage.** This change adds no storage of passwords, MFA tokens, or session tokens; it only adds a middleware to existing endpoints. The `signup-repository.ts` `skipAuth: true` calls (`/signup`, `/magic-link`) are legitimately public pre-auth endpoints and are explicitly out of scope.
- No change to the permission-resolution source (hardcoded Go maps vs Stytch RBAC policy) — that drift is tracked separately (see `rbac.go`/`token_verifier.go` fast path).
- No change to RBAC DTO shapes or the JSON contract.
- No Stytch tenant policy changes — this change requires no RBAC policy edits.

## Rollback

- **Git state:** revert every file this change touches: `go-b2b-starter/internal/modules/auth/routes.go` (additive middleware line), new `go-b2b-starter/internal/modules/auth/routes_test.go`, `next_b2b_starter/lib/api/api/repositories/rbac-repository.ts` (option removal), new `next_b2b_starter/lib/api/api/repositories/rbac-repository.test.ts`, `next_b2b_starter/e2e/specs/surrounding-processes.spec.ts` (added cases), and the OpenSpec artifacts under `openspec/changes/secure-rbac-endpoints/` (plus the synced `openspec/specs/stytch-authorization/spec.md` if the delta is folded at archive). All source edits are single-line/additive; no migration, no data.
- **Stytch tenant policy state:** no tenant policy is modified, so no policy rollback is required.

## Assumptions

- No legitimate anonymous consumer of `/api/rbac/*` exists: the only frontend caller (`invite-member.tsx`) runs post-login in settings, and `docs/05-making-api-requests.md` only documents `skipAuth` as a client capability, not a consumer of these endpoints. Verified by grep across `next_b2b_starter`.
- The `auth` middleware already returns 401 for missing/invalid sessions (existing behavior used by every other protected group), so adding it cannot change response semantics for valid sessions.
- `AUTH_MOCK_ENABLED` dev path (mock identity) satisfies the `auth` middleware without Stytch calls, so local dev and e2e remain green.
