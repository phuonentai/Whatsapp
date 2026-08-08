## ADDED Requirements

### Requirement: Duplicate webhook deliveries do not create duplicate CRM messages

The system SHALL ensure that a retried or duplicated webhook delivery does not create a duplicate `crm.messages` row. Message persistence for inbound WhatsApp messages SHALL be idempotent on `(organization_id, whatsapp_message_id)` using `INSERT ... ON CONFLICT DO NOTHING` as the primary operation, with a subsequent fetch of the existing message when the insert is a no-op.

#### Scenario: Retried webhook for an already-stored message

- **WHEN** a webhook delivery is processed for a `whatsapp_message_id` already stored for the organization (e.g., a network retry)
- **THEN** the system SHALL reuse the existing message row
- **AND** SHALL NOT return an error and SHALL NOT create a second row

#### Scenario: Concurrent deliveries of the same message

- **WHEN** two deliveries of the same message are processed concurrently
- **THEN** exactly one `crm.messages` row SHALL be persisted
- **AND** the processing SHALL complete without a unique-violation error
