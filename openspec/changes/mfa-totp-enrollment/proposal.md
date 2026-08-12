# TOTP MFA Enrollment and Organization Policy

## Why

The sign-in path is email-magic-link only. `consumeMagicLink` already surfaces Stytch's `mfa_required` / `intermediate_session_token` response fields, but the `/authenticate` page has no MFA challenge step — if an organization enabled `mfa_policy: REQUIRED_FOR_ALL`, login would dead-end after primary auth. Stytch B2B supports TOTP (authenticator apps, with recovery codes) and SMS OTP as secondary factors; TOTP is the phishing-resistant, zero-cost option and is the modern SaaS baseline for sensitive orgs. `STYTCH_CONFIGURATION.md` lists "Enable MFA" as its top security recommendation.

## What Changes

- **Member enrollment:** settings → profile exposes "Set up authenticator app": `POST /v1/b2b/totp/create` → render the Stytch-issued `qr_code` + manual secret → verify with one code (`POST /v1/b2b/totp/authenticate` with the member session) → display one-time recovery codes (Stytch `recovery_codes`), shown exactly once. A member who already has a TOTP registration is surfaced for management instead of creating a duplicate.
- **Login continuation:** when primary auth returns `member_authenticated: false` + `intermediate_session_token` (org requires MFA), the `/authenticate` page shows a TOTP entry step; on success the action exchanges the intermediate session for a full session via `POST /v1/b2b/totp/authenticate` and sets the existing session cookies.
- **Recovery codes:** "Use a recovery code" path on the MFA step — `POST /v1/b2b/totp/recovery_codes/recover` with the `intermediate_session_token` returns the full session directly (the code is consumed in that call); cookies are set from that response. Recovery-code attempts are rate-limited per IP and per member.
- **Org policy:** settings → compliance/security lets org admins set `mfa_policy` (`OPTIONAL` default | `REQUIRED_FOR_ALL`) and restrict methods (`mfa_methods`/`allowed_mfa_methods`, TOTP default), persisted to Stytch via the organization update API (`PUT /v1/b2b/organizations`) through the Go backend.
- **SSOT:** TOTP secrets, recovery codes, and MFA state live exclusively in Stytch; local DB stores only the org's displayed policy mirror (read from Stytch org object on fetch). The mirror is display-only — it is never consulted for authorization decisions; Stytch is the sole enforcement point.

## Revision (council, 2026-08-12)

This revision addresses the council verdict (`VERDICT.md`, REJECTED). Required changes folded in:

1. **Recovery flow contract fixed:** `recovery_codes.recover` returns `session_token`/`session_jwt` directly (verified against `B2BRecoveryCodesRecoverResponse`); cookies are set from that response and `login_succeeded` is recorded — no chained "repeat TOTP authenticate" round trip.
2. **QR dependency removed:** `totp.create` returns a server-generated `qr_code` (verified against `B2BTOTPsCreateResponse`); the UI renders it directly — no QR library or hand-rolled SVG.
3. **Recovery-code rate limiting added:** per-IP and per-member sliding-window limiter (reusing the magic-link limiter pattern) bounds recovery-code attempts, with bounded `mfa_challenge_failed` audit detail.
4. **Enrollment idempotency specified:** the member object exposes `totp_registration_id` (verified); the flow surfaces an existing registration for management instead of creating a duplicate on retry.
5. **Authorization invariant stated:** the locally mirrored org policy is display-only and MUST NOT gate authorization; Stytch enforces MFA at session mint.

## Capabilities

### New Capabilities
- `multi-factor-auth`: TOTP enrollment, MFA challenge continuation at login, recovery codes, and per-organization MFA policy.

### Modified Capabilities
- None.

## Impact

- **Frontend:** new MFA step component on `/authenticate`; enrollment UI in settings profile; org policy UI in compliance section; new `authenticateTotp`/`createTotp` actions (`lib/actions/auth/mfa.ts`); MFA rate limiter (`lib/auth/mfa-limiter.ts`).
- **Backend:** org settings service gains `UpdateMfaPolicy` (calls Stytch organizations.update via the auth adapter, inside the existing circuit breaker); optional `GetMfaPolicy` read.
- **Dependencies:** none new — Stytch returns a ready-made `qr_code` image plus the manual secret; no QR rendering library required.
- **Stytch:** per-org `mfa_policy` + `allowed_mfa_methods` become managed tenant policy state.

## Rollback

- **Git:** revert action/UI/backend changes; orgs with `mfa_policy` untouched remain magic-link-only.
- **Stytch tenant policy state:** `UpdateMfaPolicy` is reversible — set policy back to `OPTIONAL`/`ALL_ALLOWED` via the same endpoint. Document the revert procedure in the change; enrollment records (TOTP instances) are per-member and can be removed via `DELETE /v1/b2b/totp/{totp_id}` if needed.

## Non-Goals

- NOT supporting email as an MFA factor (Stytch B2B does not allow it — secondary factors are `sms_otp` and `totp` only).
- NOT implementing SMS OTP in this change (cost + country allowlist complexity); `allowed_mfa_methods` restricts to `totp`; SMS can be added later.
- NOT storing MFA secrets, recovery codes, or session material locally (SSOT constitution).
- NOT adding step-up MFA for high-risk actions (billing, exports) in this change; policy-level login MFA only.

## Assumptions

- Stytch multi-registration semantics (can one member hold multiple TOTP instances?) are undocumented in the SDK types; the design guards against duplicates via `totp_registration_id` and the assumption is verified in the Stytch test project during E2E (recorded in tasks).
- Whether Stytch applies its own attempt limits to the recovery-code endpoint is undocumented; our per-IP/per-member limiter is the first line of defense regardless.
