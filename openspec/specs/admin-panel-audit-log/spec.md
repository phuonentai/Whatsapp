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
