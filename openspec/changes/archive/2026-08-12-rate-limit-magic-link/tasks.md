## 1. Limiter Module

- [x] 1.1 [FE-NEXT] Create `next_b2b_starter/lib/auth/magic-link-limiter.ts`: in-process sliding-window limiter with `checkMagicLinkRateLimit({ email, ip })`; keys `email:<normalized>` and `ip:<ip>`; env overrides `MAGIC_LINK_RATE_LIMIT_PER_EMAIL_PER_HOUR` (default 5) and `MAGIC_LINK_RATE_LIMIT_PER_IP_PER_HOUR` (default 20); prune stale windows on access; document single-instance assumption. Verification: `pnpm lint` passes; unit tests in 1.2 pass.

  Gate results:
  - `pnpm lint` → exit 0, 0 errors / 4 pre-existing warnings (baseline).
  - `pnpm exec vitest run magic-link-limiter` → 9/9 tests pass (see 1.2).

- [x] 1.2 [FE-NEXT] Add unit tests `magic-link-limiter.test.ts`: 5th send within window allowed, 6th blocked; window slide re-allows after 1h; email and IP keys independent; env overrides honored. Verification: `pnpm test -- magic-link-limiter` passes.

  Gate results:
  - `pnpm exec vitest run magic-link-limiter` → Test Files 1 passed, Tests 9 passed (5th/6th boundary, window slide via `vi.useFakeTimers`, email/IP key independence both directions, email normalization, per-email + per-IP env overrides, unparseable-env fallback).

## 2. Wire into Server Action

- [x] 2.1 [FE-NEXT] In `sendMagicLink.ts`, derive IP (`x-forwarded-for` first entry → `x-real-ip` → `127.0.0.1`), call `checkMagicLinkRateLimit` before `members.search`; on throttle return success-shaped ActionResult with `throttled: true` and NO Stytch calls. Verification: `pnpm lint`; `pnpm build` passes.

  Gate results:
  - `npx tsc --noEmit` → exit 0, clean.
  - `pnpm lint` → exit 0, 0 errors / 4 pre-existing warnings.
  - `pnpm build` → exit 0 (route list includes `/auth`).
  - Audit behavior preserved: `recordAuthAudit({ type: "magic_link_requested", … })` per org untouched; rate-limit check sits before `getStytchB2BClient()` / `members.search` / `loginOrSignup`.

- [x] 2.2 [FE-NEXT] Extend `ActionResult` type union or result shape to carry optional `throttled`; update `app/auth/page.tsx` to render the "too many requests" hint when `throttled` is true. Verification: `pnpm build`; manual: submit 6 emails → 6th shows hint, Stytch dashboard logs show 5 sends.

  Gate results:
  - `ActionResult` success variant extended with optional `throttled?: boolean`; `createActionSuccess(data, { throttled: true })` supports it.
  - `app/auth/page.tsx` renders "Too many requests — please try again later." with a non-destructive "Slow down a moment" alert (form view) / hint text (success view resend path) when `result.throttled`.
  - `npx tsc --noEmit` → clean; `pnpm build` → exit 0.
  - Manual (recorded, not executed): submit 6 emails for one address → 6th shows the throttle hint; Stytch dashboard would show only 5 `loginOrSignup` calls. Stytch dashboard check deferred-external (no tenant creds in this environment).

## 3. Verification Gate

- [x] 3.1 [FE-NEXT] `pnpm test` (limiter + auth-related suites) passes; `pnpm lint` passes; `pnpm build` passes.

  Gate results:
  - `pnpm exec vitest run magic-link-limiter lib/auth/audit.test.ts` → Test Files 2 passed, Tests 20 passed (9 limiter + 11 auth audit).
  - `pnpm lint` → exit 0, 0 errors / 4 pre-existing warnings.
  - `pnpm build` → exit 0 (single build at end of work; no `.next` lock contention).

- [x] 3.2 [OPS-GOV] Confirm no session/credential data stored by limiter (code review); `openspec validate rate-limit-magic-link` passes.

  Gate results:
  - Code review: limiter retains only `email:<normalized>` / `ip:<ip>` → `number[]` timestamps in a module-level `Map`; no tokens, credentials, session, or MFA material anywhere; state is in-memory only and cleared on process restart.
  - `openspec validate rate-limit-magic-link` → "Change 'rate-limit-magic-link' is valid" (exit 0).

**Archive deferred:** centralized verification phase per repo practice.
