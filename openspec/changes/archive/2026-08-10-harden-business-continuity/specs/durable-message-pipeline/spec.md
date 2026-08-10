## ADDED Requirements

### Requirement: Durable outbox for WhatsApp message events

The system SHALL persist WhatsApp message events (`whatsapp.message.received`, `whatsapp.message.echo`) to a durable outbox table in the same transaction as the webhook log, and SHALL dispatch them asynchronously.

#### Scenario: Event persisted before ack

- **WHEN** a WhatsApp webhook is processed
- **THEN** the outbox entries for its events SHALL be committed to the database before the HTTP 200 response is sent

#### Scenario: Crash after commit does not lose events

- **WHEN** the backend crashes after committing the webhook transaction but before dispatching
- **THEN** the events SHALL be dispatched from the outbox after restart

#### Scenario: No dispatch in the request path

- **WHEN** a webhook is processed
- **THEN** event handlers SHALL NOT run synchronously inside the webhook HTTP handler

### Requirement: Outbox dispatcher

The system SHALL run a dispatcher that polls the outbox and invokes event handlers with retry and backoff.

#### Scenario: Successful dispatch

- **WHEN** the dispatcher picks up a pending outbox event
- **THEN** it SHALL invoke the subscribed event handler
- **AND** SHALL mark the event dispatched only after the handler succeeds

#### Scenario: Failed dispatch is retried

- **WHEN** a handler returns an error
- **THEN** the event SHALL remain pending
- **AND** SHALL be retried with exponential backoff up to the configured maximum attempts
- **AND** the retry count and last error SHALL be recorded on the outbox row

#### Scenario: Dead-letter after max attempts

- **WHEN** an event exceeds the maximum retry attempts
- **THEN** the event SHALL be marked dead-lettered
- **AND** SHALL NOT be dispatched again automatically

### Requirement: Outbox event ordering and concurrency safety

The system SHALL dispatch outbox events for the same organization without double-processing, and SHALL guard concurrent dispatchers.

#### Scenario: Two dispatcher instances do not double-dispatch

- **WHEN** two dispatcher processes claim the same outbox event concurrently
- **THEN** exactly one SHALL process it (transaction-isolated claim, e.g., `FOR UPDATE SKIP LOCKED`)
- **AND** the event SHALL NOT be dispatched twice

#### Scenario: Idempotent handler completion

- **WHEN** a handler succeeds but the dispatcher fails to mark completion
- **THEN** the event SHALL be retried
- **AND** downstream effects SHALL remain idempotent (no duplicate CRM rows or sends)

### Requirement: Replay from webhook logs

The system SHALL allow re-enqueueing events from the raw payloads stored in `whatsapp.webhook_logs`.

#### Scenario: Replay re-enqueues events

- **WHEN** an operator requests replay for a webhook log entry
- **THEN** the system SHALL recreate the outbox entries for that payload
- **AND** SHALL record the replay in the log metadata
