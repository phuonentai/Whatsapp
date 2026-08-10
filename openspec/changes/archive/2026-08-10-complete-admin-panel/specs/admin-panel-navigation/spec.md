## ADDED Requirements

### Requirement: Sidebar exposes Inbox and CRM navigation

The dashboard sidebar SHALL include "Inbox" linking to `/dashboard/inbox` and "CRM" linking to `/dashboard/crm`, following the existing permission-filtered navigation pattern.

#### Scenario: Entitled user sees Inbox and CRM

- **WHEN** a user whose role or module entitlements grant access to Inbox and CRM views the dashboard sidebar
- **THEN** the sidebar SHALL display "Inbox" and "CRM" entries linking to `/dashboard/inbox` and `/dashboard/crm`

#### Scenario: User without entitlement does not see restricted entries

- **WHEN** a user lacks the entitlements or permissions required for Inbox or CRM
- **THEN** the corresponding sidebar entry SHALL NOT be rendered

#### Scenario: Sidebar entry is active on nested routes

- **WHEN** a user is on `/dashboard/inbox` or `/dashboard/crm?view=...`
- **THEN** the matching sidebar entry SHALL be highlighted as active

### Requirement: Dead header actions are removed or wired

The dashboard header SHALL NOT contain inert icon buttons; the Support and Preferences controls SHALL either navigate to a real destination or be removed.

#### Scenario: Support button has a destination

- **WHEN** a user activates the header Support control
- **THEN** it SHALL navigate to a configured support destination (e.g., `mailto:` or `/dashboard/settings`)

#### Scenario: Preferences button has a destination

- **WHEN** a user activates the header Preferences control
- **THEN** it SHALL navigate to the workspace settings page (`/dashboard/settings`)
