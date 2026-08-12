## ADDED Requirements

### Requirement: M2M JWT verification middleware

The system SHALL provide an M2M auth middleware accepting `Authorization: Bearer <jwt>` for service principals. Verification SHALL use a two-tier strategy: fast path verifies the JWT signature via the locally cached project JWKS (≤300s TTL, consistent with member JWTs) with issuer, audience, and expiry checks; slow path calls `POST /v1/b2b/m2m/token/authenticate-access-token` behind the existing circuit breaker when the fast path cannot resolve the key. Failed verification SHALL return 401 with no identity established. Breaker-open during the slow path SHALL return 503 (`m2m_auth_unavailable`), never a silent accept.

#### Scenario: Valid M2M JWT verified locally

- **WHEN** a request presents a valid M2M JWT signed with a cached JWKS key
- **THEN** the middleware SHALL verify signature, issuer, audience, and expiry locally
- **AND** SHALL establish the service-principal identity

#### Scenario: Unknown key triggers API fallback

- **WHEN** the JWT's key is not in the local JWKS cache
- **THEN** the middleware SHALL validate via `POST /v1/b2b/m2m/token/authenticate-access-token`
- **AND** SHALL establish the identity on success

#### Scenario: Invalid token rejected

- **WHEN** a request presents an invalid, expired, or wrong-audience JWT
- **THEN** the middleware SHALL return 401
- **AND** no identity SHALL be established

#### Scenario: Breaker open during fallback

- **WHEN** the circuit breaker is open during the API fallback
- **THEN** the middleware SHALL return 503 (`m2m_auth_unavailable`)

### Requirement: Scopes map to the existing permission model

The service principal's token scopes SHALL resolve to the existing permission strings via a declarative scope-to-permission table (e.g., `whatsapp:send` → `whatsapp:send`, `crm:read` → contact/deal view permissions). The resolved permission set SHALL be exposed on the request identity (via the `authcontext` seam) so existing permission gates apply unchanged to M2M callers. Unknown scopes SHALL be denied.

#### Scenario: Scoped M2M caller passes a permission gate

- **WHEN** an M2M request carries a scope granting the required permission
- **THEN** the existing permission middleware SHALL allow the request

#### Scenario: Unscoped M2M caller denied

- **WHEN** an M2M request lacks the scope for a required permission
- **THEN** the permission gate SHALL deny the request

#### Scenario: Unknown scope denied

- **WHEN** a token carries a scope not in the scope-to-permission table
- **THEN** the effective permission set SHALL NOT include permissions for it

### Requirement: Org binding via allowed-orgs allowlist

M2M tokens are project-scoped, so requests SHALL carry `X-Stytch-Organization-Id`, and the middleware SHALL validate it against the client's allowed-orgs claim (`org_ids`). Missing or non-allowlisted headers SHALL return 403. On success, the resolved organization SHALL be set as the request's `OrganizationID` in `authcontext`.

#### Scenario: Header within allowlist

- **WHEN** an M2M request carries an org id inside the client's `org_ids` allowlist
- **THEN** the middleware SHALL resolve that organization as the request context
- **AND** org-scoped handlers SHALL operate within it

#### Scenario: Header outside allowlist

- **WHEN** an M2M request carries an org id not in the client's allowlist
- **THEN** the middleware SHALL return 403

#### Scenario: Missing header

- **WHEN** an M2M request omits `X-Stytch-Organization-Id`
- **THEN** the middleware SHALL return 403

### Requirement: No local credential storage

M2M client secrets SHALL NOT be stored by the platform; tokens SHALL be validated and never persisted; audit of M2M calls SHALL contain only client id, organization id, scope, timestamp, and outcome.

#### Scenario: No secret or token material stored

- **WHEN** an M2M client authenticates
- **THEN** no local database row SHALL store the client secret or the presented JWT
