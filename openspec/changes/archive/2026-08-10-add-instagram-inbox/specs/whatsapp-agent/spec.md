## MODIFIED Requirements

### Requirement: Inbound message triggers the agent pipeline

The system SHALL start or advance a conversation flow whenever a `whatsapp.message.received` or `instagram.message.received` event is published, by subscribing to the same events consumed by the CRM message listeners. The pipeline SHALL resolve the contact and active conversation idempotently (independent of the CRM listener's event ordering) and SHALL NOT block the webhook HTTP response (the eventbus dispatches handlers asynchronously).

#### Scenario: Inbound WhatsApp message starts a flow

- **WHEN** a `whatsapp.message.received` event arrives for a contact
- **THEN** the system SHALL create or reuse an active `conversation_flows` row in `running` status
- **AND** SHALL NOT modify the webhook ingress response

#### Scenario: Inbound Instagram message starts a flow

- **WHEN** an `instagram.message.received` event arrives for a contact
- **THEN** the system SHALL create or reuse an active `conversation_flows` row in `running` status
- **AND** SHALL process the message with the IG `mid` as the provider message id
- **AND** SHALL NOT modify the webhook ingress response

#### Scenario: Redelivered webhook does not double-process

- **WHEN** a retried webhook delivers a message whose provider message id already has a pending suggestion
- **THEN** the system SHALL skip processing for that message
- **AND** SHALL NOT create a second suggestion or send

#### Scenario: Non-text messages are ignored

- **WHEN** a message event has a non-text type or empty content
- **THEN** the pipeline SHALL return without analysis or suggestions
