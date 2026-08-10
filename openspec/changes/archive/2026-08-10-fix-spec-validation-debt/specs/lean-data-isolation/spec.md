## MODIFIED Requirements

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
