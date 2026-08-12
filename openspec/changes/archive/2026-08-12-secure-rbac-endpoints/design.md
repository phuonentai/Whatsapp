# Design: secure-rbac-endpoints

## Context

The RBAC catalog API (`/api/rbac/*`) is the only unauthenticated surface on the Go backend. `internal/modules/auth/routes.go:21-49` registers the group with no auth middleware and an explicit "public" comment, contradicting `openspec/specs/stytch-authorization/spec.md` ("RBAC API endpoint authentication" — all six endpoints SHALL require a valid authenticated session; unauthenticated MUST get 401). The frontend compounds it: `rbac-repository.ts:39-41` calls `/rbac/roles` with `skipAuth: true`, and `api-client.ts:138` only attaches the access token when `skipAuth` is unset.

The security impact is real: the anonymous response exposes the full role/permission model — including `org:manage`, `contact:export`, and other grants — and enables unauthenticated `check-permission` probing.

Verified facts (premise validation, 2026-08-11):
- Spec requirement: `openspec/specs/stytch-authorization/spec.md` (lines 133-135).
- Code: `go-b2b-starter/internal/modules/auth/routes.go:21-49` — `/rbac` group without `resolver.Get("auth")`; comment at lines 22-23.
- Pattern: `go-b2b-starter/internal/modules/whatsapp/routes.go:25-28` — `mgmt := router.Group("/v1/whatsapp")` + `mgmt.Use(resolver.Get("auth"), resolver.Get("org_context"), resolver.Get("subscription"))`.
- Frontend consumers of `/rbac/*`: only `next_b2b_starter/app/dashboard/settings/components/invite-member.tsx:86` (`rbacRepository.getRoles()`), rendered post-login inside the settings page. No anonymous consumer exists (grep across `next_b2b_starter` for `rbacRepository|/rbac/|check-permission`; `docs/05-making-api-requests.md:222` only documents `skipAuth` as a client capability).
- `api-client.ts:200-204` — 401 responses route to `handleUnauthorizedResponse` (session-expiry redirect) whenever `skipAuth` is unset.
- `signup-repository.ts:28-29,49-50` uses `skipAuth: true` for `/signup` and `/magic-link` — legitimately public pre-auth endpoints; out of scope.

## Goals / Non-Goals

**Goals:**
- Make `/api/rbac/*` require an authenticated session; unauthenticated → 401 (spec conformance).
- Make the frontend RBAC fetch carry the session; remove the `skipAuth` bypass.
- Zero change to RBAC handlers, DTOs, response shapes, or the JSON contract.
- Unit-level proof of the 401/200 behavior on the group.

**Non-Goals:**
- No change to the permission-resolution source of truth (hardcoded `rbac.go` maps vs Stytch policy — tracked separately).
- No change to `signup`/`magic-link` (legitimately public).
- No Stytch tenant policy changes; no local credential storage (nothing is stored).
- No org-scoping of the RBAC catalog (the catalog is global, not org-scoped data).

## Decisions

### D1: Group-level `auth` middleware on `/rbac` (not per-route)

Add `rbacGroup.Use(resolver.Get("auth"))` once in `routes.go`, mirroring the whatsapp mgmt-group pattern. Rationale: one enforcement point, no chance of missing a future route added to the group, matches the codebase's dominant convention (all other protected groups use group-level middleware). Alternatives: per-route middleware — rejected (verbose, easy to miss a new route); a named dedicated middleware — rejected (the existing `auth` already returns 401 for missing/invalid sessions, which is exactly the required semantics).

### D2: `auth` only, no `org_context`

The spec requires only "a valid authenticated session". The RBAC catalog (roles/permissions/metadata) is global configuration, not org-scoped data; `check-permission` takes a role+permission pair with no org parameter. Adding `org_context` would reject sessions without an org (or force org selection) and adds no security. Revisit only if an org-scoped RBAC endpoint is ever added.

### D3: Frontend — drop `skipAuth: true` from `getRoles()`

`api-client.ts:138` computes `shouldAttachAuth = !skipAuth && !headers["Authorization"]`, so removing the flag attaches the session access token automatically, and the 401 path (`handleUnauthorizedResponse`) already exists for session-expiry UX. The roles response is cached in-memory per repository instance, so the cost is one authenticated fetch per page lifetime — unchanged from today. Alternative: attach `Authorization` explicitly — rejected (duplicates client logic, misses the 401 handling).

### D4: Delta spec MODIFIES the existing requirement

The base spec already mandates RBAC endpoint auth; the delta (a) expands 401 coverage to per-endpoint scenarios (check-permission, metadata) and (b) adds the frontend no-bypass contract (`skipAuth` MUST NOT be used for RBAC). MODIFIED is correct because the requirement exists and is being tightened/operationalized, not introduced.

### D5: Verification via unit test + curl

Go: extend the auth module's route tests — construct the router with the auth middleware, assert 401 without a session and 200 with a valid mock session on `/api/rbac/roles` (plus 401 on `check-permission`/`metadata`). Gate: `go build ./...`, `go vet ./...`, `go test ./internal/modules/auth/...`, live curl `401`/`200` check, `pnpm lint`, `npx tsc --noEmit`. E2E specs that exercise invite flows cover the authenticated path.

## Risks / Trade-offs

- **401 redirect on stale session**: if the settings page holds a stale session, the roles fetch now 401s and redirects to login (previously it silently served). This is the intended, spec-mandated behavior and matches every other authenticated fetch.
- **In-memory roles cache persists across logout within a page lifetime**: cached `RbacRole[]` is static catalog data (no org/member-specific values); a post-logout call within the same page would still resolve from cache without a network round trip. Low risk; acceptable. Optionally cleared on 401 in a follow-up.
- **AUTH_MOCK_ENABLED dev parity**: the mock-auth path injects an identity into the middleware, satisfying the group — local dev and e2e continue to work. Verified at gate by the e2e invite/settings specs.
- **Scope creep guard**: no handler, DTO, or policy edits; if a review suggests otherwise, it belongs in a separate change.
