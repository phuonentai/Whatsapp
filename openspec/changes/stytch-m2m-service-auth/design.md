# Stytch M2M Service Auth — Design

## Context

Verified Stytch B2B M2M contract: M2M clients are created in the dashboard (client_id/client_secret; free tier includes 1,000); `POST /v1/b2b/m2m/token/get-access-token` exchanges credentials for a JWT signed with the project's JWKS (RS256, default 1-hour lifetime) carrying the client's scopes; `POST /v1/b2b/m2m/token/authenticate-access-token` validates a token server-side. M2M clients are **project-scoped** (not organization-scoped), so org binding must come from the platform. The repo's existing infrastructure to reuse: `internal/platform/stytch` breaker client (`Client.Run`, threshold 5/10s/2) and JWKS cache (`jwks_cache.go`, ≤300s TTL) used by the member-token verifier; the `authcontext` platform seam for resolved request identity; the permission model from `stytch-authorization` (policy-driven RBAC, permission strings like `org:manage`). Current background machinery (`internal/platform/outbox/dispatcher.go`) is in-process and does not call the HTTP API — so the first M2M consumer is the scheduled/automated campaign delivery trigger (verified surface: `whatsapp-campaigns` launch lifecycle), confirmed during implementation.

## Goals / Non-Goals

**Goals:**

- Service principals can call protected APIs with Stytch-issued M2M JWTs, verified locally via the existing JWKS cache with an API fallback.
- Scopes map to the existing permission model; org binding preserves multi-tenancy (a service can act only for orgs it is allowed to act for).
- Coexists with member-session auth; additive middleware, no changes to member paths.
- Strict SSOT: no local credential storage; tokens validated, never persisted.

**Non-Goals:**

- Local storage of client secrets; per-org self-service client creation UI; M2M session-management UI; changing member auth; billing metering of M2M.

## Decisions

### D1 — Middleware: JWKS fast path + `authenticate-access-token` slow path

`M2MAuthMiddleware` accepts `Authorization: Bearer <jwt>`; fast path verifies signature via the existing JWKS cache (≤300s TTL, consistent with member JWTs), checks `iss`, `aud`, expiry, and the M2M audience; slow path (signature/key unknown) calls `POST /v1/b2b/m2m/token/authenticate-access-token` behind the breaker (breaker-open → 503 `m2m_auth_unavailable`, never a silent accept). Failed verification → 401; no identity is established on failure.

- **Rationale:** mirrors the two-tier member-token strategy exactly (fast local JWKS, slow API) — one mental model, reuses the cache and breaker. Rejected: API-only validation (latency per call, breaker coupling), custom key management (redundant with project JWKS).

### D2 — Scope → permission mapping

Token `scopes` (`crm:read`, `crm:write`, `whatsapp:send`, …) map to the existing permission strings via a small declarative table (`m2m_scope_permissions`): e.g., `whatsapp:send` → `whatsapp:send`, `crm:read` → `crm:view`/`contact:view`. The middleware resolves the service principal's effective permission set into `authcontext` alongside member identities, so existing `RequirePermissionFunc` gates work unchanged for M2M callers.

- **Rationale:** zero changes to the permission-gating surface; M2M principals become first-class identities in the existing model. Scope naming follows the existing `resource:action` convention.

### D3 — Org binding via allowed-orgs claim + request header

Because M2M tokens are project-scoped, the request SHALL carry `X-Stytch-Organization-Id`; the middleware SHALL resolve the org from the token's custom `org_ids` allowlist claim (set at client creation) and SHALL 403 when the header is missing or not in the allowlist. The resolved org becomes the request's `OrganizationID` in `authcontext`.

- **Rationale:** preserves the multi-tenancy invariants (org context always server-resolved, never trusted from an unvalidated source). Rejected: trusting a bare header (spoofable), org-less M2M (breaks every org-scoped handler).

### D4 — First consumer: campaign delivery trigger

The scheduled/automated campaign send path calls the protected send surface with an M2M client provisioned for the platform (`org_ids` allowlist + `whatsapp:send` scope). Wired only if the concrete out-of-band caller exists at implementation time; otherwise the capability ships with middleware + tests + provisioned client, and the consumer task is recorded as gated (proposal Assumptions).

### D5 — Client provisioning

Clients are dashboard-provisioned (no UI). Secret rotation = create new client / rotate in dashboard + update the caller's env. The 1K free budget and naming convention (`m2m.<purpose>`) are documented in `STYTCH_CONFIGURATION.md`.

## Risks / Trade-offs

- **Token replay / theft** → short default lifetime (1h), scopes bound per client, org allowlist binding, secrets only in Stytch + caller env; revoke by deleting/rotating the client.
- **Org spoofing via header** → header always validated against the client's allowlist claim; never used unvalidated.
- **M2M tokens counting against quota** → 1K clients free; tokens themselves are bearer JWTs with no per-token cost; documented.
- **JWKS drift for M2M audience** → same ≤300s cache bound as member JWTs; slow path fallback covers key rotation races.
- **No current consumer** → capability is forward-looking; consumer task gated on the caller's existence (assumption recorded), middleware is test-covered regardless.
- **Scope mapping drift** → declarative table, reviewed against `stytch-authorization` permission strings; unknown scopes denied.

## Migration Plan

1. Test project: create the platform M2M client(s) with scopes + `org_ids` allowlist; verify token issuance + JWKS.
2. Backend: verifier + middleware + scope table + identity plumbing (`authcontext`); unit tests (fast path, slow path, breaker-open, org allowlist, scope mapping, 401/403).
3. Wire the first consumer (campaign send) if the caller exists; else record gated task.
4. Prod: provision prod client, set caller env, monitor auth failures.
5. Rollback: Git revert (no migration); delete/disable clients in dashboard.

## Open Questions

- Exact M2M JWT claim shape (audience value, `scopes` casing, custom-claims embedding) — verified in the test project during implementation.
- Whether `org_ids` should be a custom claim or derived from a per-org client convention (default: custom claim) — confirmed during implementation.
- Concrete first consumer confirmation (scheduled campaign delivery vs. a future automation surface) — resolved during implementation.
