## ADDED Requirements

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
