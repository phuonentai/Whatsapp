## ADDED Requirements

### Requirement: Passkey registration

An authenticated member SHALL be able to register a passkey from settings. Registration SHALL use the Stytch B2B WebAuthn register APIs and the browser WebAuthn ceremony, bound to the application origin (RP ID).

#### Scenario: Member registers a passkey

- **WHEN** an authenticated member starts passkey setup and completes the browser credential ceremony
- **THEN** the system SHALL complete registration with the Stytch B2B WebAuthn API
- **AND** the passkey SHALL be recorded by Stytch and listed in the member's passkey management view

#### Scenario: Registration requires a session

- **WHEN** an unauthenticated request attempts passkey registration
- **THEN** the system SHALL reject the attempt (no passkey created)

### Requirement: Passkey sign-in

The `/auth` flow SHALL support passkey authentication for existing members after email resolution. On success the standard session cookies SHALL be set. If the member has no passkeys or the ceremony fails, the flow SHALL fall back to the magic-link path.

#### Scenario: Member signs in with a passkey

- **WHEN** a member with a registered passkey completes the browser assertion
- **THEN** the system SHALL authenticate via the Stytch B2B WebAuthn API
- **AND** SHALL set `stytch_session` and `stytch_session_jwt` cookies
- **AND** SHALL redirect to the intended destination

#### Scenario: Passkey ceremony fails

- **WHEN** the browser assertion fails or is cancelled
- **THEN** the flow SHALL NOT set session cookies
- **AND** SHALL offer the magic-link path as fallback

#### Scenario: MFA required after passkey

- **WHEN** the passkey authentication response indicates `mfa_required`
- **THEN** the flow SHALL NOT set session cookies
- **AND** SHALL route to the MFA challenge step carrying the `intermediate_session_token` (mirroring the magic-link MFA continuation), so the existing TOTP challenge can exchange it for a full session

#### Scenario: Primary auth required after passkey

- **WHEN** the passkey authentication response indicates `primary_required` (e.g., org enforces SSO)
- **THEN** the flow SHALL NOT set session cookies
- **AND** SHALL route the member to the appropriate primary-auth continuation instead of dead-ending

#### Scenario: Passkey auth challenge start is rate-limited

- **WHEN** a client invokes passkey authentication challenge-start for a member email more often than the configured per-email or per-IP limit
- **THEN** the system SHALL reject the additional challenge-start attempts
- **AND** SHALL NOT reveal whether a passkey exists for that member

### Requirement: Circuit-breaker fallback for passkey Stytch calls

All passkey outbound Stytch calls (register start/complete, authenticate start/complete, member WebAuthn list/delete) SHALL run through the frontend circuit-breaker wrapper (threshold 5, timeout 10s, half-open probe 2 — mirroring the Go adapter contract). On breaker-open, the system SHALL return a structured 503-style error and SHALL NOT issue session cookies.

#### Scenario: Breaker open during passkey authentication

- **WHEN** the Stytch B2B WebAuthn API is failing and the circuit breaker is open
- **THEN** the flow SHALL return a structured `passkey_unavailable` error
- **AND** SHALL NOT set session cookies
- **AND** SHALL offer the magic-link path as fallback

#### Scenario: Breaker open during registration or management

- **WHEN** the circuit breaker is open during passkey registration, listing, or deletion
- **THEN** the system SHALL return a structured error state
- **AND** SHALL NOT perform partial registration or leave inconsistent local state (no local passkey records exist by design)

### Requirement: Browser ceremony timeout and outcome taxonomy

The browser WebAuthn ceremony SHALL be bounded by an abort timeout (configurable: 60s for credential creation, 120s for assertion). The flow SHALL distinguish user cancellation from Stytch failure from network failure and handle each distinctly.

#### Scenario: User cancels the passkey prompt

- **WHEN** the user dismisses or aborts the browser passkey prompt
- **THEN** the flow SHALL treat it as a user cancellation (not an error)
- **AND** SHALL silently offer the magic-link path without failure noise or audit noise

#### Scenario: Stytch or network failure during the ceremony exchange

- **WHEN** the Stytch API call fails or the network is unavailable
- **THEN** the flow SHALL return a structured error
- **AND** SHALL record a bounded `login_failed` audit detail (best-effort, non-blocking)
- **AND** SHALL offer the magic-link path as fallback

### Requirement: Self-service passkey management

A member SHALL be able to list and delete their own registered passkeys. List and delete SHALL derive `member_id` and `organization_id` from the authenticated session — client-supplied member/org identifiers SHALL be ignored.

#### Scenario: Member deletes a passkey

- **WHEN** a member deletes a passkey
- **THEN** the system SHALL remove it via the Stytch member WebAuthn API using the session-derived member/org scope
- **AND** the passkey SHALL no longer be offered at sign-in
- **AND** a delete of an already-deleted passkey SHALL be treated as success (idempotent)

#### Scenario: Management actions ignore client-supplied scoping

- **WHEN** a client submits a list/delete request with member/org identifiers that do not match the authenticated session
- **THEN** the system SHALL ignore the client-supplied identifiers
- **AND** SHALL operate only within the authenticated session's member/org scope

### Requirement: Passkey material stored only in Stytch

Local storage MUST NOT contain passkey public keys, attestation objects, assertion responses, or session material. The local database SHALL NOT store any passkey-related records.

#### Scenario: No local passkey records

- **WHEN** any passkey operation completes
- **THEN** no passkey credential material SHALL be written to the local database or audit stream
