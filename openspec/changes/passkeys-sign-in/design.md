# Passkeys Sign-In — Design

## Context

The sign-in path is email-magic-link only (server actions in `lib/actions/auth/`). This change adds passkey (WebAuthn/FIDO2) registration, sign-in, and self-service management via the Stytch B2B WebAuthn surface (`POST /v1/b2b/webauthn/register/start`, `/register`, `/authenticate/start`, `/authenticate`; member WebAuthn list/delete via `/v1/b2b/member_webauthn`). The `@stytch/nextjs/b2b` SDK drives the browser `navigator.credentials` ceremony.

This revision incorporates the council verdict (`VERDICT.md`, REJECTED, 2026-08-12). All five required design changes are folded in; premises below were re-verified against the codebase.

**Verified premises (codebase):**

- `getStytchB2BClient()` in `lib/auth/stytch/server.ts` constructs `new Stytch.B2BClient(...)` directly — **no circuit breaker wraps the Next.js Stytch client**. The two-tier breaker (threshold 5 / timeout 10s / half-open probe 2) exists only in the Go adapter (`go-b2b-starter/internal/infrastructure/auth/stytch`).
- `consumeMagicLink` (`lib/actions/auth/consume-magic-link.ts`) is the reference pattern: sets `stytch_session` / `stytch_session_jwt` cookies (via `getCookieConfig` / `getSecureCookieConfig`, maxAge from `getSessionDurationMinutes()`) only when `member_authenticated`; when MFA-gated it returns `intermediateSessionToken`, `mfaRequired`, `primaryRequired`; audit writes (`recordAuthAudit`, types `login_succeeded` / `login_failed` / `mfa_challenge_*`) are best-effort and non-blocking.
- `sendMagicLink` performs email-first membership validation via `organizations.members.search` — the server-side re-resolution seam used by this design.
- The only rate limiter is the in-process sliding window in `lib/auth/magic-link-limiter.ts` (per-email + per-IP, env-configurable, single-instance assumption documented).
- `mfa-totp-enrollment` (in-flight) provides the MFA challenge step on `/auth` that exchanges `intermediate_session_token` for a full session.

## Goals / Non-Goals

**Goals:**

- Passkey registration (settings → profile → Security), passkey sign-in on `/auth` with magic-link fallback, self-service list/delete of own passkeys.
- Explicit, implementable resilience contracts: frontend circuit-breaker wrapper, per-email/per-IP rate limiting, ceremony abort timeout with outcome taxonomy, session-derived ownership scoping.
- Strict SSOT: no local passkey material; session cookies only after a valid Stytch session.

**Non-Goals:**

- No backend changes — the Go breaker adapter is untouched; this change adds a NEW frontend seam.
- Not retrofitting the breaker onto existing magic-link actions (they keep calling the SDK directly; the wrapper is the seam for future auth actions).
- No RP ID changes post-launch; no SSO-federated passkey ownership; no local passkey storage; magic links remain the fallback.

## Decisions

### D1 — Frontend circuit-breaker wrapper (new seam, verdict #1)

Add `lib/auth/stytch/breaker.ts`: a stateful circuit breaker mirroring the Go adapter contract — threshold 5 consecutive failures, open timeout 10s, half-open probe 2 — plus `runWithBreaker(fn)` that wraps any Stytch B2B client call. Breaker-open and Stytch 5xx map to a structured `passkey_unavailable` error (503-style). All passkey server actions execute their Stytch calls through this wrapper; no session cookies are ever issued on the failure path.

*Alternatives considered:* (a) raw try/catch on each action — rejected: governance requires explicit breaker/fallback states per outbound Stytch invocation and the verdict called the original unverifiable claim; (b) reuse the Go breaker via a new backend endpoint — rejected: backend scope is explicitly out of this change and would add latency for no benefit.

### D2 — Registration (settings → profile → Security)

