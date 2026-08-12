# TOTP MFA — Design

## Context

Stytch B2B MFA facts (docs + SDK types): secondary factors are SMS OTP and TOTP only; org-level `mfa_policy` is `REQUIRED_FOR_ALL` or `OPTIONAL`; `mfa_methods` is `ALL_ALLOWED` or `RESTRICTED` with `allowed_mfa_methods ∈ [sms_otp, totp]`. When MFA is required, primary auth returns `member_authenticated: false` and an `intermediate_session_token`; the full session is minted only after MFA success. TOTP create returns `qr_code` (server-generated image), `secret`, `totp_id`, and one-time `recovery_codes`.

This revision incorporates the council verdict (`VERDICT.md`, REJECTED, 2026-08-12). All five required design changes are folded in; premises were re-verified against the codebase and the Stytch SDK.

**Verified premises (codebase + SDK):**

- Go backend for org policy is implemented and wired: `domain/mfa_policy.go` (typed enums, `MfaPolicyUpdater`, `ValidateMfaPolicyUpdate`), `stytchOrganizationRepository.UpdateMfaPolicy` (implements `domain.MfaPolicyUpdater` via `PUT /v1/b2b/organizations`), `organization_service.go`, `mfa_policy_handler.go`, route `PUT /api/organizations/mfa-policy` gated by `auth.RequirePermissionFunc("org", "manage")`, breaker-open → 503 structured error (`mfa_policy_update_unavailable`).
- Circuit breaker verified in `internal/platform/stytch/circuit_breaker.go` (threshold 5 / cooldown 10s / half-open probes 2); `Client.Run()` guards outbound Stytch calls; `ErrCircuitOpen`; `BreakerState()` diagnostics.
- `B2BRecoveryCodesRecoverResponse` (SDK) returns `session_token`/`session_jwt` **directly** — the recover endpoint completes the MFA flow and consumes the code in one call.
- `B2BTOTPsCreateResponse` (SDK) returns a server-generated `qr_code` plus `secret` and `recovery_codes` — no QR rendering dependency is needed.
- `B2BTOTPsAuthenticateResponse` (SDK) returns the session directly (success = session; failure = thrown error); the member object exposes `totp_registration_id`.
- The in-process sliding-window limiter pattern exists in `lib/auth/magic-link-limiter.ts` (per-email/per-IP, env-configurable, single-instance assumption documented).

## Goals / Non-Goals

**Goals:**

- TOTP enrollment (self-service), MFA challenge continuation at login (intermediate session → TOTP/recovery → session cookies), and per-org MFA policy management (`org:manage`).
- Correct Stytch contract usage: recovery completes directly; QR is server-generated; enrollment is duplicate-guarded.
- Bounded attack surface: recovery-code attempts rate-limited per IP/member; bounded audit detail.
- Strict SSOT: no local MFA material; the org policy mirror is display-only and never authorizes.

**Non-Goals:**

- SMS OTP as an MFA factor (cost + country allowlist complexity) — `allowed_mfa_methods` restricts to `totp`.
- Email as an MFA factor (not supported by Stytch B2B).
- Step-up MFA for high-risk actions (billing, exports) — policy-level login MFA only.
- No local storage of secrets, recovery codes, or session material.

## Decisions

### D1 — Login continuation: intermediate session → TOTP challenge

Extend the `/authenticate` page flow: when `memberAuthenticated === false && mfaRequired` (from `consumeMagicLink`), render an MFA step (TOTP code input + "use recovery code" toggle) instead of erroring. New server action `authenticateTotp({ intermediateSessionToken, code })`:

- **TOTP path:** `POST /v1/b2b/totp/authenticate` with `intermediate_session_token` → on success set `stytch_session` / `stytch_session_jwt` cookies (same cookie config as `consumeMagicLink`) → record `login_succeeded` (after MFA, per `auth-audit-events`). Wrong code → StytchError → show error on the challenge step, no cookies, bounded `mfa_challenge_failed` audit detail.
- **Recovery path:** `POST /v1/b2b/totp/recovery_codes/recover` with `intermediate_session_token` + `recovery_code` → **the response carries `session_token`/`session_jwt` directly** (verified `B2BRecoveryCodesRecoverResponse`) → set cookies from that response → record `login_succeeded`. There is NO chained "repeat TOTP authenticate": the intermediate token is single-use and consumed by the recover call.
- Member/org IDs are NOT taken from the client as authority: the `intermediate_session_token` is bound by Stytch to the member; any supplied IDs are context only (log mismatch if observed). This matches the passkeys-sign-in ownership-binding posture.
- State-transition invariant: cookies are set ONLY after a successful TOTP or recovery exchange.

### D2 — Enrollment: settings → profile (duplicate-guarded)

`createTotp(sessionJwt)` → `POST /v1/b2b/totp/create` with the member `session_jwt` → render the **server-generated `qr_code`** (verified `B2BTOTPsCreateResponse`) alongside the manual secret; require one verification code (`totp/authenticate` with `session_jwt`) before marking enrolled; then show `recovery_codes` once with an "I saved these" confirmation.

