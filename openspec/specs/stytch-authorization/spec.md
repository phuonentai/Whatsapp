stytch-go from v16 to v18

The system SHALL use `github.com/stytchauth/stytch-go/v18` as the Stytch Go SDK dependency.

All Stytch API client calls in the Go backend SHALL be compatible with the v18 SDK package structure and method signatures.

The `LoginOrSignup` magic link method SHALL NOT be called during organization bootstrap. Member invite emails SHALL be sent via `Members.Create` with `SendInvite: true`.

#### Scenario: stytch-go v18 compiles and all existing Stytch API calls work

- **WHEN** the Go module is updated to use stytch-go v18
- **AND** `go build` is executed
- **THEN** compilation succeeds without import or signature errors
- **AND** organization creation, member management, RBAC, and session operations remain functional

## MODIFIED Requirements

### Requirement: Permission resolution from Stytch RBAC policy

The system SHALL resolve all role-to-permission mappings exclusively from the Stytch RBAC policy API. Hardcoded role→permission maps in Go code MUST NOT be used as a source of truth for authorization decisions.

The Stytch RBAC policy MUST be cached in Redis with a 5-minute TTL. On cache hit, the cached policy MUST be used without calling the Stytch API. On cache miss, the policy MUST be fetched from Stytch, cached, and then used.

If the Stytch RBAC API is unreachable and the cache is empty, the system MUST return a 503 Service Unavailable error for any authorization check.

#### Scenario: Permission check with cached policy
- **WHEN** a request requires permission verification
- **AND** the Stytch RBAC policy is in the Redis cache
- **THEN** the system reads permissions from the cached policy
- **AND** the system does NOT call the Stytch API

#### Scenario: Permission check with cache miss
- **WHEN** a request requires permission verification
- **AND** the Stytch RBAC policy is NOT in the Redis cache
- **THEN** the system fetches the policy from the Stytch RBAC API
- **AND** caches it in Redis with a 5-minute TTL
- **AND** resolves the permission from the fetched policy

#### Scenario: Permission check with Stytch API unavailable
- **WHEN** a request requires permission verification
- **AND** the Stytch RBAC policy is NOT in the Redis cache
- **AND** the Stytch RBAC API is unreachable
- **THEN** the system returns a 503 Service Unavailable error

## ADDED Requirements

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

### Requirement: Log mismatches between edge and Go validation

The Go backend SHALL log a warning when the edge middleware's `X-Forwarded-Auth` and the Go backend's independent Stytch API validation produce different results (e.g., JWT valid at edge but rejected by Stytch API).

#### Scenario: Edge says valid, Go says invalid

- **WHEN** a request arrives with `X-Forwarded-Auth: true`
- **AND** the Go backend attempts independent validation via Stytch API
- **AND** Stytch API returns an invalid/expired session response
- **THEN** the Go backend SHALL log a warning with the `stytch_member_id`, `stytch_organization_id`, and Stytch API error
- **AND** SHALL return a 401 Unauthorized