- `createPasskeyRegistration(sessionJwt)` → Stytch `webauthn.register.start` (breaker-wrapped) → creation options; client runs `navigator.credentials.create` (conditional UI where available, bounded by a 60s abort timeout); `completePasskeyRegistration(credentialJson)` → Stytch `webauthn.register` (breaker-wrapped).
- Invariant: registration requires an authenticated session (`session_jwt`); a passkey is recorded by Stytch only after a successful complete call; no partial registration on failure (no local state exists to corrupt).

### D3 — Sign-in on `/auth` (passkey branch after email resolution)

- Keep email-first membership validation (`members.search`) — preserves anti-enumeration UX and prevents passkey prompts for unknown emails.
- `startPasskeyAuthentication(email)` re-resolves the member server-side via `members.search` from the email — **never accepts an opaque client-supplied `memberId`** (verdict #3). It is rate-limited (D5) and breaker-wrapped, then returns Stytch `webauthn.authenticate.start` options for the browser `navigator.credentials.get` (bounded by a 120s abort timeout).
- `completePasskeyAuthentication(assertionJson, sessionDurationMinutes)` → Stytch `webauthn.authenticate` (breaker-wrapped). On `member_authenticated: true`: set the existing cookie pair, record `login_succeeded` (best-effort), redirect to destination. On `member_authenticated: false`: return `intermediateSessionToken` + `mfaRequired` + `primaryRequired` to the client; no cookies; the existing `/auth` MFA challenge step (per `mfa-totp-enrollment`) exchanges the intermediate token (verdict #4, mirrors `consumeMagicLink` exactly).
- If the member has no passkeys, the user cancels, or the ceremony/Stytch call fails, fall back to the magic-link path — no dead-end.

### D4 — Session and cookie reuse

- Passkey auth mints the same session cookie pair as magic links; refresh/logout/edge-validation paths are unchanged. Logout revokes the session (existing behavior). Cookies are set only from the Stytch authenticate response — never from client input.

### D5 — Rate limiting (verdict #2)

Add `lib/auth/passkey-limiter.ts` reusing the magic-link limiter's in-process sliding-window pattern: per-email and per-IP windows, env-configurable (`PASSKEY_AUTH_RATE_LIMIT_PER_EMAIL_PER_HOUR`, `PASSKEY_AUTH_RATE_LIMIT_PER_IP_PER_HOUR`; defaults 10/hr email, 30/hr IP). Applied to `startPasskeyAuthentication` and `createPasskeyRegistration`. Bounds challenge-start spam against valid member emails and registration brute-force attempts. Documents the single-instance assumption (matches the magic-link limiter; swap to a distributed limiter if scaled).

### D6 — Ceremony lifecycle & observability (verdict #5)

- Both `navigator.credentials.create` and `.get` are wrapped with `AbortController` timeouts (60s / 120s, env-configurable).
- Outcome taxonomy: user-cancel (`NotAllowedError` / abort by user) → silent magic-link fallback, no failure audit; Stytch failure or breaker-open → structured `passkey_unavailable` error + bounded `login_failed` audit detail + magic-link fallback; network failure → same as Stytch failure path.
- Audit writes are best-effort and never block the auth outcome (existing `recordAuthAudit` contract). Reuses the `auth-audit-events` taxonomy: `login_succeeded` / `login_failed` with `detail: "passkey"` context; no credential material in audit detail.

### D7 — RP ID

RP ID = app domain (e.g., `yourdomain.com`), configured in the Stytch dashboard once, before rollout (RP ID is immutable per project in practice). Documented in `STYTCH_CONFIGURATION.md`.

## Stytch Boundary & Fallback (revised)

| Operation | Stytch API | Failure behavior (breaker 5/10s/2) |
|---|---|---|
| Register start/complete | `POST /v1/b2b/webauthn/register/start`, `/register` | breaker-open/5xx → 503 `passkey_unavailable`; no partial registration; UI error state |
| Authenticate start/complete | `POST /v1/b2b/webauthn/authenticate/start`, `/authenticate` | breaker-open/5xx → 503 `passkey_unavailable`; no session issuance; magic-link fallback |
| List/delete | `GET/DELETE /v1/b2b/member_webauthn` | breaker-open → 503 error state; delete idempotent (404 treated as success) |
| MFA-gated response | `member_authenticated: false` | no cookies; `intermediate_session_token` → existing TOTP challenge step |

## Security Invariants

- Passkey material (public keys, attestation, assertion responses) handled by Stytch only; nothing stored locally (SSOT).
- The passkey ceremony runs on the app origin only; no third-party iframe reliance for the custom flow.
- Server actions never trust client-supplied member/org IDs for management; list/delete derive scope from the verified session JWT; challenge start re-resolves the member from the email server-side.
- Rate limits bound challenge-start spam; the flow does not reveal passkey existence.
- Cookies set only after Stytch returns a valid session.
- Auth audit events reuse the `auth-audit-events` taxonomy (`login_succeeded` / `login_failed`, `detail: "passkey"` context), best-effort and non-blocking, with bounded detail only.

## Risks / Trade-offs

- [Risk] In-process rate limiter is inaccurate under multi-instance deployment → Mitigation: documented single-instance assumption (identical to the existing magic-link limiter); swap to a distributed limiter when the app scales.
- [Risk] Abort timeout too short for slow authenticators (e.g., hardware keys) → Mitigation: configurable timeouts, defaults 60s/120s; user-cancel path is a clean magic-link fallback, not an error.
- [Risk] Breaker threshold tuning on the frontend path could false-trip under transient Stytch latency → Mitigation: mirror the proven Go adapter contract (5/10s/2); half-open probe 2 recovers quickly.
- [Risk] Rollback semantics of "disable passkeys product" are undocumented (does it revoke enrolled credentials?) → Mitigation: assumption recorded in the proposal; verify in the Stytch test project during E2E; per-member `DELETE /v1/b2b/member_webauthn/{id}` is the deterministic cleanup path.
- [Risk] RP ID is immutable per project → Mitigation: chosen deliberately before rollout; documented in `STYTCH_CONFIGURATION.md`; rollout gated on dashboard config.

## Migration Plan

- **Deploy:** enable the passkeys product in the Stytch test project; set RP ID in the dashboard; ship frontend (breaker wrapper → limiter → actions → UI). Magic-link path untouched and remains default; passkeys are additive.
- **Rollback (Git):** revert frontend actions/UI.
- **Rollback (Stytch):** disable the passkeys product or delete member WebAuthn instances via `DELETE /v1/b2b/member_webauthn/{member_webauthn_id}`; no org-level policy change required.

## Open Questions

- Does disabling the Stytch passkeys product revoke already-enrolled member WebAuthn credentials? (Assumption: it blocks new registrations; enrolled keys may still work. Verify in the test project.)
- Does Stytch's `webauthn.authenticate.start` surface to clients whether a member has passkeys (challenge options shape)? The UI decides branch visibility from the member WebAuthn list; rate limiting bounds any residual probing.

## Testing Strategy

- Unit: breaker wrapper (trips at 5, opens 10s, half-open probe 2, 503 mapping); passkey limiter (per-email/IP windows); passkey actions with mocked Stytch client — cookies only on success; `mfa_required` → intermediate token passthrough + no cookies; breaker-open → 503 + no cookies; delete with session-derived scoping ignores client-supplied IDs.
- Component: registration UI states; `/auth` passkey branch visibility (member with passkeys vs without); user-cancel → silent magic-link fallback; MFA routing.
- E2E (Stytch test project): register passkey → sign out → sign in with passkey → session cookie set; delete passkey → sign-in falls back to magic link; breaker-open (mocked) → 503 → magic link.
