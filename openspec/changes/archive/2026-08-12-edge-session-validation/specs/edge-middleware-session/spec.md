## ADDED Requirements

### Requirement: Proxy validates JWT statelessly before allowing protected routes

The Next.js proxy (`next_b2b_starter/proxy.ts`; Next.js 16 middleware) SHALL validate the `stytch_session_jwt` cookie statelessly against the project's JWKS endpoint (cached locally with a TTL of 300 seconds or less) before allowing protected routes (`/dashboard/:path*`, `/settings/:path*`, `/api/protected/:path*`). Presence-only checks of session cookies are PROHIBITED.

#### Scenario: Valid JWT on protected route

- **WHEN** a request carries a valid `stytch_session_jwt` cookie for a protected route
- **THEN** the proxy SHALL validate the JWT locally (no synchronous Stytch API calls)
- **AND** SHALL forward `X-Stytch-Organization-Id` and `X-Stytch-Member-Id` from JWT claims plus `X-Forwarded-Auth: true` to downstream handlers
- **AND** SHALL allow the request to proceed

#### Scenario: Expired JWT on protected route

- **WHEN** a request carries an expired `stytch_session_jwt` cookie for a protected route
- **THEN** the proxy SHALL clear `stytch_session` and `stytch_session_jwt` cookies
- **AND** SHALL return a 302 redirect to `/auth` with the original path as `returnTo`

#### Scenario: Missing JWT on protected route

- **WHEN** a request has no session cookies for a protected route
- **THEN** the proxy SHALL return a 302 redirect to `/auth` with the original path as `returnTo`

#### Scenario: JWKS fetch failure falls back to cached keys

- **WHEN** the proxy needs JWKS keys and the fetch fails
- **AND** cached keys exist and are unexpired
- **THEN** validation SHALL continue with the cached keys
- **AND** if no usable keys exist, protected routes SHALL return HTTP 500

### Requirement: Backend fast path trusts only proxy-validated headers

The Go backend SHALL accept `X-Forwarded-Auth: true` with valid UUID `X-Stytch-Organization-Id` / `X-Stytch-Member-Id` headers only when `AUTH_TRUST_FORWARDED_AUTH` is enabled, and SHALL perform independent JWT verification when the header set is absent or malformed (existing behavior).

#### Scenario: Headers present and flag enabled

- **WHEN** `AUTH_TRUST_FORWARDED_AUTH=true` and the forwarded-auth header set is well-formed
- **THEN** the Go middleware SHALL authenticate from the headers (fast path)

#### Scenario: Headers absent

- **WHEN** `X-Forwarded-Auth` is absent or `false`
- **THEN** the Go middleware SHALL validate the JWT independently via local JWKS verification with Stytch API fallback
- **AND** MUST NOT trust user-supplied org/member headers

#### Scenario: Malformed forwarded headers

- **WHEN** `X-Forwarded-Auth: true` is present but `X-Stytch-Organization-Id` is not a valid UUID
- **THEN** the Go middleware SHALL reject with 401 and log a warning
