## ADDED Requirements

### Requirement: Contact entity with organization scoping

The system SHALL store contacts in `crm.contacts` scoped by `organization_id` with a unique constraint on `(organization_id, phone_number)`.

#### Scenario: New contact is created from webhook message

- **WHEN** a message arrives from a phone number that does not exist in `crm.contacts` for the organization
- **THEN** the system SHALL create a new contact with the E.164 phone number and default `display_name`
- **AND** SHALL return the created contact

#### Scenario: Existing contact is returned on duplicate phone

- **WHEN** a message arrives from a phone number that already exists in `crm.contacts` for the organization
- **THEN** the system SHALL return the existing contact without modification
- **AND** SHALL not create a duplicate row

### Requirement: Conversation entity with status tracking

The system SHALL store conversations in `crm.conversations` scoped by `organization_id` and linked to a `contact_id`, with status values of `active`, `closed`, or `archived`.

#### Scenario: New conversation created when no active window exists

- **WHEN** a message arrives and no `active` conversation exists for the contact with `last_message_at` within the last 24 hours
- **THEN** the system SHALL create a new conversation with `status = 'active'` and `last_message_at` set to the message timestamp

#### Scenario: Active conversation matched within 24-hour window

- **WHEN** a message arrives and an `active` conversation exists for the contact with `last_message_at` within the last 24 hours
- **THEN** the system SHALL return the existing conversation
- **AND** SHALL update `last_message_at` to the new message timestamp

#### Scenario: Closed conversations are not matched

- **WHEN** a message arrives and the most recent conversation for the contact has `status = 'closed'`
- **THEN** the system SHALL create a new conversation with `status = 'active'`

### Requirement: Message entity with WhatsApp-specific fields

The system SHALL store inbound messages in `crm.messages` with `organization_id`, `conversation_id`, `contact_id`, `whatsapp_message_id` (unique per org), `direction`, `message_type`, `content`, and `status`.

#### Scenario: Text message is persisted

- **WHEN** a `whatsapp.message.received` event arrives with `message_type = 'text'` and a text body
- **THEN** the system SHALL insert a row into `crm.messages` with `direction = 'inbound'`, `content` set to the text body, and `status = 'received'`

#### Scenario: Media message is persisted with URL

- **WHEN** a `whatsapp.message.received` event arrives with `message_type = 'image'` and a media URL
- **THEN** the system SHALL insert a row with `content` set to the media URL and the message_type set accordingly

#### Scenario: Duplicate whatsapp_message_id is silently skipped

- **WHEN** a `whatsapp.message.received` event arrives with a `whatsapp_message_id` that already exists for the organization
- **THEN** the system SHALL NOT insert a duplicate row
- **AND** SHALL return nil (no error)

### Requirement: Event subscriber processes MessageReceived events asynchronously

The system SHALL subscribe to `whatsapp.message.received` events on the eventbus and process them asynchronously into CRM records.

#### Scenario: Subscriber processes a valid MessageReceived event

- **WHEN** a `whatsapp.message.received` event is published to the eventbus
- **THEN** the CRM subscriber SHALL receive the event
- **AND** SHALL canonicalize the sender phone number to E.164
- **AND** SHALL upsert the contact
- **AND** SHALL resolve or create the conversation
- **AND** SHALL persist the message

#### Scenario: Subscriber handles non-text message types

- **WHEN** a `whatsapp.message.received` event arrives with `message_type = 'image'`, `'video'`, `'audio'`, `'document'`, `'location'`, `'sticker'`, or `'interactive'`
- **THEN** the system SHALL persist the message with the appropriate `message_type` and content/metadata

#### Scenario: Subscriber errors do not crash the eventbus

- **WHEN** the CRM subscriber encounters a database error during processing
- **THEN** the system SHALL return the error to the eventbus
- **AND** the eventbus SHALL log the error and continue processing other events

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
