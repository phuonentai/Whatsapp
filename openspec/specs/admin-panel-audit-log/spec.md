## Purpose

Defines the admin settings audit log view that exposes governance events and is strictly read-only for admins.
## Requirements
### Requirement: Audit log view in settings

The dashboard settings page SHALL expose a read-only audit log view (reachable via `?view=audit`) that lists the organization's unified CRM activity, visible only to users holding the `audit:view` permission. The view SHALL reuse the existing `GET /api/crm/actividades` endpoint; no new backend endpoint is introduced.

#### Scenario: Audit view shows org activity

- **WHEN** a user with `audit:view` permission navigates to settings with `?view=audit`
- **THEN** the system SHALL display the organization's activity list (notes, calls, emails, meetings, tasks, WhatsApp messages, and system events)
- **AND** each row SHALL show the activity tipo, subject, content, performing member, timestamp, and any linked entity references

#### Scenario: Audit view hidden without permission

- **WHEN** a user without `audit:view` permission views the settings page
- **THEN** the audit log view SHALL NOT be reachable and its overview entry SHALL NOT be rendered

#### Scenario: Filter audit log by tipo

- **WHEN** a user selects a tipo filter (e.g., `llamada`)
- **THEN** the system SHALL fetch activities filtered by that tipo via the `tipo` query parameter

#### Scenario: Audit log pagination

- **WHEN** the activity list exceeds the page size
- **THEN** the system SHALL provide pagination controls that page through results using `limit`/`offset`

### Requirement: Audit log is read-only

The audit log view SHALL NOT provide create, edit, or delete controls; activities are immutable records.

#### Scenario: No activity mutations offered

- **WHEN** a user views the audit log
- **THEN** the view SHALL expose no controls to create, modify, or delete activities

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

