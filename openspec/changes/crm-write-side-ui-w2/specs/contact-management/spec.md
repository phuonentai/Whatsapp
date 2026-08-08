## ADDED Requirements

### Requirement: Contact detail lists associated negocios

The system SHALL support listing a contact's associated negocios via the `contact_id` filter on the deals list endpoint `GET /api/crm/negocios`.

#### Scenario: Filter deals by contact

- **WHEN** a GET request is made to `/api/crm/negocios?contact_id=1`
- **THEN** the system SHALL return only deals associated with contact ID 1

#### Scenario: Contact with no negocios returns empty list

- **WHEN** a GET request is made to `/api/crm/negocios?contact_id=99` and the contact has no deals
- **THEN** the system SHALL return an empty array
