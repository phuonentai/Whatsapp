## ADDED Requirements

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
