## ADDED Requirements

### Requirement: Magic-link send throttled per email and IP

The magic-link send path (`sendMagicLink` server action) SHALL throttle requests per normalized email address and per client IP using an in-process sliding-window limiter. Defaults SHALL be 5 sends per email per hour and 20 sends per IP per hour, overridable via `MAGIC_LINK_RATE_LIMIT_PER_EMAIL_PER_HOUR` and `MAGIC_LINK_RATE_LIMIT_PER_IP_PER_HOUR`. When throttled, the action SHALL NOT call Stytch.

#### Scenario: Burst above email limit is throttled

- **WHEN** the same email is submitted for magic-link sending more than the per-email hourly limit
- **THEN** the action SHALL return without calling Stytch
- **AND** the response SHALL be the neutral "If an account exists with that email, a magic link has been sent." message with a `throttled: true` flag

#### Scenario: Distinct emails under IP limit proceed

- **WHEN** distinct emails are submitted and neither the per-email nor per-IP limit is exceeded
- **THEN** the action SHALL perform the existing membership search and send flow

#### Scenario: Window slide re-allows sending

- **WHEN** the hourly window elapses for a previously throttled email/IP
- **THEN** the action SHALL allow new sends

### Requirement: Throttle responses do not leak member existence

Throttled responses SHALL be worded identically to the no-member response and SHALL NOT reveal whether the email corresponds to an existing member.

#### Scenario: Throttled response is neutral

- **WHEN** a request is throttled
- **THEN** the user-visible message SHALL match the standard neutral message
- **AND** no account-existence information SHALL be included

### Requirement: Limiter stores no credentials or tokens

The limiter SHALL retain only email addresses, IP addresses, and timestamps in memory. It MUST NOT store session tokens, magic-link tokens, passwords, or MFA material.

#### Scenario: Limiter state is ephemeral and minimal

- **WHEN** the server process restarts
- **THEN** limiter state SHALL be cleared (in-memory only)
- **AND** no credential or token material SHALL have been persisted at any point
