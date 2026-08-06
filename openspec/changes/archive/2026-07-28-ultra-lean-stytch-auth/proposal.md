## Why

The current Stytch B2B integration relies heavily on the Go backend for session management and auth flows, requiring custom auth code, custom admin UIs, and synchronous backend calls for every protected route. Eliminating ~90% of this custom code by shifting auth responsibilities to Stytch's pre-built frontend SDKs and stateless edge middleware reduces the system to under 100 lines of authentication code while maintaining the dual SSOT architecture invariants.

## What Changes

- **Drop custom login/signup forms**: Replace with pre-built `<StytchB2B />` and `<B2BDiscovery />` components from `@stytch/nextjs/b2b` on auth pages
- **Stateless edge middleware sessions**: Validate JWT cookies at the Next.js edge layer using local JWKS key cache — no database lookup, no Go backend round-trip per request. **BREAKING**: Changes how protected routes are gated; existing Go middleware session validation must accept pre-verified edge tokens
- **Zero-code admin portal**: Replace custom member management and SSO admin UIs with `<AdminPortalMemberManagement />` and `<AdminPortalSSO />` from `@stytch/nextjs/b2b`
- **Lean multi-tenant data isolation**: Enforce `organization_id`-scoped queries at every database access layer using the JWT claim, with optional RLS policies for defense-in-depth
- **Constitutional guardrails**: Add formal verification anchors for tenant isolation and stateless auth, circuit breaker fallback on JWKS fetch failures, and a compliance audit gate enforcing the `< 100 LOC` auth code invariant

## Capabilities

### New Capabilities
- `stytch-nextjs-components`: Pre-built Stytch B2B UI components for login, discovery, SSO configuration, and member management rendered via `@stytch/nextjs/b2b`
- `edge-middleware-session`: Stateless JWT session validation at the Next.js edge layer using locally cached JWKS public keys, with no synchronous backend calls
- `lean-data-isolation`: Every tenant-specific database query MUST be scoped by `organization_id` from the verified JWT claim, enforced at the repository/query layer with optional PostgreSQL RLS

### Modified Capabilities
- `stytch-authorization`: Extend the existing Go backend RBAC/permission resolution to acknowledge that the Next.js edge middleware independently validates session JWTs; the Go backend must not reject requests with valid JWT claims already verified at the edge
- `crm-frontend`: Auth pages (login, signup, settings) are replaced with pre-built Stytch components; protected route gating moves from server-side checks to edge middleware

## Non-Goals

- Local password hashing or credential storage
- Custom session tables or databases
- Replacing Stytch RBAC as the source of truth for permissions
- Modifying the existing Go backend Stytch v18 SDK integration patterns

## Impact

- **Frontend**: `next_b2b_starter/` — new dependency `@stytch/nextjs` and `@stytch/vanilla-js`; auth pages (`/login`, `/settings`) rewritten to use pre-built components; new `middleware.ts` in root
- **Backend**: `go-b2b-starter/` — Go session middleware must accept valid JWTs already verified at the edge via `X-Forwarded-Auth` header; no breaking changes to existing Stytch API calls (v18 SDK)
- **Database**: Optional PostgreSQL RLS policies on tenant-scoped tables; no schema changes required
- **Constitution**: New formal verification anchors (FVA-TENANT-ISOLATION, FVA-STATELESS-AUTH), circuit breaker rules for JWKS fetch, and a `max_auth_lines_of_code` compliance gate
