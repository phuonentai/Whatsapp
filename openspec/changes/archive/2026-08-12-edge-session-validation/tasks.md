## 1. Proxy Stateless Validation

- [x] 1.1 [FE-NEXT] Rewrite `next_b2b_starter/proxy.ts`: for protected routes (`/dashboard/:path*`, `/settings/:path*`, plus `/api/protected/:path*` if present), validate `stytch_session_jwt` statelessly (JWKS cache TTL ≤ 300s) and forward `X-Stytch-Organization-Id`, `X-Stytch-Member-Id`, `X-Forwarded-Auth: true`; missing/invalid/expired → clear `stytch_session` + `stytch_session_jwt` cookies and 302 to `/auth?returnTo=…`; public routes (`/auth`, `/authenticate`, `/signup`, `/api/auth`) pass through; `AUTH_MOCK_ENABLED` branch preserved. Verification: `pnpm lint`; `pnpm build` passes. NOTE: `/api/protected/:path*` does not exist in this app — matcher covers `/dashboard` + `/settings` (per task "if present"). Mechanism note: `@stytch/nextjs/b2b/edge` does not exist in any published `@stytch/nextjs` version; stateless RS256 verification against the project JWKS is implemented directly with Web Crypto (delta spec/design updated to match).

- [x] 1.2 [FE-NEXT] Add proxy unit tests (`proxy.test.ts`): valid JWT → headers emitted + `next()`; missing JWT → 302 to `/auth` with returnTo; expired/invalid JWT → cookie cleared + 302; public route → pass-through; mock branch still grants access when `AUTH_MOCK_ENABLED=true`. Verification: `pnpm exec vitest run proxy` passes (11/11).

- [x] 1.3 [FE-NEXT] Update `e2e/specs/proxy.spec.ts` expectations for expired/invalid JWT to the new clear+302 behavior. Verification: `pnpm exec playwright test e2e/specs/proxy.spec.ts --list` lists 8 tests cleanly (syntax-clean; full e2e run is Phase 5).

## 2. Go Fast Path Enablement

- [x] 2.1 [BE-INFRA] Add `AUTH_TRUST_FORWARDED_AUTH` env (default false) to backend config and wire into `MiddlewareConfig.TrustForwardedAuth`; keep default off. Verification: `go build ./internal/modules/auth/...` passes; `go test ./internal/modules/auth/...` (existing tests, flag unset) passes — behavior unchanged.

- [x] 2.2 [BE-INFRA] Add test asserting that when `X-Forwarded-Auth` is absent, `RequireAuth` performs independent token verification (no header trust). Verification: `go test ./internal/modules/auth/...` passes (`TestRequireAuth_NoForwardedAuthHeaderUsesTokenVerification`, `TestRequireAuth_NoForwardedAuthHeaderInvalidTokenRejected`).

## 3. Verification Gate

- [x] 3.1 [FE-NEXT] `pnpm lint` (0 errors / 4 warnings — baseline), `npx tsc --noEmit` (clean), `pnpm exec vitest run proxy` (11/11) pass; `pnpm build` recorded at end of phase (see Gate Results).
- [x] 3.2 [BE-INFRA] `go test ./internal/modules/auth/...` passes; `go build ./...` passes (green after concurrent billing/organizations agents landed their in-flight edits).
- [x] 3.3 [OPS-GOV] `openspec validate edge-session-validation` passes.

## Gate Results

- 1.1: PASS — `pnpm lint` → 0 errors / 4 warnings (baseline); `npx tsc --noEmit` → clean; `pnpm build` pending shared-`.next` build at end of phase (lock-shared with other agents; constraint: run ONCE at end).
- 1.2: PASS — `pnpm exec vitest run proxy` → 11/11 tests passed (valid-JWT headers, client-header strip, missing-JWT 302+returnTo, expired/invalid/malformed/unknown-key 302+cookie-clear, JWKS-unavailable 500, public pass-through, mock branch).
- 1.3: PASS (syntax) — `pnpm exec playwright test e2e/specs/proxy.spec.ts --list` → 8 tests listed; cookie-clear assertions added for expired + malformed JWT tests; full e2e run is Phase 5 per constraints.
- 2.1: PASS — `go build ./internal/modules/auth/...` OK; `go test ./internal/modules/auth/...` (all existing + new) pass with `AUTH_TRUST_FORWARDED_AUTH` unset (default off).
- 2.2: PASS — new tests `TestRequireAuth_NoForwardedAuthHeaderUsesTokenVerification` + `TestRequireAuth_NoForwardedAuthHeaderInvalidTokenRejected` pass; stub provider proves `VerifyToken` is called and forwarded headers are ignored when `X-Forwarded-Auth` absent.
- 3.1: PASS — lint 0/4, tsc clean, proxy vitest 11/11, `pnpm build` exit 0 (proxy compiled as "ƒ Proxy (Middleware)").
- 3.2: PASS — `go build ./...` rc=0; `go test ./internal/modules/auth/...` ok. (Earlier failure in `internal/modules/billing/infra/repositories/subscription_repository.go:93` was a concurrent MPS-seams agent's in-flight sqlc regen; cleared once they landed.)
- 3.3: PASS — `openspec validate edge-session-validation` → "Change 'edge-session-validation' is valid".

## Archive Decision

**Archive deferred:** centralized verification phase per repo practice. On archive: re-run `go build ./...` and `go test ./internal/modules/auth/...` (green as of this change), and confirm the delta fold replaces the living `edge-middleware-session` spec's `@stytch/nextjs/b2b/edge` reference with the JWKS-based mechanism.
