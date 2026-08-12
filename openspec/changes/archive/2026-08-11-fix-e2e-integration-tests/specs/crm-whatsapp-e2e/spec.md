# Delta Spec: crm-whatsapp-e2e — fix-e2e-integration-tests

## MODIFIED Requirements

### Requirement: Duplicate webhook delivery persists a single message

The system SHALL verify idempotent persistence when the same `provider_message_id` is delivered more than once.

#### Scenario: Retried delivery of the same message

- **WHEN** a webhook with `provider_message_id` already stored is delivered a second time
- **THEN** the webhook SHALL return HTTP 200
- **AND** exactly one message SHALL be visible in the conversation thread
- **AND** the conversation SHALL not display a duplicate message
