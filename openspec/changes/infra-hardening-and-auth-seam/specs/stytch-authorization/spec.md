## ADDED Requirements

### Requirement: JWKS public keys cached locally with bounded TTL

The system SHALL cache Stytch B2B JWKS public keys locally with a TTL of at most 300 seconds, so key rotation is reflected promptly and stale keys are never accepted beyond the constitutional freshness bound.

#### Scenario: Key rotation reflected within five minutes

- **WHEN** Stytch rotates a signing key
- **THEN** the previously cached key SHALL expire within 300 seconds
- **AND** the next validation SHALL fetch the refreshed JWKS from Stytch

#### Scenario: Cache hit serves the current key

- **WHEN** a request presents a token whose key ID is cached
- **AND** the cached entry is younger than 300 seconds
- **THEN** validation SHALL use the cached key without calling the Stytch JWKS endpoint
