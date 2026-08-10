## ADDED Requirements

### Requirement: Instagram contact resolution by instagram_user_id

The system SHALL resolve contacts for inbound Instagram messages by `instagram_user_id` in `crm.contacts`, scoped to the organization, with a partial unique index `(organization_id, instagram_user_id) WHERE instagram_user_id IS NOT NULL`. Instagram contacts SHALL be created with `phone_number = NULL`, `source = 'instagram'`, and `instagram_username`/`avatar_url` backfilled asynchronously from the Instagram Graph API.

#### Scenario: New Instagram contact created

- **WHEN** an `instagram.message.received` event arrives from an IG user id that does not exist in `crm.contacts` for the organization
- **THEN** the system SHALL create a contact with `instagram_user_id` set, `phone_number` NULL, `source = 'instagram'`, and `lead_status = 'nuevo'`
- **AND** SHALL enqueue an async `GetIGUser` backfill to populate `instagram_username` and `avatar_url`

#### Scenario: Existing Instagram contact returned

- **WHEN** an `instagram.message.received` event arrives from an IG user id that already exists in `crm.contacts` for the organization
- **THEN** the system SHALL return the existing contact without creating a duplicate row
- **AND** SHALL NOT overwrite manually set fields

#### Scenario: Backfill failure retries

- **WHEN** the `GetIGUser` backfill call fails with a transient error (timeout, 5xx)
- **THEN** the system SHALL retry with the outbox retry/backoff mechanism
- **AND** SHALL dead-letter after exhausting attempts, leaving `instagram_username` NULL

## MODIFIED Requirements

### Requirement: Activity record created for inbound WhatsApp messages

The system SHALL create an Activity record of type `whatsapp_message` (for WhatsApp messages) or `instagram_message` (for Instagram messages) when processing an inbound message via the CRM inbound pipeline, only when the `crm_activities` feature is enabled for the organization.

#### Scenario: Activity created when feature enabled

- **WHEN** the CRM inbound pipeline successfully creates a Contact, Conversation, and Message, and `crm_activities` is enabled
- **THEN** the system SHALL create an Activity with:
  - type = 'whatsapp_message' for WhatsApp messages, 'instagram_message' for Instagram messages
  - contact_id = the upserted contact's ID
  - conversation_id = the resolved conversation's ID
  - subject = "Mensaje de WhatsApp de {phone_number}" for WhatsApp, "Mensaje de Instagram de {instagram_username}" for Instagram
  - content = first 500 characters of the message content
  - performed_at = the message timestamp
  - metadata = {message_id: msg.ID, direction: "inbound", channel: "whatsapp"|"instagram"}
- **AND** the system SHALL publish a `crm.actividad.creada` event

#### Scenario: Activity skipped when feature disabled

- **WHEN** the CRM inbound pipeline successfully persists Contact, Conversation, and Message, but `crm_activities` is disabled
- **THEN** no Activity record SHALL be created
- **AND** the method SHALL return nil (success)

#### Scenario: Activity creation failure does not block message processing

- **WHEN** the CRM inbound pipeline successfully persists the Contact, Conversation, and Message but fails to create the Activity (e.g., database error)
- **THEN** the error SHALL be logged as a warning
- **AND** the method SHALL return nil (success)

### Requirement: Idempotent message insertion by provider id

The system SHALL insert CRM messages conflict-safely: a unique index on `crm.messages (organization_id, channel, provider_message_id)` SHALL exist, and message creation for inbound messages SHALL use an insert with `ON CONFLICT (organization_id, channel, provider_message_id) DO NOTHING` as the primary operation rather than a check-then-insert pattern. The provider message id column (`provider_message_id`, renamed from `whatsapp_message_id`) SHALL store the WhatsApp message ID for WhatsApp messages and the Instagram `mid` for Instagram messages.

#### Scenario: Duplicate provider message does not duplicate the message row

- **WHEN** an inbound message with a `provider_message_id` that already exists for the organization and channel is processed
- **THEN** the system SHALL NOT insert a second message row
- **AND** SHALL return the existing message without error

