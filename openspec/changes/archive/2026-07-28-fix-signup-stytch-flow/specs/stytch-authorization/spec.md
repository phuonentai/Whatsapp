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
