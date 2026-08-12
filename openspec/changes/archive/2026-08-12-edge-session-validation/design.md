# Edge Session Validation — Design

## Context

Next.js 16 renamed `middleware.ts` → `proxy.ts`; the file exists at `next_b2b_starter/proxy.ts` and currently performs SEO redirects + cookie-presence checks. The `@stytch/nextjs/b2b/edge` package exposes `StytchB2BEdgeClientProvider` / `validateSessionJWT` helpers that validate session JWTs locally with cached JWKS.

## Decisions

### D1 — Stateless validation via locally verified JWKS

- In `proxy.ts`, for protected routes, read `stytch_session_jwt` cookie and validate it locally: RS256 signature verified against the project JWKS (`{base}/v1/b2b/sessions/jwks/{project_id}`, the same endpoint the Stytch SDK uses) via Web Crypto, with the standard `exp`/issuer checks. Note: `@stytch/nextjs/b2b/edge` (originally referenced by this spec) does not exist in any published `@stytch/nextjs` version; the spec's behavioral requirements are implemented directly.
- JWKS keys are cached at the edge with a TTL ≤ 300s (`JWKS_CACHE_TTL_MS = 300_000`) to honor the constitution invariant `TTL <= 300s`; no synchronous Stytch API calls from the edge runtime (spec: "no synchronous backend calls").

### D2 — Header emission and forwarding

- On valid JWT: set `X-Stytch-Organization-Id`, `X-Stytch-Member-Id`, `X-Forwarded-Auth: true` on the rewritten request (`NextResponse.next({ request: { headers } })`) so server components, route handlers, and Go API calls receive them.
- Order: validate → emit headers → allow. Missing JWT → clear cookies → 302 to `/auth?returnTo=<path>`. Invalid/expired JWT → clear cookies → 302 to `/auth`.

### D3 — Mock auth compatibility

- Preserve the existing `AUTH_MOCK_ENABLED` branch (E2E-only) ahead of real validation so `X-Test-Org-ID` mock sessions continue to work; the mock branch bypasses JWT validation only when explicitly enabled.

### D4 — Go TrustForwardedAuth enablement

- Keep the Go fast path env-gated: `AUTH_TRUST_FORWARDED_AUTH=true` in environments where the Next proxy is the only ingress and rewrites strip/override client-supplied headers (document in deployment config). The Go middleware already 401s on malformed header combos (`middleware.go:125-135`) and falls back to independent verification when `X-Forwarded-Auth` is absent.

## Stytch Boundary

- Fallback/circuit-breaker: JWKS fetch failure at edge → continue with cached keys (existing spec scenario). Cache empty + fetch failure → 500 for protected routes (existing spec scenario). No outbound session API calls from the edge.
- State-transition invariants (Go): `X-Forwarded-Auth: true` + valid UUID headers ⇒ authenticated from headers; absent/false ⇒ independent JWT validation; true + malformed UUID ⇒ 401 + warning log. These already exist and are tested; the change only makes the edge actually produce the headers.

## Security Invariants

- The edge MUST NOT trust client-supplied `X-Forwarded-Auth`; only the proxy's own validated claims may be forwarded, and the Go middleware independently validates when the header set is absent (existing behavior).
- Session material is never stored; cookies remain httpOnly `stytch_session` + JS-readable `stytch_session_jwt` per current constants.

## Testing Strategy

- Proxy unit tests: valid JWT → headers present + next(); missing JWT → 302 + cookie cleared; expired/invalid JWT → 302 + cookie cleared; public routes pass through; mock-auth branch unchanged when enabled.
- Go: existing `middleware_test.go` forwarded-auth scenarios already cover header acceptance/rejection; add one test asserting absence of headers triggers full verification path (if not already present).
- E2E `proxy.spec.ts` (e2e/specs) asserts expired JWT cookie is rejected — update to expect the new 302/clear behavior.