#### Scenario: Concurrent processing of the same message yields one row

- **WHEN** two webhook deliveries for the same `(organization_id, channel, provider_message_id)` are processed concurrently
- **THEN** exactly one `crm.messages` row SHALL exist
- **AND** neither processing path SHALL surface a unique-violation error

#### Scenario: Same provider id on different channels does not collide

- **WHEN** a WhatsApp message and an Instagram message share the same `provider_message_id` string within an organization
- **THEN** both SHALL be stored as separate rows

### Requirement: One active conversation per contact per channel

The system SHALL enforce at most one conversation with `status = 'active'` per `(organization_id, contact_id, channel)`, via a partial unique index `(organization_id, contact_id, channel) WHERE status = 'active'`. Active conversation resolution SHALL use an idempotent insert (`ON CONFLICT (organization_id, contact_id, channel) WHERE status = 'active' DO NOTHING`) with a fallback fetch of the existing active conversation.

#### Scenario: Concurrent message arrivals create one active conversation

- **WHEN** two messages from the same contact arrive concurrently and no active conversation exists
- **THEN** exactly one active conversation SHALL be created for that contact
- **AND** both messages SHALL be associated with that same conversation

#### Scenario: Closing a conversation allows a new active conversation

- **WHEN** the active conversation for a contact is closed or archived
- **THEN** the system SHALL allow creating a new active conversation for that contact

#### Scenario: Same contact on both channels keeps separate conversations

- **WHEN** a contact messages via WhatsApp and via Instagram
- **THEN** the system SHALL maintain distinct active conversations for `channel = 'whatsapp'` and `channel = 'instagram'`

### Requirement: Contact entity with organization scoping

The system SHALL store contacts in `crm.contacts` scoped by `organization_id`. Phone-based uniqueness SHALL be a partial unique index `(organization_id, phone_number) WHERE phone_number IS NOT NULL` (phone SHALL be nullable to support Instagram contacts), and Instagram identity SHALL be unique via `(organization_id, instagram_user_id) WHERE instagram_user_id IS NOT NULL`. Contacts SHALL also support optional fields: `email` (with partial unique index per organization), `company_id` (FK to crm.companies), `source` (CHECK: whatsapp/instagram/manual/import/api), `lead_status`, `job_title`, `assigned_to` (FK to organizations.accounts), `tipo_documento` (Colombian document type: CC/NIT/CE/TI/PP), `numero_documento`, `instagram_user_id`, and `instagram_username`.

#### Scenario: New contact is created from WhatsApp webhook message

- **WHEN** a message arrives from a Colombian phone number that does not exist in `crm.contacts` for the organization
- **THEN** the system SHALL create a new contact with the E.164 phone number, default `display_name`, `source = 'whatsapp'`, and `lead_status = 'nuevo'`
- **AND** SHALL return the created contact

#### Scenario: Existing contact is returned on duplicate phone

- **WHEN** a message arrives from a phone number that already exists in `crm.contacts` for the organization
- **THEN** the system SHALL return the existing contact without overwriting email, company_id, source, lead_status, job_title, assigned_to, tipo_documento, or numero_documento
- **AND** SHALL NOT create a duplicate row
- **AND** SHALL update display_name, avatar_url, and last_message_at if the incoming values are non-empty/newer

#### Scenario: Contact created without phone for Instagram

- **WHEN** an `instagram.message.received` event creates a new contact
- **THEN** the contact SHALL be created with `phone_number` NULL and `instagram_user_id` set

#### Scenario: Null phone rows do not block WhatsApp contacts

- **WHEN** a contact has `phone_number` NULL and a new WhatsApp message arrives for the same organization
- **THEN** the partial unique index SHALL NOT reject the new phone-bearing row

### Requirement: Conversation entity with status tracking

The system SHALL store conversations in `crm.conversations` scoped by `organization_id` and linked to a `contact_id`, with a `channel` column (CHECK: whatsapp/instagram, default whatsapp) and status values of `active`, `closed`, or `archived`.

