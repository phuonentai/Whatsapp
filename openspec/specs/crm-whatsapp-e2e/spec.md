## Purpose

Define the E2E behavior of WhatsApp webhook simulation: payload building and HMAC-SHA256 signing, inbound message rendering in the inbox UI, idempotent duplicate delivery, signature/unknown-number rejection, the verification handshake, and observable webhook_logs stats.
## Requirements
### Requirement: WhatsApp webhook simulation helper

The system SHALL provide a Playwright test helper (`e2e/helpers/whatsapp.ts`) that builds WhatsApp Cloud API webhook payloads and signs them with HMAC-SHA256 for delivery to the real webhook endpoint.

#### Scenario: Payload builder produces Cloud API shape

- **WHEN** the helper builds a payload for a text message
- **THEN** the payload SHALL contain `entry[].changes[].value.metadata.phone_number_id` matching a seeded `whatsapp.whatsapp_configs` row
- **AND** SHALL contain `entry[].changes[].value.messages[].id`, `from`, `type = "text"`, `text.body`, and `timestamp`

#### Scenario: HMAC-SHA256 signer produces valid signature

- **WHEN** the helper signs a raw body with the org's seeded `webhook_secret`
- **THEN** the signature SHALL be `x-hub-signature-256: sha256=<hex HMAC-SHA256 of the raw body>`
- **AND** SHALL pass the backend's `VerifySignature` constant-time comparison

### Requirement: Inbound WhatsApp message renders in the inbox UI

The system SHALL verify that a simulated inbound text webhook delivery results in the message appearing in `/dashboard/inbox` through the real eventbus → `crm.messages` → conversations API path.

#### Scenario: Signed webhook creates visible conversation

- **WHEN** a validly signed inbound text message webhook is POSTed to `/api/v1/webhooks/whatsapp` for a seeded org
- **THEN** the webhook SHALL return HTTP 200 with no body
- **AND** a conversation SHALL appear in the inbox conversation list for that org
- **AND** opening the conversation SHALL display the message content with `direction = inbound`

### Requirement: Duplicate webhook delivery persists a single message

The system SHALL verify idempotent persistence when the same `provider_message_id` is delivered more than once.

#### Scenario: Retried delivery of the same message

- **WHEN** a webhook with `provider_message_id` already stored is delivered a second time
- **THEN** the webhook SHALL return HTTP 200
- **AND** exactly one message SHALL be visible in the conversation thread
- **AND** the conversation SHALL not display a duplicate message

### Requirement: Invalid HMAC signature is rejected

The system SHALL verify that a tampered or malformed signature is rejected and produces no message in the inbox.

#### Scenario: Tampered body or invalid signature header

- **WHEN** a webhook is delivered with a signature that does not match the HMAC-SHA256 of the raw body, or with a malformed `x-hub-signature-256` header
- **THEN** the webhook SHALL return HTTP 401 with error code `invalid_signature`
- **AND** no conversation or message SHALL appear in the inbox UI for that delivery

### Requirement: Unknown phone_number_id is rejected

The system SHALL verify that a validly signed webhook for an unconfigured `phone_number_id` is rejected.

#### Scenario: Signed webhook for unconfigured number

- **WHEN** a webhook carries a valid signature but a `phone_number_id` with no matching active `whatsapp.whatsapp_configs` row
- **THEN** the webhook SHALL return HTTP 404 with error code `unknown_phone_number`
- **AND** no conversation or message SHALL be created

### Requirement: Webhook verification challenge handshake

The system SHALL verify the `hub.mode=subscribe` subscription handshake returns the challenge for a valid `verify_token`.

#### Scenario: Valid verify_token returns challenge

- **WHEN** a GET request arrives at `/api/v1/webhooks/whatsapp` with `hub.mode=subscribe`, a `hub.verify_token` matching the seeded config, and a `hub.challenge`
- **THEN** the system SHALL return HTTP 200 with the exact `hub.challenge` string as the response body

#### Scenario: Invalid verify_token is rejected

- **WHEN** a GET request arrives with a `hub.verify_token` that does not match the seeded config
- **THEN** the system SHALL return HTTP 403

### Requirement: webhook_logs stats observable after simulated delivery

The system SHALL verify that simulated deliveries are recorded in `whatsapp.webhook_logs` and surfaced through the config health endpoint.

#### Scenario: Successful delivery reflected in health stats

- **WHEN** a validly signed webhook has been delivered to a seeded org
- **THEN** `GET /api/v1/whatsapp/config/health` for that org SHALL return stats reflecting the received webhook



### Requirement: Webhook edge-case scenarios are E2E-tested

The system SHALL cover the webhook error and boundary scenarios with E2E tests that exercise the real `/api/v1/webhooks/whatsapp` endpoint: inactive config, invalid `hub.verify_token` handshake, malformed JSON body, failed-webhook logging, inbound direction labeling, and echo handling.

#### Scenario: Inactive config returns 404

- **WHEN** a signed webhook is delivered for a `phone_number_id` whose config has `is_active = false`
- **THEN** the response SHALL have status 404 with error code `unknown_phone_number`

#### Scenario: Invalid verify_token handshake returns 403

- **WHEN** a GET handshake request arrives with a `hub.verify_token` that does not match any active config
- **THEN** the response SHALL have status 403

#### Scenario: Malformed JSON payload returns 400

- **WHEN** a POST request arrives with a valid signature but a non-JSON body
- **THEN** the response SHALL have status 400 with error code `invalid_json`

#### Scenario: Failed webhook is logged

- **WHEN** a webhook with a valid known `phone_number_id` but invalid HMAC signature is delivered
- **THEN** the response SHALL have status 401
- **AND** the organization's webhook health stats SHALL reflect a `failed` status row

#### Scenario: Inbound message carries direction=inbound

- **WHEN** a valid signed inbound text webhook is delivered
- **THEN** the persisted message retrieved via `/crm/conversaciones/:id/mensajes` SHALL have `direction` equal to `inbound`

#### Scenario: Echo messages are not rendered as inbound

- **WHEN** a signed webhook delivers a message with `origin.type = "echo"`
- **THEN** no inbound message row SHALL be persisted for that `whatsapp_message_id`
