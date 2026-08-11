## ADDED Requirements

### Requirement: Paginated CRM lists include a total count

The paginated CRM list endpoints — `GET /api/crm/contactos`, `GET /api/crm/empresas`, and the activity list endpoints — SHALL return a `total` field alongside the `data` array in the response envelope. `total` SHALL equal the number of rows matching the request's organization scope and active filters (same WHERE clauses as the page query, ignoring `limit`/`offset`). The `data` array SHALL remain the existing list shape so non-paginated consumers are unaffected.

#### Scenario: Contact list returns total count

- **WHEN** a GET request is made to `/api/crm/contactos?limit=25&offset=0`
- **THEN** the response SHALL contain `{ data: [...], total: <count> }` where `total` is the number of contacts for the organization matching the same filters

#### Scenario: Total respects filters

- **WHEN** a GET request is made to `/api/crm/contactos?lead_status=calificado&limit=25&offset=0`
- **THEN** `total` SHALL reflect only the contacts matching `lead_status=calificado` for the organization

#### Scenario: Company list returns total count

- **WHEN** a GET request is made to `/api/crm/empresas?limit=25&offset=0`
- **THEN** the response SHALL contain `{ data: [...], total: <count> }`

#### Scenario: Empty result set returns total zero

- **WHEN** an organization has no matching rows for a paginated list endpoint
- **THEN** the response SHALL contain an empty `data` array and `total: 0`

### Requirement: Total is scoped per organization

The `total` field SHALL be computed under the same organization scoping as the list rows. No cross-tenant counts SHALL be exposed.

#### Scenario: Total limited to requesting organization

- **WHEN** a request is made to a paginated CRM list endpoint
- **THEN** `total` SHALL count only rows belonging to the requesting organization
