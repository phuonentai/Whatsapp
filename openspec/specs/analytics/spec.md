# Spec: analytics

## Purpose

Read-only sales reporting for organizations over local CRM and invoicing data: invoiced revenue by period, top customers by revenue, pipeline funnel, and inactive contacts. Reports are org-scoped, gated by the `analytics` module, and reuse existing permissions — no new Stytch RBAC resources.

## Requirements

### Requirement: Invoiced revenue by period

The system SHALL provide `GET /api/analytics/revenue` returning invoiced revenue bucketed by week or month. The source SHALL be `invoicing.invoices` rows with `status = 'valid'`, summed by `amount` (COP) and grouped via `date_trunc` on the invoice creation timestamp. The endpoint SHALL accept `period` (`week` | `month`, default `month`), `from`, and `to` (ISO dates, both optional). When `from` or `to` are omitted, the window SHALL default to the last 30 days. Validation SHALL reject `from > to` and spans longer than 13 months with HTTP 400 and a Spanish error message. All rows SHALL be filtered by the requesting organization; data from other organizations SHALL NOT be included.

#### Scenario: Revenue for a month bucket

- **WHEN** an authenticated member with `invoice:view` calls `GET /api/analytics/revenue?period=month&from=2026-07-01&to=2026-07-31` for an org with two valid invoices (100000 and 250000) created in July
- **THEN** the response SHALL contain one bucket for July 2026 with `total = 350000`
- **AND** SHALL NOT include invoices with status other than `valid` or invoices from other organizations

#### Scenario: Invalid date range rejected

- **WHEN** a request provides `from` after `to` (e.g., `from=2026-08-01&to=2026-07-01`)
- **THEN** the system SHALL return HTTP 400 with a Spanish error message
- **AND** SHALL NOT execute the aggregation

#### Scenario: Default window used when dates omitted

- **WHEN** a request omits `from` and `to`
- **THEN** the system SHALL aggregate over the last 30 days from the current date

### Requirement: Top customers by invoiced revenue

The system SHALL provide `GET /api/analytics/top-customers` returning the top N customers by summed invoiced revenue (`invoices` with `status = 'valid'`). The customer name SHALL be resolved from the deal's linked company via `COALESCE` (fallback: linked contact display name or phone). The endpoint SHALL accept `limit` (default 10, max 50). The response SHALL be ordered by total revenue descending. All rows SHALL be org-scoped.

#### Scenario: Top customers ordered by revenue

- **WHEN** an authenticated member with `invoice:view` calls `GET /api/analytics/top-customers?limit=10`
- **THEN** the response SHALL list customers ordered by summed `valid` invoice amount descending
- **AND** SHALL include only the organization's own invoices

#### Scenario: Company missing falls back to contact

- **WHEN** a deal linked to a valid invoice has no company but has a linked contact with display name
- **THEN** the customer entry SHALL use the contact's display name (phone as fallback when display name is missing)

### Requirement: Pipeline funnel

The system SHALL provide `GET /api/analytics/funnel` returning the organization's deal pipeline: for each stage of the default pipeline, the count of open deals and the sum of their `monto`; plus aggregate counts of deals in `ganado`, `perdido`, and `abandonado` states, and the sum of `monto` for `ganado` deals. Deals in non-default pipelines SHALL be aggregated in a single `otras_pipelines` entry. All rows SHALL be org-scoped.

#### Scenario: Funnel reflects open deals per stage

- **WHEN** an authenticated member with `deal:view` calls `GET /api/analytics/funnel`
- **THEN** the response SHALL contain one entry per stage of the default pipeline with `cantidad` and `monto_total` for open deals
- **AND** SHALL contain `ganado`, `perdido`, `abandonado` aggregates with counts and, for `ganado`, the summed monto

#### Scenario: Deals in other pipelines are grouped

- **WHEN** the organization has open deals in a non-default pipeline
- **THEN** those deals SHALL be aggregated under a single `otras_pipelines` entry

### Requirement: Inactive contacts

The system SHALL provide `GET /api/analytics/inactive-contacts` returning contacts with no WhatsApp message activity since a threshold. The endpoint SHALL accept `days` (default 30, min 1, max 365). Contacts whose `last_message_at` is older than `now() - days` SHALL be classified as `inactivo`; contacts with `last_message_at` NULL (never messaged) SHALL be classified as `sin_actividad` in a separate bucket. The response SHALL include the contact's phone and display name. All rows SHALL be org-scoped.

#### Scenario: Contacts inactive since threshold

- **WHEN** an authenticated member with `contact:view` calls `GET /api/analytics/inactive-contacts?days=30`
- **THEN** the response SHALL contain contacts whose `last_message_at` is strictly older than 30 days
- **AND** SHALL NOT contain contacts with activity within the threshold

#### Scenario: Never-messaged contacts reported separately

- **WHEN** a contact has `last_message_at` NULL
- **THEN** the contact SHALL appear in the `sin_actividad` bucket and SHALL NOT appear in `inactivo`

#### Scenario: Invalid days parameter rejected

- **WHEN** a request provides `days=0` or `days=366`
- **THEN** the system SHALL return HTTP 400 with a Spanish error message

### Requirement: Analytics module gating and permissions

The analytics endpoints SHALL be gated by the `analytics` module in the module registry: requests from an organization where the module is disabled SHALL receive HTTP 403 with error code `module_disabled`. The module SHALL be seeded enabled for organizations by default. Each endpoint SHALL reuse an existing permission as its authorization gate: `invoice:view` for revenue and top-customers, `deal:view` for funnel, `contact:view` for inactive-contacts. The system SHALL NOT introduce new RBAC resources and SHALL NOT modify Stytch tenant policy state. All endpoints SHALL require the authenticated member's organization context.

#### Scenario: Module disabled blocks analytics

- **WHEN** an authenticated member calls an analytics endpoint for an organization where the `analytics` module is disabled
- **THEN** the system SHALL return HTTP 403 with error code `module_disabled`

#### Scenario: Missing permission blocks the report

- **WHEN** an authenticated member with the `analytics` module enabled but without `invoice:view` calls the revenue endpoint
- **THEN** the system SHALL return HTTP 403 with a Spanish error message
- **AND** SHALL NOT return revenue data

### Requirement: Org-scoped read-only aggregation

All analytics queries SHALL filter by `organization_id` derived from the authenticated session context, SHALL use only existing tables (no new tables, columns, or migrations), and SHALL NOT mutate data. Each query SHALL take `organization_id` as its first argument.

#### Scenario: Cross-tenant data never leaked

- **WHEN** an organization queries any analytics endpoint
- **THEN** results SHALL be limited to rows whose `organization_id` matches the requesting organization
- **AND** SHALL NOT mutate any table
