## 1. Frontend Resilience Seam (breaker + limiter) — prerequisite

- [ ] 1.1 [FE-NEXT] Add `lib/auth/stytch/breaker.ts`: stateful circuit breaker (threshold 5 consecutive failures, open timeout 10s, half-open probe 2 — mirroring the Go adapter contract) and `runWithBreaker(fn)`; map breaker-open and Stytch 5xx to a structured `passkey_unavailable` error (503-style). Unit tests: trips at 5, opens/recovers per timeout, half-open probe 2, 503 mapping. Verification: `pnpm lint`; `pnpm test -- breaker` passes; `pnpm build` passes.

- [ ] 1.2 [FE-NEXT] Add `lib/auth/passkey-limiter.ts` reusing the magic-link limiter's in-process sliding-window pattern: per-email and per-IP windows, env-configurable (`PASSKEY_AUTH_RATE_LIMIT_PER_EMAIL_PER_HOUR` default 10, `PASSKEY_AUTH_RATE_LIMIT_PER_IP_PER_HOUR` default 30), single-instance assumption documented. Unit tests: per-email/IP window enforcement. Verification: `pnpm test -- passkey-limiter` passes.

## 2. Passkey Server Actions

- [ ] 2.1 [FE-NEXT] Create `next_b2b_starter/lib/actions/auth/passkeys.ts`: `createPasskeyRegistration(sessionJwt)`, `completePasskeyRegistration(credentialJson)` — breaker-wrapped Stytch `webauthn.register.start`/`register`; `startPasskeyAuthentication(email)` — server-side re-resolution via `members.search` (never accepts opaque client memberId), rate-limited (1.2) and breaker-wrapped; `completePasskeyAuthentication(assertionJson, sessionDurationMinutes)` — sets session cookies ONLY on `member_authenticated: true`, propagates `intermediateSessionToken`/`mfaRequired`/`primaryRequired` when gated; `listPasskeys()` / `deletePasskey(memberWebauthnId)` — derive member/org from the verified session (ignore client-supplied IDs), delete idempotent (404 → success). Verification: `pnpm lint`; `pnpm build` passes.

- [ ] 2.2 [FE-NEXT] Unit tests with mocked Stytch client: register start/complete round-trip; authenticate success sets cookies; authenticate failure sets no cookies; `mfa_required` → intermediate token passthrough + no cookies; breaker-open → 503 `passkey_unavailable` + no cookies; delete ignores client-supplied scoping and treats 404 as success. Verification: `pnpm test -- passkeys` passes.

## 3. Registration UI (Settings → Profile → Security)

- [ ] 3.1 [FE-NEXT] Passkey section: "Add a passkey" → browser `navigator.credentials.create` (conditional UI) wrapped in a 60s abort timeout → complete → list registered passkeys with delete. Verification: `pnpm build`; component tests pass.

## 4. Sign-In Branch on /auth

- [ ] 4.1 [FE-NEXT] After email membership resolution, render "Sign in with passkey" when applicable; run `navigator.credentials.get` wrapped in a 120s abort timeout; on success redirect to destination; on user-cancel → silent magic-link fallback (no failure noise); on Stytch/network failure → structured error + magic-link fallback; `mfa_required` response → route to the MFA challenge step with the intermediate session token (no cookies). Verification: `pnpm build`; page tests cover passkey success, user-cancel fallback, ceremony failure fallback, MFA routing.

## 5. Config + Docs

- [ ] 5.1 [OPS-GOV] Document RP ID selection and passkeys-product enablement in `STYTCH_CONFIGURATION.md` (RP ID immutable — set once before rollout); document breaker/limiter env vars. Verification: doc review; `openspec validate passkeys-sign-in` passes.

## 6. Verification Gate

- [ ] 6.1 [FE-NEXT] `pnpm lint`, `pnpm build`, `pnpm test` pass.
- [ ] 6.2 [OPS-GOV] Confirm no local storage of passkey material (code review) and that breaker-open paths issue no session cookies; `openspec validate passkeys-sign-in` passes.

## Phase 0 baseline checkpoint (2026-08-11, repo-wide active-changes run)

- [x] Repo-wide baseline recorded BEFORE further implementation work on this change (working tree: ~330 modified files across both apps from sibling in-flight changes):
  - `go build ./...` PASS (exit 0) · `go vet ./...` PASS · `go test ./...` PASS (all packages, exit 0) — go-b2b-starter
  - `npx tsc --noEmit` PASS · `pnpm lint` PASS (0 errors / 4 pre-existing warnings) · `pnpm build` PASS — next_b2b_starter
  - Context: this baseline anchors later verification gates — failures introduced by this change are distinguishable from pre-existing tree state.
