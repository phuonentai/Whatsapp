## MODIFIED Requirements

### Requirement: Auth events recorded in the audit stream

The system SHALL record sign-in and lifecycle events into the organization audit stream (the CRM activity timeline, `tipo=sistema`) so they appear in the existing `?view=audit` settings view. Event types SHALL include: `magic_link_requested`, `login_succeeded`, `login_failed`, `logout`, `mfa_challenge_passed`, `mfa_challenge_failed`. Governance events sourced from verified Stytch lifecycle webhooks (`stytch-lifecycle-webhooks`) SHALL also be recorded: `member_invited`, `member_role_changed`, `member_removed`, `organization_updated`, `member_deprovisioned`, `support_impersonation_started`, `support_impersonation_ended` (from `support-impersonation`).

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

#### Scenario: Governance events appear in the audit view

- **WHEN** a verified lifecycle webhook produces a governance event (e.g., `member_role_changed`)
- **THEN** the corresponding `tipo=sistema` audit row SHALL appear in the `?view=audit` list
- **AND** the row SHALL contain only member IDs, organization ID, bounded detail, and timestamp

#### Scenario: Impersonation events appear in the audit view

- **WHEN** a support member starts and ends an impersonation
- **THEN** `support_impersonation_started` and `support_impersonation_ended` rows SHALL appear in the audit view
