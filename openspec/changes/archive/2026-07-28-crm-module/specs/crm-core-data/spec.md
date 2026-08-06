## ADDED Requirements

### Requirement: Contact entity includes extended profile fields with Colombian document types

The system SHALL store the following fields on `crm.contacts` in addition to the existing phone-centric fields: `email` (VARCHAR, nullable, with partial unique index), `company_id` (FK to crm.companies, ON DELETE SET NULL), `source` (VARCHAR, default 'whatsapp'), `lead_status` (VARCHAR, default 'nuevo'), `job_title` (VARCHAR, nullable), `assigned_to` (FK to organizations.accounts, ON DELETE SET NULL), `tipo_documento` (VARCHAR, nullable, CHECK: CC/NIT/CE/TI/PP), and `numero_documento` (VARCHAR, nullable).

#### Scenario: WhatsApp upsert preserves manually set fields

- **WHEN** a WhatsApp message arrives and upserts a contact by phone number
- **THEN** the system SHALL NOT overwrite manually set email, company_id, source, lead_status, job_title, assigned_to, tipo_documento, or numero_documento fields
- **AND** the upsert SHALL only update display_name, avatar_url, last_message_at, and metadata as before

#### Scenario: New contact from WhatsApp gets default Colombian values

- **WHEN** a WhatsApp message arrives from a new Colombian phone number
- **THEN** the contact SHALL be created with `source = 'whatsapp'` and `lead_status = 'nuevo'`
- **AND** document type fields SHALL be NULL

### Requirement: Activity record created for inbound WhatsApp messages

The system SHALL create an Activity record of type `whatsapp_message` when processing an inbound WhatsApp message via CRMService.ProcessInboundMessage, only when the `crm_activities` feature is enabled for the organization.

#### Scenario: Activity created when feature enabled

- **WHEN** CRMService.ProcessInboundMessage successfully creates a Contact, Conversation, and Message, and `crm_activities` is enabled
- **THEN** the system SHALL create an Activity with:
  - type = 'whatsapp_message'
  - contact_id = the upserted contact's ID
  - conversation_id = the resolved conversation's ID
  - subject = "Mensaje de WhatsApp de {phone_number}"
  - content = first 500 characters of the message content
  - performed_at = the WhatsApp message timestamp
  - metadata = {message_id: msg.ID, direction: "inbound"}
- **AND** the system SHALL publish a `crm.actividad.creada` event

#### Scenario: Activity skipped when feature disabled

- **WHEN** CRMService.ProcessInboundMessage successfully persists Contact, Conversation, and Message, but `crm_activities` is disabled
- **THEN** no Activity record SHALL be created
- **AND** the method SHALL return nil (success)

#### Scenario: Activity creation failure does not block message processing

- **WHEN** CRMService.ProcessInboundMessage successfully persists the Contact, Conversation, and Message but fails to create the Activity (e.g., database error)
- **THEN** the error SHALL be logged as a warning
- **AND** the method SHALL return nil (success)

## MODIFIED Requirements

### Requirement: Contact entity with organization scoping

The system SHALL store contacts in `crm.contacts` scoped by `organization_id` with a unique constraint on `(organization_id, phone_number)`. Contacts SHALL also support optional fields: `email` (with partial unique index per organization), `company_id` (FK to crm.companies), `source`, `lead_status`, `job_title`, `assigned_to` (FK to organizations.accounts), `tipo_documento` (Colombian document type: CC/NIT/CE/TI/PP), and `numero_documento`.

#### Scenario: New contact is created from webhook message

- **WHEN** a message arrives from a Colombian phone number that does not exist in `crm.contacts` for the organization
- **THEN** the system SHALL create a new contact with the E.164 phone number, default `display_name`, `source = 'whatsapp'`, and `lead_status = 'nuevo'`
- **AND** SHALL return the created contact

#### Scenario: Existing contact is returned on duplicate phone

- **WHEN** a message arrives from a phone number that already exists in `crm.contacts` for the organization
- **THEN** the system SHALL return the existing contact without overwriting email, company_id, source, lead_status, job_title, assigned_to, tipo_documento, or numero_documento
- **AND** SHALL NOT create a duplicate row
- **AND** SHALL update display_name, avatar_url, and last_message_at if the incoming values are non-empty/newer
