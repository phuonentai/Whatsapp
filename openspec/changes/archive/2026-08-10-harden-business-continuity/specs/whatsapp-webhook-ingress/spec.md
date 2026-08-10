## MODIFIED Requirements

### Requirement: Webhook endpoint accepts WhatsApp Cloud API payloads

The system SHALL expose a `POST /api/v1/webhooks/whatsapp` endpoint that accepts WhatsApp Cloud API webhook JSON payloads.

#### Scenario: Valid webhook with text message

- **WHEN** a POST request arrives at `/api/v1/webhooks/whatsapp` with a valid `x-hub-signature-256` header and a well-formed WhatsApp Cloud API JSON body containing a text message
- **THEN** the system SHALL return HTTP 200 with no body
- **AND** the system SHALL durably enqueue a `whatsapp.message.received` event by persisting it to the outbox in the same database transaction as the webhook log row
- **AND** the event SHALL be dispatched asynchronously after the HTTP response is committed

#### Scenario: Webhook verification challenge (hub.mode=subscribe)

- **WHEN** a GET request arrives at `/api/v1/webhooks/whatsapp?hub.mode=subscribe&hub.verify_token=<token>&hub.challenge=<challenge>`
- **THEN** the system SHALL compare `hub.verify_token` against the configured verify token for the resolved organization
- **AND** if valid, return HTTP 200 with the `hub.challenge` value as the response body
- **AND** if invalid, return HTTP 403

#### Scenario: Missing or invalid HMAC signature

- **WHEN** a POST request arrives at `/api/v1/webhooks/whatsapp` without a valid `x-hub-signature-256` header or with a signature that does not match the computed HMAC-SHA256 of the raw body
- **THEN** the system SHALL return HTTP 401 with error code `invalid_signature`

#### Scenario: Unknown phone_number_id

- **WHEN** a POST request arrives with a valid signature but the `phone_number_id` in the payload metadata does not match any entry in `whatsapp.whatsapp_configs`
- **THEN** the system SHALL return HTTP 404 with error code `unknown_phone_number`

### Requirement: Raw webhook payload logging

The system SHALL store the raw webhook request body, headers, and processing metadata in `whatsapp.webhook_logs` BEFORE dispatching any event, atomically with the outbox entries in a single database transaction.

#### Scenario: Successful webhook logged and enqueued atomically

- **WHEN** a webhook passes signature validation and organization resolution
- **THEN** the system SHALL insert a row into `whatsapp.webhook_logs` with `status = 'received'`, the raw payload, and the resolved `organization_id`
- **AND** in the same transaction SHALL insert the corresponding outbox entries for the `whatsapp.message.received` events
- **AND** the system SHALL commit the transaction before returning HTTP 200

#### Scenario: Transaction failure prevents HTTP 200

- **WHEN** the webhook log or outbox insert fails and the transaction rolls back
- **THEN** the system SHALL return a non-2xx response so the provider retries delivery

#### Scenario: Failed webhook still logged

- **WHEN** a webhook fails signature validation or organization resolution
- **THEN** the system SHALL insert a row into `whatsapp.webhook_logs` with `status = 'failed'` and the error message

## ADDED Requirements

### Requirement: Webhook delivery deduplication

The system SHALL deduplicate webhook deliveries before dispatch so a retried delivery of the same payload is acknowledged without re-dispatching its events.

#### Scenario: Duplicate delivery acknowledged without re-dispatch

- **WHEN** a webhook delivery is processed whose payload (webhook ID / message ID set) was already processed
- **THEN** the system SHALL return HTTP 200
- **AND** SHALL NOT create new outbox entries or re-dispatch events

#### Scenario: First delivery is dispatched normally

- **WHEN** a webhook delivery is processed whose payload was not previously processed
- **THEN** the system SHALL persist webhook log + outbox entries and dispatch events as normal

### Requirement: Outbox dispatch retry and dead-letter

The system SHALL dispatch outbox events asynchronously with retry on failure, and SHALL move permanently failing events to a dead-letter state after exhausting retries.

#### Scenario: Handler failure triggers retry

- **WHEN** an outbox event handler fails
- **THEN** the event SHALL remain in the outbox and be retried with exponential backoff up to the configured maximum attempts

#### Scenario: Exhausted retries move to dead-letter

- **WHEN** an outbox event has failed all configured retry attempts
- **THEN** the system SHALL mark the event dead-lettered with the last error recorded
- **AND** the event SHALL NOT be dispatched again automatically

#### Scenario: Process restart resumes pending events

- **WHEN** the backend restarts with pending outbox events
- **THEN** the system SHALL resume dispatching un-dispatched and retryable events
- **AND** SHALL NOT lose events that were committed before the restart

### Requirement: Outbox event replay

The system SHALL support replaying dead-lettered or lost events from the raw webhook payloads stored in `whatsapp.webhook_logs`.

#### Scenario: Replay of a dead-lettered message event

- **WHEN** an operator triggers replay for a dead-lettered message event
- **THEN** the system SHALL re-enqueue the event from its stored raw payload
- **AND** SHALL record the replay action in the webhook log metadata
