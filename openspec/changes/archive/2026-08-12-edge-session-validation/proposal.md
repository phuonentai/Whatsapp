# Edge Session Validation (Stateless JWT Check in proxy.ts)

## Why

The living spec `edge-middleware-session` requires stateless JWT validation at the edge (JWKS cache TTL ≤ 300s, `X-Forwarded-Auth` headers, redirect + cookie clear on invalid/expired JWTs). The actual implementation does not do this:

- `next_b2b_starter/proxy.ts` (Next.js 16's middleware) checks only **cookie presence** (`stytch_session` or `stytch_session_jwt`) before allowing `/dashboard` and `/settings`. An expired or invalid JWT cookie still passes the proxy.
- The proxy never emits `X-Forwarded-Auth: true` / `X-Stytch-Organization-Id` / `X-Stytch-Member-Id`, so the Go backend's `TrustForwardedAuth` fast path (`internal/modules/auth/middleware.go:125`) is dead code — every API request pays full token verification.
- Server components call `sessions.authenticateJwt` (Stytch API) on every authenticated render, adding latency and a hard dependency on Stytch availability for page loads.

## What Changes

- Upgrade `proxy.ts` to validate `stytch_session_jwt` statelessly using `@stytch/nextjs/b2b/edge` (JWKS cached with TTL ≤ 300s; fall back to cached keys on fetch failure per the existing spec).
- On valid JWT: extract `organization_id` / `member_id` from claims, forward as `X-Stytch-Organization-Id` / `X-Stytch-Member-Id` plus `X-Forwarded-Auth: true` to downstream (Next server + Go via rewrites/API calls).
- On missing/expired/invalid JWT: clear the session cookies and 302 to `/auth?returnTo=…` (matching the spec).
- Enable `TrustForwardedAuth` in Go config where the edge is the sole ingress (env-gated), making the dead fast path live.
- No change to Go JWT verification: when `X-Forwarded-Auth` is absent or malformed, the backend still validates independently (existing behavior).

## Capabilities

### New Capabilities
- None.

### Modified Capabilities
- `edge-middleware-session`: proxy (Next.js 16 `proxy.ts`, formerly `middleware.ts`) SHALL validate the JWT statelessly before allowing protected routes and SHALL emit forwarded-auth headers; cookie-presence-only checks are prohibited.

## Impact

- **Frontend:** `proxy.ts` rewrite; add `@stytch/nextjs/b2b/edge` usage (already a dependency); new proxy unit tests.
- **Backend:** enable `TrustForwardedAuth` via env (`AUTH_TRUST_FORWARDED_AUTH=true`) in deployment config; no Go code change required (path exists and is tested in `middleware_test.go`).
- **Dependencies:** none new (`@stytch/nextjs` already present).
- **Stytch:** no tenant policy changes; JWKS keys fetched at edge (≤300s TTL), reducing API introspection calls.

## Rollback

- **Git:** revert `proxy.ts` and the env flag; presence-only behavior is restored.
- **Stytch tenant policy state:** none modified. Edge validation is purely local key verification; no Stytch-side state changes.

## Non-Goals

- NOT introducing opaque-session (server-side) validation at the edge; JWT stateless validation is the Stytch-documented pattern for edge runtimes.
- NOT removing the Go backend's independent verification path; edge validation is a fast-path optimization and defense-in-depth, not a trust replacement.
- NOT changing route matcher semantics beyond the existing spec (`/dashboard/:path*`, `/settings/:path*`, `/api/protected/:path*` public exclusions).
- NOT storing credentials or session material locally (SSOT compliance).
