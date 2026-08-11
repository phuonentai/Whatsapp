## Purpose

Defines the contact entity with extended profile fields including Colombian document types, company association, and team member assignment.

## Requirements

### Requirement: Contact has extended profile fields including Colombian document types

The system SHALL extend `crm.contacts` with new nullable columns: `email`, `company_id`, `source`, `lead_status`, `job_title`, `assigned_to`, `tipo_documento`, and `numero_documento`.

#### Scenario: Contact created with Colombian document

- **WHEN** a contact is created with `tipo_documento = 'CC'` (Cédula de Ciudadanía) and `numero_documento = '1234567890'`
- **THEN** both fields SHALL be persisted

#### Scenario: Contact created with NIT

- **WHEN** a contact is created with `tipo_documento = 'NIT'` and `numero_documento = '900123456'`
- **THEN** both fields SHALL be persisted

#### Scenario: Valid document types accepted

- **WHEN** a contact is created with any of: CC (Cédula de Ciudadanía), NIT (NIT), CE (Cédula de Extranjería), TI (Tarjeta de Identidad), PP (Pasaporte)
- **THEN** the tipo_documento SHALL be accepted

#### Scenario: Invalid documento type rejected

- **WHEN** a contact is created with `tipo_documento = 'DNI'`
- **THEN** the system SHALL return validation error: "Tipo de documento inválido. Valores permitidos: CC, NIT, CE, TI, PP."

#### Scenario: Existing WhatsApp contact retains default values

- **WHEN** the migration adds new columns with default values
- **THEN** all existing contacts SHALL have `source = 'whatsapp'` and `lead_status = 'nuevo'`
- **AND** documento fields SHALL be NULL

### Requirement: Contact can be associated with a CRM Company

The system SHALL allow a contact to be linked to a CRM company via the `company_id` foreign key referencing `crm.companies(id)`. The FK SHALL use `ON DELETE SET NULL`.

#### Scenario: Contact linked to existing company

- **WHEN** a contact is updated with a valid `company_id`
- **THEN** the contact SHALL be associated with that company (empresa)

### Requirement: Contact can be assigned to a team member

The system SHALL allow a contact to be assigned to an account via the `assigned_to` field referencing `organizations.accounts(id)`.

#### Scenario: Contact assigned to a valid account (responsable)

- **WHEN** a contact is updated with `assigned_to` set to a valid account ID within the same organization
- **THEN** the contact SHALL be assigned to that account as its responsable

### Requirement: Contact email uniqueness within an organization

The system SHALL enforce that no two contacts within the same organization have the same non-null email address, via a partial unique index.

#### Scenario: Duplicate email rejected

- **WHEN** a contact is created with an email that already exists for another contact in the same organization
- **THEN** the system SHALL return: "Ya existe un contacto con este correo electrónico."

### Requirement: Contact list supports pagination and filtering

The system SHALL provide a paginated contact list endpoint with optional filters for `source`, `lead_status`, `company_id`, and `assigned_to`, and the response SHALL include a `total` count of rows matching the organization scope and active filters (ignoring `limit`/`offset`).

#### Scenario: Contacts filtered by lead_status in Spanish

- **WHEN** a GET request is made to `/api/crm/contacts?lead_status=calificado`
- **THEN** the system SHALL return only contacts with `lead_status = 'calificado'`

#### Scenario: Contact list includes total count

- **WHEN** a GET request is made to `/api/crm/contacts?limit=25&offset=0`
- **THEN** the response SHALL contain `{ data: [...], total: <count> }` where `total` reflects the organization-scoped, filter-matched contact count

### Requirement: Contact CRUD is RBAC-protected with Spanish error messages

The system SHALL require `contact:view` permission for read operations and `contact:manage` permission for create/update operations.

#### Scenario: User without permission sees Spanish error

- **WHEN** a user without `contact:manage` permission attempts to create a contact
- **THEN** the system SHALL return HTTP 403 with message "No tienes permiso para gestionar contactos."

### Requirement: Contact search by name, email, phone, or documento

The system SHALL support searching contacts by display_name, email, phone_number, or numero_documento via a query parameter.

#### Scenario: Search by document number

- **WHEN** a GET request is made to `/api/crm/contacts?search=1234567890`
- **THEN** the system SHALL return contacts whose display_name, email, phone_number, or numero_documento contains "1234567890" (case-insensitive)

### Requirement: Contact detail lists associated negocios

The system SHALL support listing a contact's associated negocios via the `contact_id` filter on the deals list endpoint `GET /api/crm/negocios`.

#### Scenario: Filter deals by contact

- **WHEN** a GET request is made to `/api/crm/negocios?contact_id=1`
- **THEN** the system SHALL return only deals associated with contact ID 1

#### Scenario: Contact with no negocios returns empty list

- **WHEN** a GET request is made to `/api/crm/negocios?contact_id=99` and the contact has no deals
- **THEN** the system SHALL return an empty array
