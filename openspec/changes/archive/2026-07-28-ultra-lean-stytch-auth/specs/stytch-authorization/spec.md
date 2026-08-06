## MODIFIED Requirements

### Requirement: Go backend accepts pre-validated JWT from edge middleware

The Go backend SHALL accept the `X-Forwarded-Auth: true` header set by the Next.js edge middleware. When this header is present and the `X-Stytch-Organization-Id` and `X-Stytch-Member-Id` headers carry valid UUIDs, the Go session middleware MAY skip redundant Stytch API token introspection for performance.

The system SHALL still validate the JWT signature independently if `X-Forwarded-Auth` is absent or the headers are malformed. The Go backend MUST NOT trust unvalidated headers.

#### Scenario: Request arrives with valid X-Forwarded-Auth headers

- **WHEN** a request arrives at the Go API
- **AND** `X-Forwarded-Auth: true` is set
- **AND** `X-Stytch-Organization-Id` and `X-Stytch-Member-Id` are valid UUIDs
- **THEN** the Go session middleware SHALL use these headers as the authenticated context
- **AND** MAY skip calling Stytch API for token introspection
- **AND** the `organization_id` and `member_id` SHALL be set on the request context

#### Scenario: Request arrives without X-Forwarded-Auth

- **WHEN** a request arrives at the Go API
- **AND** `X-Forwarded-Auth` is absent or `false`
- **THEN** the Go session middleware SHALL validate the JWT or session token independently via the Stytch API (existing behavior)
- **AND** MUST NOT trust any user-supplied `organization_id` or `member_id` headers

#### Scenario: X-Forwarded-Auth is present but headers are malformed

- **WHEN** a request arrives at the Go API
- **AND** `X-Forwarded-Auth: true` is set
- **AND** `X-Stytch-Organization-Id` is not a valid UUID
- **THEN** the Go session middleware SHALL reject the request with a 401 Unauthorized response
- **AND** SHALL log a warning about malformed edge-auth headers

## ADDED Requirements

### Requirement: Log mismatches between edge and Go validation

The Go backend SHALL log a warning when the edge middleware's `X-Forwarded-Auth` and the Go backend's independent Stytch API validation produce different results (e.g., JWT valid at edge but rejected by Stytch API).

#### Scenario: Edge says valid, Go says invalid

- **WHEN** a request arrives with `X-Forwarded-Auth: true`
- **AND** the Go backend attempts independent validation via Stytch API
- **AND** Stytch API returns an invalid/expired session response
- **THEN** the Go backend SHALL log a warning with the `stytch_member_id`, `stytch_organization_id`, and Stytch API error
- **AND** SHALL return a 401 Unauthorized