- **Idempotency/duplicate guard (verdict #4):** before create, read the session member's `totp_registration_id` (verified on the SDK member object). If a registration already exists, surface "manage existing authenticator" (verify/remove/rotate) instead of creating a second instance. Stytch multi-registration semantics are undocumented — verify in the test project during E2E and record the outcome in tasks (proposal Assumptions).
- Enrollment is possible even when org policy is `OPTIONAL` (Stytch supports optional enrollment; once enrolled the member must pass MFA at login).

### D3 — Org policy: backend-mediated Stytch update

Go: `organizations` service `UpdateMfaPolicy(ctx, orgID, policy, methods)` calls Stytch `organizations.update` (`mfa_policy`, `mfa_methods`, `allowed_mfa_methods`) via the auth adapter, through the existing two-tier circuit breaker (threshold 5 / timeout 10s / half-open probe 2 — verified `internal/platform/stytch/circuit_breaker.go`); breaker-open or Stytch unreachable → 503 with structured error (`mfa_policy_update_unavailable`). Frontend compliance section reads current policy from the member/org profile (Stytch org object via session) and posts updates to the Go endpoint.

- **Authorization invariant (verdict #5):** the locally read/mirrored policy is display-only. Authorization is always enforced by Stytch at session mint (`mfa_policy: REQUIRED_FOR_ALL` blocks session creation until MFA). No code path may consult the mirrored values to gate access.

### D4 — RBAC gating

Org policy UI gated by `org:manage` (verified on the route: `auth.RequirePermissionFunc("org", "manage")`); enrollment gated by any authenticated member (self-service).

### D5 — MFA rate limiting (verdict #3)

Add `lib/auth/mfa-limiter.ts` reusing the magic-link limiter's in-process sliding-window pattern: per-IP and per-member windows, env-configurable (`MFA_RATE_LIMIT_PER_MEMBER_PER_HOUR`, `MFA_RATE_LIMIT_PER_IP_PER_HOUR`; defaults 10/hr member, 30/hr IP). Applied to `authenticateTotp` — especially the recovery-code path (recovery codes are static bearer secrets). Wrong-TOTP attempts are naturally bounded by 30s code windows but are still logged with bounded `mfa_challenge_failed` detail; on limiter rejection the flow returns a generic "try again later" error that does not reveal whether the code was close or valid. Documents the single-instance assumption (matches the magic-link limiter).

## Stytch Boundary & Fallback

| Operation | Stytch API | Failure behavior |
|---|---|---|
| Enroll | `POST /v1/b2b/totp/create`, `POST /v1/b2b/totp/authenticate` | Circuit breaker → 503; UI error, no partial state; existing registration surfaces instead of duplicate |
| Login MFA (TOTP) | `POST /v1/b2b/totp/authenticate` | breaker → error state on MFA step, session NOT set; bounded `mfa_challenge_failed` audit |
| Login MFA (recovery) | `POST /v1/b2b/totp/recovery_codes/recover` | rate limiter rejection or breaker → error state, session NOT set; recover returns the session directly on success |
| Policy update | `PUT /v1/b2b/organizations` | breaker → 503 structured error; policy unchanged |

All outbound calls: existing two-tier breaker (5/10s/2); 2h task cap per unit; signature-validation + mock fallback tests per governance rules.

## Security Invariants

- TOTP secrets, recovery codes, and MFA state exist ONLY in Stytch (SSOT). Local DB/audit stores only event types (`mfa_challenge_passed/failed`) and policy display values.
- Recovery codes render once and never enter logs; audit `detail` stays bounded.
- Intermediate session tokens are single-use and short-lived; they are never logged or persisted.
- Recovery-code attempts are rate-limited per member and per IP; rejection never reveals code validity.
- The org policy mirror is display-only — never used for authorization decisions; Stytch is the sole enforcement point.

## Risks / Trade-offs

- [Risk] In-process rate limiter is inaccurate under multi-instance deployment → Mitigation: documented single-instance assumption (identical to the magic-link limiter); swap to a distributed limiter when the app scales.
- [Risk] Stytch multi-TOTP-registration semantics unknown → Mitigation: duplicate guard via `totp_registration_id`; verify in test project during E2E; fallback is surface-existing + rotate.
- [Risk] Recovery-code UX friction (consumed codes are gone; `recovery_codes_remaining` decrements) → Mitigation: surface `recovery_codes_remaining` in the settings view and document rotation (`recovery_codes/rotate`).
- [Risk] A member with REQUIRED_FOR_ALL org but no enrolled factor cannot complete login → Mitigation: org admins set the policy deliberately; recovery codes are shown exactly once at enrollment; document the admin escape hatch (policy revert + per-member TOTP delete).

## Migration Plan

- **Deploy:** backend already shipped (verified implemented + tested); ship FE in order: limiter → MFA actions → `/authenticate` step → enrollment UI → policy UI. Magic-link path untouched and remains default; MFA is additive per org policy.
- **Rollback (Git):** revert FE action/UI changes; orgs with `mfa_policy` untouched remain magic-link-only.
- **Rollback (Stytch):** `UpdateMfaPolicy` is reversible (`OPTIONAL`/`ALL_ALLOWED` via the same endpoint); TOTP instances deletable via `DELETE /v1/b2b/totp/{totp_id}`.

## Open Questions

- Can a single member hold multiple TOTP registrations in Stytch B2B? (Assumption + duplicate guard; verify in test project.)
- Does Stytch apply its own attempt limiting to `recovery_codes/recover`? (Our limiter is the first line regardless.)

## Testing Strategy

- Go: `UpdateMfaPolicy` adapter unit tests already in place (success, Stytch 4xx, breaker-open 503); domain service test asserting no Stytch import.
- FE unit: `mfa-limiter` window tests; `authenticateTotp` with mocked Stytch client — TOTP success sets cookies, wrong code sets none, recovery success sets cookies directly from the recover response (assert NO chained authenticate call), rate-limit rejection returns generic error; `createTotp` duplicate guard (existing `totp_registration_id` → no create call).
- Component: `/authenticate` MFA step (challenge render, wrong-code error, recovery toggle, cookie set only on success); enrollment flow (qr_code render, verify, recovery-codes-once, existing-registration surfacing).
- E2E (Stytch test project): org with `REQUIRED_FOR_ALL` — login dead-end regression test now passes with TOTP step; recovery-code sign-in completes in one exchange; duplicate-enrollment retry creates no second instance.
