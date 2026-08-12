# Council Verdict — mfa-totp-enrollment

STATUS: REJECTED

## Premise verification notes (facts checked against the codebase and Stytch SDK)

- Go backend for org policy: **verified implemented** — `domain/mfa_policy.go`, `stytch_organization_repository.go` (implements `MfaPolicyUpdater` via `PUT /v1/b2b/organizations` behind the shared breaker), `organization_service.go`, `mfa_policy_handler.go`, route `PUT /api/organizations/mfa-policy` gated by `auth.RequirePermissionFunc("org", "manage")`, with breaker-open→503 tests.
- Circuit breaker: **verified present** in `internal/platform/stytch/circuit_breaker.go` — `NewCircuitBreaker` defaults threshold 5 / cooldown 10s / half-open probes 2; `Client.Run()` guards outbound Stytch calls and fails fast with `ErrCircuitOpen`; `BreakerState()` diagnostics exist. The D3 "existing two-tier breaker" premise is TRUE.
- Stytch B2B SDK contracts (`node_modules/stytch/types/lib/b2b/`):
  - `B2BRecoveryCodesRecoverResponse` returns `session_token`/`session_jwt` **directly** — the recover endpoint itself completes the MFA flow and consumes the code. It has no continuation/intermediate-session path.
  - `B2BTOTPsCreateResponse` includes `qr_code` (server-generated QR image) in addition to `secret` and `recovery_codes`.
  - `B2BTOTPsAuthenticateResponse` returns the session directly (no `member_authenticated` field; success = session, failure = thrown error).
- `app/authenticate/page.tsx` exists (magic-link consume page); `mfa_challenge_passed`/`mfa_challenge_failed` exist in the audit taxonomy.

## Per-persona findings

### 1. Staff Security Engineer

- **[MEDIUM] Recovery-path contract is wrong (D1).** The design says `recovery_codes/recover` → intermediate session → "repeat TOTP authenticate". Per `B2BRecoveryCodesRecoverResponse` the recover call returns `session_token`/`session_jwt` directly; the intermediate token is single-use and consumed by the recover call. Following the design verbatim produces a broken recovery sign-in (second round trip would fail on a consumed token). Required: set cookies directly from the recover response; record `login_succeeded` after it.
- **[MEDIUM] No rate limiting on the recovery-code path.** Recovery codes are static bearer secrets. No per-IP/per-member attempt bounds are specified (the magic-link limiter pattern exists and should be reused). Wrong-TOTP attempts are naturally bounded by 30s windows; recovery attempts are not.
- **[LOW] `authenticateTotp({intermediateSessionToken, organizationId, memberId, ...})`** accepts client-supplied member/org IDs. Acceptable residual risk — Stytch binds the intermediate token to the member — but the design should state these IDs are context only and the token is the authority.
- **[OK] RBAC verified:** policy route gated by `org:manage`; enrollment is self-service; no local MFA material; audit `detail` bounded; intermediate tokens never logged/persisted.

### 2. Staff DBA

- **[CLEAN] No schema changes.** No migrations, indexes, or transaction-boundary concerns. The org policy mirror is display-only (read from the Stytch org object on fetch).
- **[LOW] Ensure the displayed policy mirror is never used for authorization decisions** — Stytch remains the sole enforcement point (MFA enforcement happens at session mint). State this explicitly so a future reader does not consult the mirror in an authz path.

### 3. Staff SRE

- **[MEDIUM] Enrollment lacks idempotency/duplicate guard.** `totp.create` has no idempotency key; a retry after a successful create+verify can create a second TOTP registration (default `expiration_minutes` 60). Either guard the completed state in the FE state machine / surface the existing registration, or explicitly accept Stytch multi-registration semantics — verify in the test project and record the decision.
- **[LOW] Missing MFA failure observability.** No metrics/logging specified for failed TOTP or recovery attempts (needed for incident response per STYTCH_CONFIGURATION.md "Monitor failed attempts"); breaker diagnostics exist (`BreakerState()`) and should be surfaced on the 503 path.
- **[OK] Breaker/fallback verified and specified:** breaker-open → structured 503, policy unchanged; rollback is reversible (`UpdateMfaPolicy` back to `OPTIONAL`; per-member TOTP instances deletable).

## Required design changes (numbered)

1. **Fix the recovery flow contract:** `recovery_codes.recover` returns `session_token`/`session_jwt` directly (verified against `B2BRecoveryCodesRecoverResponse`) — set cookies from that response, record `login_succeeded`, and remove the "repeat TOTP authenticate" step (the intermediate token is single-use). Update the delta spec's recovery scenario to match.
2. **Remove the QR-library dependency:** `totp.create` returns a server-generated `qr_code` (verified `B2BTOTPsCreateResponse`) — render it directly; drop the "minimal QR lib or SVG" plan from design D2, tasks 2.2, and the proposal's Dependencies section.
3. **Add rate limiting for the recovery-code path** (per-IP and per-member sliding window, reusing the magic-link limiter pattern), with bounded `mfa_challenge_failed` audit detail on failed recovery attempts; state how TOTP attempt failures are bounded and monitored.
4. **Specify enrollment idempotency:** guard against creating a second TOTP instance when a member retries a completed enrollment (FE state machine guard and/or surfacing the existing registration); verify Stytch's multi-registration semantics in the test project and record the decision in tasks.
5. **State the authorization invariant:** the locally mirrored/displayed org policy is display-only and MUST NOT be consulted for authorization decisions — Stytch remains the enforcement point.
