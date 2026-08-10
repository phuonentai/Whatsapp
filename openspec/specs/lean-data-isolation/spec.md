## Purpose

Define how the system enforces tenant isolation using the `organization_id` from the verified JWT session claim, ensuring cross-tenant data isolation at the query layer.

## Requirements

### Requirement: Tenant-scoped queries enforced by organization_id

The system SHALL scope every query on tenant-specific tables by the `organization_id` extracted from the verified JWT session claim. The `organization_id` SHALL flow from the edge middleware through the API gateway to the repository/query layer via the `X-Stytch-Organization-Id` header.

#### Scenario: Repository query includes organization_id filter

- **WHEN** a repository method fetches data from a tenant-scoped table (e.g., `projects`, `contacts`, `deals`)
- **THEN** the query SHALL include a `WHERE organization_id = $orgId` clause
- **AND** `orgId` SHALL be derived from the `X-Stytch-Organization-Id` request header or the validated JWT claim

#### Scenario: Cross-tenant access attempt

- **WHEN** a request includes an `X-Stytch-Organization-Id` that does not match the `organization_id` of the resource being accessed
- **THEN** the query SHALL return zero rows
- **AND** the API SHALL return a 404 or 403 response (resource not found, not unauthorized, to avoid leaking existence)

### Requirement: JWT claim is authoritative source for organization_id

The `organization_id` in the verified JWT SHALL be treated as the authoritative tenant context. No user-provided input SHALL override the JWT's organization_id when determining data access scope.

#### Scenario: Request with mismatched org IDs in JWT and payload

- **WHEN** a request contains an `organization_id` in the request body that differs from the one in the verified JWT
- **THEN** the system SHALL use the JWT's `organization_id` for all database queries
- **AND** the system SHALL ignore the user-provided `organization_id`

### Requirement: Optional PostgreSQL RLS policy for defense-in-depth

The system SHALL support an optional PostgreSQL Row-Level Security (RLS) defense-in-depth layer on tenant-scoped tables. When RLS policies are enabled, the system SHALL enforce row-level filtering based on the `app.current_organization_id` session variable. The system MAY expose RLS policy enablement as an opt-in deployment choice rather than a default; however, once enabled, enforcement SHALL follow the scenarios below.

#### Scenario: RLS policy enforced by PostgreSQL

- **WHEN** an RLS policy is enabled on a tenant-scoped table
- **AND** a query is executed without setting `app.current_organization_id`
- **THEN** PostgreSQL SHALL return zero rows (RLS blocks the query)

#### Scenario: RLS policy does not interfere with app_session role queries

- **WHEN** a background job or system-level operation (e.g., webhook handler, cron job) queries a tenant-scoped table
- **THEN** the operation SHALL set `app.current_organization_id` to the appropriate organization UUID
- **OR** the session role SHALL be set to `app_session` with RLS bypass privileges

### Requirement: Database-level referential tenant isolation

The system SHALL enforce tenant isolation as a database constraint in addition to query-layer scoping: every CRM table SHALL expose `UNIQUE (organization_id, id)` on its primary key, and cross-table references SHALL use composite foreign keys that include `organization_id`, so the database rejects references to rows belonging to another organization.

#### Scenario: Cross-tenant reference rejected at the database layer

- **WHEN** an insert or update attempts to reference an account, contact, company, deal, conversation, pipeline, pipeline stage, or tag from a different `organization_id`
- **THEN** the database SHALL reject the statement with a foreign key violation
- **AND** no application-level check SHALL be required to prevent the invalid reference

#### Scenario: Existing query-layer isolation remains in effect

- **WHEN** an application query is executed without an `organization_id` filter
- **THEN** the database-level constraints SHALL NOT grant any additional visibility (they prevent invalid writes only)
- **AND** query-layer scoping per the "Tenant-scoped queries enforced by organization_id" requirement SHALL remain the access control mechanism
