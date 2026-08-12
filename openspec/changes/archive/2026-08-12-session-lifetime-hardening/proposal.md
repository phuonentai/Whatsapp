# Session Lifetime Hardening

## Why

Three related session-lifecycle gaps:

1. **Fixed, not sliding, sessions.** Session lifetime is set once at login (`session_duration_minutes` from `NEXT_PUBLIC_STYTCH_SESSION_DURATION_MINUTES`, default 480 min). Users hit a hard logout mid-work regardless of activity. Stytch supports sliding sessions, implemented client-side by periodically re-authenticating with `session_duration_minutes` (docs pattern: `sessions.authenticate` every ~10 min while the app is open).
2. **Env/doc drift.** Code default is 480 min (8h) (`lib/auth/server-constants.ts`); `STYTCH_CONFIGURATION.md` documents 43200 (30 days). Two sources of truth that disagree.
3. **No revocation on member deactivation.** When an admin deactivates a member, the local `status` field changes, but active Stytch sessions are not revoked. The Go JWT fast path (local JWKS) keeps accepting the member's JWT until expiry (Stytch docs: JWT revocation is delayed until the JWT window ends; session-token revocation is immediate). Deactivated members keep API access up to the session lifetime.

## What Changes

- **Sliding sessions:** while the app is open, the client periodically calls `sessions.authenticate` with `session_duration_minutes` (env value) to extend the session; on failure (session revoked/expired), the client logs out / redirects to `/auth`. Interval ~10 min, honors document visibility (`document.visibilityState`).
- **Single session default:** standardize on 480 min (8h) as the documented default; `STYTCH_CONFIGURATION.md` and `docs/02-authentication.md` updated to match the env-driven truth; env remains the override.
- **Revoke on deactivation:** the admin deactivation path (backend member status update) additionally revokes the member's active sessions via Stytch (list sessions for member → revoke each). This makes deactivation effective within the JWT window (token revoked immediately; JWT honored only until its own expiry, bounded by the 8h session).
- Session revocation is a Stytch-side operation; nothing local stores session state (SSOT).

## Capabilities

### New Capabilities
- `session-management`: sliding session renewal, session revocation on member deactivation, and consistent session-duration configuration.

### Modified Capabilities
- None.

## Impact

- **Frontend:** sliding-renewal hook/manager in the client (works with `token-manager.ts` refresh logic); logout-on-revoked-session handling; docs updates.
- **Backend:** member deactivation service gains a Stytch session-revocation step (via the auth adapter + circuit breaker); no schema change.
- **Dependencies:** none new.
- **Stytch:** deactivation now mutates tenant session state (revocations); documented rollback = re-invite/reactivate member.

## Rollback

- **Git:** revert the renewal hook and the deactivation step; fixed-lifetime behavior and deactivate-without-revoke return.
- **Stytch tenant policy state:** revocations are per-session and irreversible (sessions are gone); reactivation of a member requires a fresh login — documented as the intended semantics. No org-level policy changes.

## Non-Goals

- NOT implementing server-side session management (Stytch is the session store per SSOT).
- NOT adding a "sign out of all devices" UI in this change (session listing UI is future work; the deactivation revocation covers the admin-driven case).
- NOT reducing the session duration; only making renewal sliding and documentation consistent.
