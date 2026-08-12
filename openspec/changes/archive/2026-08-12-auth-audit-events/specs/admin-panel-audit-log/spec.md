## ADDED Requirements

### Requirement: Auth events recorded in the audit stream

The system SHALL record sign-in related events into the organization audit stream (the CRM activity timeline, `tipo=sistema`) so they appear in the existing `?view=audit` settings view. Event types SHALL include: `magic_link_requested`, `login_succeeded`, `login_failed`, `logout`, `mfa_challenge_passed`, `mfa_challenge_failed`.

#### Scenario: Successful login records an event

- **WHEN** a member successfully authenticates via magic link
- **THEN** the system SHALL record a `login_succeeded` audit row attributed to the member's `stytch_member_id` in the member's organization
- **AND** the row SHALL be visible in the audit log view

#### Scenario: Failed login records an event

- **WHEN** magic-link token consumption fails (invalid, expired, or rejected token)
- **THEN** the system SHALL record a `login_failed` audit row with a bounded failure detail (e.g., `invalid_token`, `expired_token`)
- **AND** no session token, JWT, or magic-link token SHALL be included in the row

#### Scenario: Magic link request records an event

- **WHEN** the send-magic-link action successfully sends a link for a member organization
- **THEN** the system SHALL record a `magic_link_requested` audit row for that organization

#### Scenario: Logout records an event

- **WHEN** a member logs out and a session was present
- **THEN** the system SHALL record a `logout` audit row for the member's organization

### Requirement: Audit recording is best-effort and non-blocking

Audit recording SHALL NOT alter the outcome of the authentication action. A failure to record an audit event SHALL be logged as a warning and SHALL NOT fail the underlying auth operation.

#### Scenario: Audit write failure does not block login

- **WHEN** the audit-recording call fails during a successful login
- **THEN** the login SHALL still complete successfully
- **AND** the failure SHALL be logged as a warning

### Requirement: Audit rows contain no credential or session material

Audit rows SHALL contain only event type, `stytch_member_id`, `organization_id`, timestamp, and a bounded non-sensitive detail. They MUST NOT contain session tokens, JWTs, magic-link tokens, passwords, MFA codes, or recovery codes.

#### Scenario: Audit payload is minimal

- **WHEN** any auth event is recorded
- **THEN** the payload SHALL NOT include tokens, credentials, or raw error bodies
- **AND** `detail` SHALL be drawn from a bounded enum
