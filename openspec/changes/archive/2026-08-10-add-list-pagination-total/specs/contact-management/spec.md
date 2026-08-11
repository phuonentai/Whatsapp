## MODIFIED Requirements

### Requirement: Contact list supports pagination and filtering

The system SHALL provide a paginated contact list endpoint with optional filters for `source`, `lead_status`, `company_id`, and `assigned_to`, and the response SHALL include a `total` count of rows matching the organization scope and active filters (ignoring `limit`/`offset`).

#### Scenario: Contacts filtered by lead_status in Spanish

- **WHEN** a GET request is made to `/api/crm/contacts?lead_status=calificado`
- **THEN** the system SHALL return only contacts with `lead_status = 'calificado'`

#### Scenario: Contact list includes total count

- **WHEN** a GET request is made to `/api/crm/contacts?limit=25&offset=0`
- **THEN** the response SHALL contain `{ data: [...], total: <count> }` where `total` reflects the organization-scoped, filter-matched contact count
