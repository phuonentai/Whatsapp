# Stytch M2M Service Auth

## Why

The platform has no way for service principals (background automation, scheduled jobs, external tooling) to call protected APIs — every authenticated path requires a member session. Stytch B2B M2M (machine-to-machine) authentication is free within the free tier (1,000 M2M clients; tokens are JWTs signed by the project JWKS, RS256, 1-hour default lifetime). The current background machinery (`internal/platform/outbox/dispatcher`) is in-process and sessionless, but the roadmap has out-of-band callers: scheduled campaign delivery, webhook-triggered automation, and future platform services. This change adds the M2M service-auth capability now — client provisioning, token issuance (`POST /v1/b2b/m2m/token/get-access-token`), JWT verification in Go (reusing the existing JWKS cache), scope→permission mapping, and org-scoped request binding — so the first consumer wires in without reworking auth.

## What Changes

- **M2M clients**: provisioned in the Stytch dashboard (client_id/client_secret; 1K free budget), each with scopes (`crm:read`, `whatsapp:send`, …) and an allowed-orgs custom claim.
- **Token issuance**: `POST /v1/b2b/m2m/token/get-access-token` with client credentials returns a JWT (RS256, project JWKS, 1h default); scopes requested bound the token to the client's assigned scopes.
- **Go verification**: new M2M auth middleware accepting `Authorization: Bearer <jwt>` — fast path verifies via the existing JWKS cache (≤300s TTL, same as member JWTs); slow path `POST /v1/b2b/m2m/token/authenticate-access-token`; audience/issuer/expiry checks; maps token scopes to the existing permission model (`stytch-authorization`).
- **Org binding**: M2M tokens are project-scoped, not org-scoped — the request MUST carry `X-Stytch-Organization-Id`, which the middleware validates against the client's allowed-orgs claim, preserving multi-tenancy invariants; requests outside the allowlist get 403.
- **First consumer**: the campaign delivery trigger (scheduled/automated sends) calls the protected send surface as an M2M principal; the concrete out-of-band caller is confirmed during implementation (assumption recorded — see Assumptions).
- **Docs**: `STYTCH_CONFIGURATION.md` gains M2M client provisioning, the 1K-client budget, scope naming convention, and token/rotation notes.

## Capabilities

### New Capabilities
- `m2m-service-auth`: M2M client scopes, token issuance, JWKS-verified JWT middleware with `authenticate-access-token` fallback, scope→permission mapping, org-scoped request binding, first consumer wiring.

### Modified Capabilities
- None.

## Impact

- **Backend**: new `internal/platform/m2m` (verifier + middleware) wired into the auth module; `internal/modules/whatsapp` campaign send path gains M2M-auth'ed caller support; identity context exposes the service principal (`authcontext` seam) alongside member identity.
- **Database**: none — no new tables; M2M clients live in Stytch.
- **Frontend**: none (M2M is server-to-server).
- **Dependencies**: none new (SDK already has M2M support; reuses existing JWKS cache + breaker).
- **Stytch tenant policy state**: M2M clients + scopes + custom claims are dashboard-managed; revocable per client (delete client or rotate secret).
- **Pricing posture**: free within 1K M2M clients; tokens are bearer JWTs with no per-token cost; $0 beyond MAU limits (M2M tokens do not count as MAU).

## Rollback

- **Git**: revert middleware, scope mapping, campaign-send wiring. No DB migration to roll back.
- **Stytch tenant policy state**: delete or disable M2M clients in the dashboard; rotate client secrets. Members and orgs are unaffected (M2M is additive).

## Non-Goals

- **NO local credential storage**: M2M client secrets live only in Stytch (and the caller's secure env); the platform never stores secrets; tokens are validated, never persisted.
- NOT member impersonation or user-token issuance for M2M — service principals only.
- NOT per-org self-service M2M client creation from the UI — clients are platform-provisioned (dashboard) in this change.
- NOT M2M-scoped session management UI, revocation dashboard, or billing metering of M2M usage.
- NOT changing existing member-session auth paths; M2M middleware is additive and coexists with `X-Forwarded-Auth`/JWT member auth.

## Assumptions

- The first out-of-band consumer (candidate: scheduled campaign delivery) is identified and wired during implementation; if no concrete out-of-band caller exists yet, the capability ships with middleware + tests + a provisioned platform client and the consumer task is recorded as gated on the caller's existence.
- M2M JWT JWKS URL and claim shape (audience, `scopes`, custom claims) verified against the test project during implementation (recorded in tasks).
- Allowed-orgs binding via custom claim is the chosen multi-tenancy guard; the alternative (per-request org header without allowlist validation) is rejected as unsafe.
