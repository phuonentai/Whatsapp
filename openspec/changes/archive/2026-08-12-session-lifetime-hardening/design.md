# Session Lifetime Hardening — Design

## Context

Stytch sessions: `session_duration_minutes` (5 min–366 days) set at auth time; sliding requires explicit periodic `sessions.authenticate({ session_duration_minutes })` from the client. JWT revocation is delayed until JWT expiry; session-token revocation is immediate; sessions auto-revoke when a member is deleted.

## Decisions

### D1 — Sliding renewal client hook

- New `useSlidingSession(durationMinutes)` hook (client): on mount and every 10 minutes while `document.visibilityState === "visible"`, call `POST /api/auth/session/refresh` (existing route) with `session_duration_minutes` so the returned session/JWT carries the extended lifetime; existing token-manager keeps the JWT fresh.
- On refresh failure (401/revoked): clear cookies, redirect to `/auth?returnTo=…` (reuse existing refresh error handling).
- Guard: only run when a session exists; skip on public pages.
- Note: the refresh route currently re-mints the JWT but does not extend the underlying session (it does not pass `session_duration_minutes`); this change makes it pass the env duration so the session itself slides.

### D2 — Single session-duration source of truth

- `getSessionDurationMinutes()` (env, default 480) remains the single code source; it is already passed at login (`consumeMagicLink`) and will be passed by the refresh route.
- Update `STYTCH_CONFIGURATION.md` (43200 → default 480, env-overridable) and `docs/02-authentication.md` (already 8h — keep) so docs match code.

### D3 — Revoke sessions on member deactivation

- Backend `member_service` deactivation path: after local status update succeeds, call the auth adapter `RevokeMemberSessions(ctx, stytchOrgID, stytchMemberID)`:
  - `GET /v1/b2b/sessions` filtered by org+member → for each `session_token`, `POST /v1/b2b/sessions/revoke`.
- Adapter implements a domain interface (`SessionRevoker` in `organizations/domain`); no Stytch imports in domain (constitution). All calls through the existing circuit breaker; on breaker-open, log warning + return the deactivation result with a structured notice that session revocation is pending (deactivation itself must not silently fail — the local status change is the source of truth for access control in the app; Stytch revocation is the enforcement at the identity layer).
- Idempotency: revoking an already-revoked session is a no-op error; treat as success (idempotent state check per governance rules).

### D4 — JWT fast-path note

- Document that Go fast-path JWT validation may accept a deactivated member's JWT until its own expiry (bounded by the sliding 8h session); mitigation = immediate session-token revocation + short JWT refresh cadence. No Go verification change in this change (stateless JWKS validation cannot check member status without an API call; the slow path already rejects via Stytch when invoked).

## Stytch Boundary & Fallback

| Operation | Stytch API | Fallback on failure |
|---|---|---|
| Sliding renewal | `sessions.authenticate` (via refresh route) | failure → logout/redirect (existing path) |
| List member sessions | `GET /v1/b2b/sessions` (org+member filter) | breaker → warning; revocation deferred (logged) |
| Revoke session | `POST /v1/b2b/sessions/revoke` | idempotent; breaker → warning |

## Security Invariants

- Sessions remain fully owned by Stytch; local storage never holds session state (SSOT).
- Deactivation semantics: local status change is authoritative for app access; Stytch revocation closes the identity-layer window promptly.
- Renewal extends lifetime only while the user actively keeps the app open; no indefinite background sessions.

## Testing Strategy

- FE: hook unit tests (interval, visibility gating, failure → redirect); refresh-route test asserting `session_duration_minutes` is passed and session extends.
- BE: adapter `RevokeMemberSessions` with mock Stytch client (list+revoke happy path, already-revoked no-op, breaker-open → deferred warning); service test asserting revocation called after status update.
- Docs: grep asserts `STYTCH_CONFIGURATION.md` session line matches env default.
