## MODIFIED Requirements

### Requirement: Contact entity includes extended profile fields with Colombian document types

The system SHALL store the following fields on `crm.contacts` in addition to the existing phone-centric fields: `email` (VARCHAR, nullable, with partial unique index), `company_id` (FK to crm.companies, ON DELETE SET NULL), `source` (VARCHAR, default 'whatsapp'), `lead_status` (VARCHAR, default 'nuevo'), `job_title` (VARCHAR, nullable), `assigned_to` (FK to organizations.accounts, ON DELETE SET NULL), `tipo_documento` (VARCHAR, nullable, CHECK: CC/NIT/CE/TI/PP), `numero_documento` (VARCHAR, nullable), `consent_status` (VARCHAR(10) NOT NULL DEFAULT 'none', CHECK: none/requested/granted/withdrawn), and `consented_at` (TIMESTAMPTZ, nullable).

#### Scenario: WhatsApp upsert preserves manually set fields

- **WHEN** a WhatsApp message arrives and upserts a contact by phone number
- **THEN** the system SHALL NOT overwrite manually set email, company_id, source, lead_status, job_title, assigned_to, tipo_documento, numero_documento, consent_status, or consented_at fields
- **AND** the upsert SHALL only update display_name, avatar_url, last_message_at, and metadata as before

#### Scenario: New contact from WhatsApp gets default Colombian values

- **WHEN** a WhatsApp message arrives from a new Colombian phone number
- **THEN** the contact SHALL be created with `source = 'whatsapp'` and `lead_status = 'nuevo'`
- **AND** document type fields SHALL be NULL
- **AND** `consent_status` SHALL default to `none` and `consented_at` SHALL be NULL

#### Scenario: Consent status transitions

- **WHEN** the compliance pipeline updates a contact's consent
- **THEN** `consent_status` SHALL transition according to the consent state machine (none → requested → granted, with withdrawn reachable from any state)
- **AND** `consented_at` SHALL be set only when the status becomes `granted`
