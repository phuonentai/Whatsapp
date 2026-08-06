## Purpose

Define how the Next.js edge middleware validates Stytch session JWTs statelessly, protecting routes without synchronous backend calls.

## Requirements

### Requirement: Edge middleware validates JWT session statelessly

The system SHALL create a `middleware.ts` at `next_b2b_starter/middleware.ts` that validates `stytch_session_jwt` cookies using `@stytch/nextjs/b2b/edge` with no synchronous backend calls.

#### Scenario: Valid session on protected route

- **WHEN** a request with a valid `stytch_session_jwt` cookie hits a protected route (`/dashboard/:path*`, `/settings/:path*`, `/api/protected/:path*`)
- **THEN** the middleware SHALL validate the JWT locally using Stytch's edge client
- **AND** SHALL extract `organization_id` and `member_id` from the validated JWT
- **AND** SHALL add `X-Stytch-Organization-Id` and `X-Stytch-Member-Id` request headers
- **AND** SHALL add `X-Forwarded-Auth: true` header to signal pre-validation to downstream services
- **AND** SHALL allow the request to proceed to the next handler

#### Scenario: Missing JWT on protected route

- **WHEN** a request without a `stytch_session_jwt` cookie hits a protected route
- **THEN** the middleware SHALL redirect to `/login`
- **AND** SHALL return a 302 redirect response

#### Scenario: Invalid or expired JWT on protected route

- **WHEN** a request with an invalid or expired `stytch_session_jwt` cookie hits a protected route
- **THEN** the middleware SHALL clear the invalid session cookie
- **AND** SHALL redirect to `/login`

#### Scenario: Public routes pass through without validation

- **WHEN** a request hits a public route (`/login`, `/`, `/api/public/:path*`, `/api/webhooks/stytch`)
- **THEN** the middleware SHALL allow the request to proceed without JWT validation

### Requirement: JWKS public keys cached at edge with TTL limit

The edge middleware SHALL cache Stytch JWKS public keys locally with a TTL of 300 seconds or less. On JWKS fetch failure, the middleware SHALL continue using the cached keys for session validation.

#### Scenario: JWKS fetch succeeds

- **WHEN** the middleware initializes and fetches Stytch JWKS public keys
- **THEN** the keys SHALL be cached in the edge runtime
- **AND** subsequent requests SHALL use the cached keys without a new fetch

#### Scenario: JWKS fetch fails and cache is valid

- **WHEN** the middleware starts up
- **AND** the existing JWKS cache has not expired
- **AND** a new JWKS fetch fails
- **THEN** the middleware SHALL continue using the cached keys

#### Scenario: JWKS fetch fails and cache is empty or expired

- **WHEN** the middleware starts up
- **AND** the JWKS cache is empty or expired
- **AND** a JWKS fetch fails
- **THEN** the middleware SHALL return a 500 error for all protected routes requiring session validation

### Requirement: Protected route matcher configuration

The middleware SHALL protect the following route patterns: `/dashboard/:path*`, `/settings/:path*`, `/api/protected/:path*`.

#### Scenario: Configuration matches expected routes

- **WHEN** the middleware `config.matcher` is checked
- **THEN** it SHALL match `/dashboard/:path*`, `/settings/:path*`, and `/api/protected/:path*`
- **AND** SHALL NOT match `/login`, `/`, or `/api/public/:path*`
