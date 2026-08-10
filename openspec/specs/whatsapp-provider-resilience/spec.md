## Purpose

Defines resilience of the Meta Graph API integration: circuit breaker, retry with backoff, and send-status feedback.

## Requirements

### Requirement: Circuit breaker for Meta Graph API

The system SHALL protect outbound Meta Graph API calls with a circuit breaker so a Meta outage does not cascade into backend failures.

#### Scenario: Breaker opens after repeated failures

- **WHEN** Meta Graph API calls fail more than the configured threshold within the window
- **THEN** the breaker SHALL open
- **AND** subsequent calls SHALL fail fast without hitting Meta

#### Scenario: Half-open probe

- **WHEN** the breaker is open and the cooldown elapses
- **THEN** the breaker SHALL allow a limited number of probe calls
- **AND** SHALL close on probe success, SHALL reopen on probe failure

#### Scenario: Fail-fast error surfaced

- **WHEN** a call is blocked by an open breaker
- **THEN** the caller SHALL receive a typed error indicating the breaker is open
- **AND** the failed send SHALL be routed to the durable retry queue rather than dropped

### Requirement: Retry with backoff for message sends

The system SHALL retry failed outbound WhatsApp message sends with exponential backoff and jitter, up to a configured maximum, using a durable queue so sends survive process restarts.

#### Scenario: Transient failure retried

- **WHEN** a message send fails with a transient error (timeout, 5xx, rate limit)
- **THEN** the system SHALL enqueue the send for retry with exponential backoff and jitter
- **AND** the send SHALL survive a backend restart

#### Scenario: Permanent failure dead-lettered

- **WHEN** a message send fails with a permanent error (invalid token, invalid phone number)
- **THEN** the system SHALL NOT retry it
- **AND** SHALL record the failure on the message record

#### Scenario: Max attempts exhausted

- **WHEN** a message send exhausts the configured maximum retry attempts
- **THEN** the system SHALL mark the send as failed
- **AND** SHALL record the last error for operator review

### Requirement: Send status feedback to senders

The system SHALL record outbound message send state transitions so operators can observe delivery health.

#### Scenario: Send state recorded

- **WHEN** an outbound message transitions between queued, sent, and failed
- **THEN** the system SHALL persist the state and error details
- **AND** SHALL correlate it with the Meta `statuses` webhook updates when available
