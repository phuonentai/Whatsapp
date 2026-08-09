## ADDED Requirements

### Requirement: Contact entity includes extended profile fields with Colombian document types

The system SHALL store the following fields on `crm.contacts` in addition to the existing phone-centric fields: `email` (VARCHAR, nullable, with partial unique index), `company_id` (FK to crm.companies, ON DELETE SET NULL), `source` (VARCHAR, default 'whatsapp'), `lead_status` (VARCHAR, default 'nuevo'), `job_title` (VARCHAR, nullable), `assigned_to` (FK to organizations.accounts, ON DELETE SET NULL), `tipo_documento` (VARCHAR, nullable, CHECK: CC/NIT/CE/TI/PP), `numero_documento` (VARCHAR, nullable), `consent_status` (VARCHAR(10) NOT NULL DEFAULT 'none', CHECK: none/requested/granted/withdrawn), and `consented_at` (TIMESTAMP, nullable).

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

### Requirement: Tenant-safe composite foreign keys on CRM assignment and parent links

The system SHALL enforce tenant isolation at the database level on all CRM assignment and parent/child references using composite foreign keys `(organization_id, <ref> -> <parent>(organization_id, id))`. The following columns SHALL reference `organizations.accounts (organization_id, id)`: `contacts.assigned_to`, `deals.assigned_to`, `companies.owner_account_id`, and `activities.realizada_por`. The following SHALL reference their target table's `(organization_id, id)`: `contacts.company_id`, `deals.contact_id`, `deals.company_id`, `conversations.contact_id`, and `messages.conversation_id`.

Delete actions SHALL preserve the pre-existing semantics: `ON DELETE SET NULL (column_list)` where the current behavior nulls the referencing column, and `ON DELETE CASCADE` where the current behavior cascades (`conversations.contact_id`, `messages.conversation_id`).

#### Scenario: Cross-tenant assignment rejected by the database

- **WHEN** an insert or update attempts to set `contacts.assigned_to` (or `deals.assigned_to`, `companies.owner_account_id`, `activities.realizada_por`) to an account whose `organization_id` differs from the row's `organization_id`
- **THEN** the database SHALL reject the statement with a foreign key violation

#### Scenario: Cross-tenant parent link rejected by the database

- **WHEN** an insert or update attempts to link a row to a `company_id`, `contact_id`, or `conversation_id` from a different `organization_id`
- **THEN** the database SHALL reject the statement with a foreign key violation

#### Scenario: Account deletion nulls assignments only

- **WHEN** an account referenced by an assignment column is deleted
- **THEN** the referencing column SHALL be set to NULL
- **AND** the row's `organization_id` SHALL remain unchanged

#### Scenario: Contact deletion cascades to conversations and messages

- **WHEN** a contact referenced by `conversations.contact_id` is deleted
- **THEN** the referencing conversations SHALL be deleted (cascade)
- **AND** messages referencing those conversations SHALL be deleted (cascade)

### Requirement: Idempotent WhatsApp message insertion

The system SHALL insert CRM messages conflict-safely: a unique index on `crm.messages (organization_id, whatsapp_message_id)` SHALL exist, and message creation for inbound WhatsApp messages SHALL use an insert with `ON CONFLICT (organization_id, whatsapp_message_id) DO NOTHING` as the primary operation rather than a check-then-insert pattern.

#### Scenario: Duplicate webhook message does not duplicate the message row

- **WHEN** an inbound message with a `whatsapp_message_id` that already exists for the organization is processed
- **THEN** the system SHALL NOT insert a second message row
- **AND** SHALL return the existing message without error

#### Scenario: Concurrent processing of the same message yields one row

- **WHEN** two webhook deliveries for the same `(organization_id, whatsapp_message_id)` are processed concurrently
- **THEN** exactly one `crm.messages` row SHALL exist
- **AND** neither processing path SHALL surface a unique-violation error

### Requirement: One active conversation per contact

The system SHALL enforce at most one conversation with `status = 'active'` per `(organization_id, contact_id)`, via a partial unique index `(organization_id, contact_id) WHERE status = 'active'`. Active conversation resolution SHALL use an idempotent insert (`ON CONFLICT (organization_id, contact_id) WHERE status = 'active' DO NOTHING`) with a fallback fetch of the existing active conversation.

#### Scenario: Concurrent message arrivals create one active conversation

- **WHEN** two messages from the same contact arrive concurrently and no active conversation exists
- **THEN** exactly one active conversation SHALL be created for that contact
- **AND** both messages SHALL be associated with that same conversation

#### Scenario: Closing a conversation allows a new active conversation

- **WHEN** the active conversation for a contact is closed or archived
- **THEN** the system SHALL allow creating a new active conversation for that contact

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

### Requirement: E.164 phone number canonicalization

The system SHALL canonicalize phone numbers to E.164 format and validate against the Colombian mobile pattern `^\+573\d{9}$`.

#### Scenario: Valid Colombian mobile number passes validation

- **WHEN** a phone number `+573001234567` is canonicalized
- **THEN** the system SHALL return the number unchanged and no error

#### Scenario: Non-Colombian number is logged and not rejected

- **WHEN** a phone number `+14155551234` (US number) does not match the Colombian pattern
- **THEN** the system SHALL log a warning with the original number
- **AND** SHALL still process the message with the unvalidated number

#### Scenario: Number without + prefix is canonicalized

- **WHEN** a phone number `573001234567` (missing the + prefix) is canonicalized
- **THEN** the system SHALL prepend `+` to produce `+573001234567`
- **AND** SHALL then validate against the E.164 pattern
