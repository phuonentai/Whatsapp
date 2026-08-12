# Council Verdict — passkeys-sign-in

STATUS: REJECTED

## Premise verification notes (facts checked against the tree)

- Existing pattern claims (D1/D2/D3) hold: `consumeMagicLink` sets `stytch_session` / `stytch_session_jwt` only after `member_authenticated`, propagates `intermediate_session_token` + `mfa_required`, and records best-effort audit events; email-first `members.search` validation exists in `send-magic-link.ts`.
- **Contradiction found:** the "Stytch Boundary & Fallback" table claims all calls flow through "the existing auth adapter circuit breaker (threshold 5 / timeout 10s / half-open probe 2)". `getStytchB2BClient()` (`lib/auth/stytch/server.ts`) constructs `new Stytch.B2BClient()` directly with **no breaker wrapper** in the Next.js path. The two-tier breaker exists only in the Go backend (`go-b2b-starter/internal/infrastructure/auth/stytch`), which this change explicitly declares out of scope ("Backend: none required"). The fallback table is therefore not implementable as specified.
- `magic-link-limiter.ts` exists but covers only the magic-link send path; no rate-limit design is specified for the new passkey actions.

## Per-persona findings

### 1. Staff Security Engineer

- **[HIGH] Unverifiable circuit-breaker premise** — the design asserts a fallback contract (breaker-open → structured 503, no session issuance) for a mechanism that does not wrap the Stytch B2B client used by the new server actions. Security controls cannot be specified against a mechanism that is absent from the path.
- **[MEDIUM] Unauthenticated challenge-start spam** — `startPasskeyAuthentication({ memberId, organizationId })` accepts client-supplied IDs. An attacker who knows (or enumerates) a member email can invoke `webauthn.authenticate.start` for that member repeatedly (challenge churn / DoS against a victim's pending-challenge state, which can invalidate a legitimate in-flight start). No rate limit (per-email / per-IP) is specified for the passkey actions, unlike the existing magic-link path.
- **[MEDIUM] Ownership scoping of list/delete** — `listPasskeys(sessionJwt)` / `deletePasskey(memberWebauthnId)` must derive member/org from the authenticated session, never from client-supplied IDs, or a member can delete another member's WebAuthn credential. The design does not state this binding.
- **[LOW] MFA/primary composition detail** — D2 routes `mfa_required` to the TOTP step but does not state that Stytch returns `member_authenticated: false` + `intermediate_session_token` on the passkey authenticate call and that the intermediate token must be propagated to the existing MFA challenge step (mirroring `consumeMagicLink`). Also unaddressed: `primary_required` handling for orgs with SSO.

### 2. Staff DBA

- **[CLEAN] No DB changes** — zero schema changes; passkey material stays in Stytch. Delta spec correctly forbids local passkey records and forbids writing credential material to the audit stream.
- **[LOW] Audit-write non-blocking guarantee** — the passkey login_succeeded/failed audit writes must follow the existing best-effort, never-block-the-auth-outcome pattern of `consumeMagicLink`; state this explicitly in the design so the invariant survives implementation.

### 3. Staff SRE

- **[HIGH] Fallback contract not implementable** — see Security [HIGH]. The design must either (a) add a frontend circuit-breaker wrapper (matching threshold 5 / 10s / half-open 2) around the Stytch B2B client used by the passkey actions, or (b) restate the fallback table in terms of actual SDK-error handling with explicit retry/abort semantics — and spec it in the delta.
- **[MEDIUM] Browser ceremony timeout/abort** — `navigator.credentials.create/get` can hang indefinitely (user leaves tab open). No timeout/abort signal is specified, and there is no defined distinction between user-cancel vs Stytch failure vs network failure for observability/alerting (a passkey outage must not be masked as user cancels).
- **[LOW] Rollback nuance** — "disable the passkeys product" and "delete member WebAuthn instances" are both per-member/per-dashboard operations, not bulk; the design should state whether disabling the product revokes already-enrolled credentials (verify Stytch semantics) and that RP ID immutability makes the domain a one-way door.

## Required design changes (numbered)

1. **Resolve the breaker contradiction.** Remove the claim that passkey calls flow through an "existing auth adapter circuit breaker" in the frontend path (none exists in `lib/auth/stytch/`); either add a frontend breaker/retry wrapper around the Stytch B2B client used by the new actions (threshold 5 / timeout 10s / half-open probe 2) or rewrite the Stytch Boundary & Fallback table with the real failure semantics (raw SDK error → structured 503, no session issuance, magic-link fallback). Update the delta spec's fallback requirement to match the chosen mechanism.
2. **Specify rate limiting for the new passkey server actions** (`startPasskeyAuthentication`, `createPasskeyRegistration`), reusing the magic-link limiter pattern (per-email and per-IP, in-process with documented single-instance assumption) to prevent challenge-start spam against valid member emails.
3. **Specify server-side ownership binding:** list/delete passkey operations SHALL derive `organization_id` / `member_id` from the authenticated session, not from request bodies; `startPasskeyAuthentication` SHALL re-resolve/validate the member server-side rather than trusting client-supplied IDs.
4. **Complete the MFA/primary composition contract:** when passkey `authenticate` returns `member_authenticated: false`, propagate `intermediate_session_token` + `mfa_required` (+ `primary_required` if applicable) to the existing MFA challenge step — mirror `consumeMagicLink` exactly; add a scenario to the delta spec for intermediate-token handoff.
5. **Add ceremony + observability requirements:** specify an abort/timeout for the browser credential ceremony, define distinct handling/telemetry for user-cancel vs Stytch failure vs network failure, and state that passkey audit writes are best-effort and non-blocking.
