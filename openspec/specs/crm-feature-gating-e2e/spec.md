## Purpose

Define the E2E behavior of feature gating across Free, Pro, and Enterprise plan tiers: hidden tabs for restricted features and API-level 403 enforcement for gated endpoints.

## Requirements

### Requirement: Free plan restricts Pro features

The E2E tests SHALL verify that organizations on the Free plan cannot access Pro-tier CRM features.

#### Scenario: Free plan hides Empresas tab
- **WHEN** a user from `test-org-free` navigates to the CRM page
- **THEN** the Empresas tab SHALL NOT be visible
- **AND** navigating to `/dashboard/crm?view=empresas` SHALL redirect or show an upgrade banner

#### Scenario: Free plan hides Negocios tab
- **WHEN** a user from `test-org-free` navigates to the CRM page
- **THEN** the Negocios tab SHALL NOT be visible

#### Scenario: Free plan hides Actividad tab
- **WHEN** a user from `test-org-free` navigates to the CRM page
- **THEN** the Actividad tab SHALL NOT be visible

### Requirement: Enterprise plan restricts Tags to Enterprise tier

The E2E tests SHALL verify that tags are only available on Enterprise plans.

#### Scenario: Pro plan hides Etiquetas tab
- **WHEN** a user from `test-org-pro` navigates to the CRM page
- **THEN** the Etiquetas tab SHALL NOT be visible

#### Scenario: Enterprise plan shows all tabs
- **WHEN** a user from `test-org-enterprise` navigates to the CRM page
- **THEN** all tabs including Etiquetas SHALL be visible

### Requirement: API-level feature gate enforcement

The E2E tests SHALL verify that feature-gated API endpoints return 403 for unauthorized plans.

#### Scenario: Pro API endpoint rejected for Free plan
- **WHEN** a user from `test-org-free` calls `POST /api/crm/empresas`
- **THEN** the response SHALL have status 403

#### Scenario: Enterprise API endpoint rejected for Pro plan
- **WHEN** a user from `test-org-pro` calls `POST /api/crm/etiquetas`
- **THEN** the response SHALL have status 403
