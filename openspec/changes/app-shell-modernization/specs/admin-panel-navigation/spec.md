# Delta Spec: admin-panel-navigation — app-shell-modernization

## MODIFIED Requirements

### Requirement: Sidebar exposes Inbox and CRM navigation

The dashboard sidebar SHALL include "Inbox" linking to `/dashboard/inbox` and "CRM" linking to `/dashboard/crm`, following the existing permission-filtered navigation pattern. The sidebar SHALL also include a "Dashboard" entry linking to `/dashboard`, and each active entry SHALL carry `aria-current="page"`.

#### Scenario: Entitled user sees Inbox and CRM

- **WHEN** a user whose role or module entitlements grant access to Inbox and CRM views the dashboard sidebar
- **THEN** the sidebar SHALL display "Inbox" and "CRM" entries linking to `/dashboard/inbox` and `/dashboard/crm`

#### Scenario: User without entitlement does not see restricted entries

- **WHEN** a user lacks the entitlements or permissions required for Inbox or CRM
- **THEN** the corresponding sidebar entry SHALL NOT be rendered

#### Scenario: Sidebar entry is active on nested routes

- **WHEN** a user is on `/dashboard/inbox` or `/dashboard/crm?view=...`
- **THEN** the matching sidebar entry SHALL be highlighted as active and SHALL carry `aria-current="page"`

#### Scenario: Dashboard entry always present

- **WHEN** any authenticated user views the sidebar
- **THEN** the sidebar SHALL display a "Dashboard" entry linking to `/dashboard`, highlighted when the current route is `/dashboard`
