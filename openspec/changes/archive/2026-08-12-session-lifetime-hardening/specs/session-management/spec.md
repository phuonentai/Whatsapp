## ADDED Requirements

### Requirement: Sliding session renewal

While the application is open and visible, the client SHALL periodically renew the session lifetime by requesting a refresh with `session_duration_minutes` so active users do not hit a hard logout at the fixed lifetime. On renewal failure (revoked or expired session) the client SHALL clear session cookies and redirect to `/auth`.

#### Scenario: Active user session extends

- **WHEN** a member keeps the application open and visible past 10 minutes
- **THEN** the client SHALL request a session refresh with the configured `session_duration_minutes`
- **AND** the underlying session lifetime SHALL be extended

#### Scenario: Renewal fails

- **WHEN** the session refresh fails because the session was revoked or expired
- **THEN** the client SHALL clear `stytch_session` and `stytch_session_jwt` cookies
- **AND** SHALL redirect to `/auth` with the current path as `returnTo`

#### Scenario: Renewal pauses when hidden

- **WHEN** the document is not visible
- **THEN** the client SHALL NOT issue renewal requests

### Requirement: Single session-duration default

The system SHALL use a single configurable session duration with a documented default of 480 minutes (8 hours), set via `NEXT_PUBLIC_STYTCH_SESSION_DURATION_MINUTES`. Documentation SHALL NOT contradict the code default.

#### Scenario: Docs match code default

- **WHEN** the session duration is checked in configuration docs and in `getSessionDurationMinutes()`
- **THEN** both SHALL report the same value (default 480)

### Requirement: Revoke sessions on member deactivation

When a member is deactivated, the system SHALL revoke the member's active Stytch sessions. Session revocation SHALL be idempotent and SHALL NOT block the deactivation when Stytch is unreachable; the failure SHALL be logged and surfaced as a pending-revocation notice.

#### Scenario: Deactivation revokes sessions

- **WHEN** an admin deactivates a member
- **THEN** the system SHALL list the member's active sessions via the Stytch API and revoke each
- **AND** the member SHALL no longer authenticate with those session tokens

#### Scenario: Stytch unreachable during revocation

- **WHEN** the session-revocation calls fail or the circuit breaker is open
- **THEN** the deactivation SHALL still complete locally
- **AND** the failure SHALL be logged with a `session_revocation_pending` notice

#### Scenario: Revoking an already-revoked session

- **WHEN** a session was already revoked
- **THEN** the revocation step SHALL treat it as a no-op success (idempotent)

### Requirement: Session state remains in Stytch

Local storage and databases SHALL NOT hold session tokens or session state; all session lifecycle operations SHALL be performed through the Stytch session APIs.

#### Scenario: No local session store

- **WHEN** any session operation completes
- **THEN** no session token or session state SHALL be persisted locally
