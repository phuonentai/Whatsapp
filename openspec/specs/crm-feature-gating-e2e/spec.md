## Purpose

Define the E2E behavior of feature gating across Free, Pro, and Enterprise plan tiers: gated tabs rendered disabled (visible, non-interactive, with upgrade hint) for restricted features, and API-level 403 enforcement for gated endpoints.
## Requirements
### Requirement: Free plan restricts Pro features

The E2E tests SHALL verify that organizations on the Free plan cannot use Pro-tier CRM features. Gated tabs SHALL remain visible but disabled, preserving the upgrade affordance.

#### Scenario: Free plan disables Empresas tab
- **WHEN** a user from `test-org-free` navigates to the CRM page
- **THEN** the Empresas tab SHALL be visible but rendered `disabled` with an upgrade hint
- **AND** navigating to `/dashboard/crm?view=empresas` SHALL show an upgrade banner

#### Scenario: Free plan disables Negocios tab
- **WHEN** a user from `test-org-free` navigates to the CRM page
- **THEN** the Negocios tab SHALL be visible but rendered `disabled` with an upgrade hint

#### Scenario: Free plan disables Actividad tab
- **WHEN** a user from `test-org-free` navigates to the CRM page
- **THEN** the Actividad tab SHALL be visible but rendered `disabled` with an upgrade hint

### Requirement: Enterprise plan restricts Tags to Enterprise tier

The E2E tests SHALL verify that tags are only available on Enterprise plans.

#### Scenario: Pro plan disables Etiquetas tab
- **WHEN** a user from `test-org-pro` navigates to the CRM page
- **THEN** the Etiquetas tab SHALL be visible but rendered `disabled` with an upgrade hint

#### Scenario: Enterprise plan shows all tabs
- **WHEN** a user from `test-org-enterprise` navigates to the CRM page
- **THEN** all tabs including Etiquetas SHALL be rendered enabled

### Requirement: API-level feature gate enforcement

The E2E tests SHALL verify that feature-gated API endpoints return 403 for unauthorized plans.

#### Scenario: Pro API endpoint rejected for Free plan
- **WHEN** a user from `test-org-free` calls `POST /api/crm/empresas`
- **THEN** the response SHALL have status 403

#### Scenario: Enterprise API endpoint rejected for Pro plan
- **WHEN** a user from `test-org-pro` calls `POST /api/crm/etiquetas`
- **THEN** the response SHALL have status 403