#### Scenario: New conversation created when no active window exists

- **WHEN** a message arrives and no `active` conversation exists for the contact and channel with `last_message_at` within the last 24 hours
- **THEN** the system SHALL create a new conversation with `status = 'active'` and `last_message_at` set to the message timestamp

#### Scenario: Active conversation matched within 24-hour window

- **WHEN** a message arrives and an `active` conversation exists for the contact and channel with `last_message_at` within the last 24 hours
- **THEN** the system SHALL return the existing conversation
- **AND** SHALL update `last_message_at` to the new message timestamp

#### Scenario: Closed conversations are not matched

- **WHEN** a message arrives and the most recent conversation for the contact and channel has `status = 'closed'`
- **THEN** the system SHALL create a new conversation with `status = 'active'`

### Requirement: Message entity with channel and provider id fields

The system SHALL store inbound messages in `crm.messages` with `organization_id`, `conversation_id`, `contact_id`, `channel` (CHECK: whatsapp/instagram, default whatsapp), `provider_message_id` (renamed from `whatsapp_message_id`, unique per org and channel), `direction`, `message_type`, `content`, and `status`.

#### Scenario: WhatsApp text message is persisted

- **WHEN** a `whatsapp.message.received` event arrives with `message_type = 'text'` and a text body
- **THEN** the system SHALL insert a row into `crm.messages` with `channel = 'whatsapp'`, `direction = 'inbound'`, `content` set to the text body, and `status = 'received'`

#### Scenario: Instagram text message is persisted

- **WHEN** an `instagram.message.received` event arrives with `message_type = 'text'` and a text body
- **THEN** the system SHALL insert a row into `crm.messages` with `channel = 'instagram'`, `direction = 'inbound'`, `content` set to the text body, and `status = 'received'`
- **AND** `provider_message_id` SHALL be set to the Instagram `mid`

#### Scenario: Media message is persisted with URL

- **WHEN** an inbound message event arrives with `message_type = 'image'` and a media URL
- **THEN** the system SHALL insert a row with `content` set to the media URL and the message_type set accordingly

#### Scenario: Duplicate provider id is silently skipped

- **WHEN** an inbound message event arrives with a `provider_message_id` that already exists for the organization and channel
- **THEN** the system SHALL NOT insert a duplicate row
- **AND** SHALL return nil (no error)

### Requirement: Event subscriber processes inbound message events asynchronously

The system SHALL subscribe to `whatsapp.message.received` and `instagram.message.received` events on the eventbus and process them asynchronously into CRM records.

#### Scenario: Subscriber processes a valid MessageReceived event

- **WHEN** a `whatsapp.message.received` event is published to the eventbus
- **THEN** the CRM subscriber SHALL receive the event
- **AND** SHALL canonicalize the sender phone number to E.164
- **AND** SHALL upsert the contact
- **AND** SHALL resolve or create the conversation
- **AND** SHALL persist the message

#### Scenario: Subscriber processes an Instagram MessageReceived event

- **WHEN** an `instagram.message.received` event is published to the eventbus
- **THEN** the CRM subscriber SHALL receive the event
- **AND** SHALL upsert the contact by `instagram_user_id`
- **AND** SHALL resolve or create the conversation for `channel = 'instagram'`
- **AND** SHALL persist the message with the IG `mid` as `provider_message_id`

#### Scenario: Subscriber handles non-text message types

- **WHEN** an inbound message event arrives with `message_type = 'image'`, `'video'`, `'audio'`, `'document'`, `'location'`, `'sticker'`, or `'interactive'`
- **THEN** the system SHALL persist the message with the appropriate `message_type` and content/metadata

#### Scenario: Subscriber errors do not crash the eventbus

- **WHEN** the CRM subscriber encounters a database error during processing
- **THEN** the system SHALL return the error to the eventbus
- **AND** the eventbus SHALL log the error and continue processing other events
