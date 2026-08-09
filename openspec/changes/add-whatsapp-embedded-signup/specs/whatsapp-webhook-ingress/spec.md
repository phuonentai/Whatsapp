## MODIFIED Requirements

### Requirement: Webhook endpoint accepts WhatsApp Cloud API payloads

The system SHALL expose a `POST /api/v1/webhooks/whatsapp` endpoint that accepts WhatsApp Cloud API webhook JSON payloads.

#### Scenario: Valid webhook with text message

- **WHEN** a POST request arrives at `/api/v1/webhooks/whatsapp` with a valid `x-hub-signature-256` header and a well-formed WhatsApp Cloud API JSON body containing a non-echo text message
- **THEN** the system SHALL return HTTP 200 with no body
- **AND** the system SHALL publish a `whatsapp.message.received` event to the platform eventbus

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

## ADDED Requirements

### Requirement: Coexistence echo messages are mirrored as outbound

The system SHALL recognize coexistence echo messages (`messages` entries whose `origin.type` is `echo`, i.e. messages sent from the organization's phone WhatsApp Business app) and SHALL NOT treat them as inbound customer messages.

#### Scenario: Echo message published as echo event

- **WHEN** a POST request arrives at `/api/v1/webhooks/whatsapp` with a valid signature and a payload containing a `messages` entry with `origin.type = "echo"`
- **THEN** the system SHALL return HTTP 200 with no body
- **AND** the system SHALL publish a `whatsapp.message.echo` event to the platform eventbus
- **AND** the system SHALL NOT publish a `whatsapp.message.received` event for that entry

#### Scenario: Echo message persisted as outbound mirror

- **WHEN** a CRM listener processes a `whatsapp.message.echo` event
- **THEN** the system SHALL persist the message in `crm.messages` with `direction = 'outbound'` and the `whatsapp_message_id` from the echo payload
- **AND** the persistence SHALL be idempotent on `(organization_id, whatsapp_message_id)` using `INSERT ... ON CONFLICT DO NOTHING` with a subsequent fetch of the existing row on no-op

#### Scenario: Non-echo messages unchanged

- **WHEN** a `messages` entry has no `origin` or `origin.type` other than `echo`
- **THEN** the system SHALL process it as an inbound `whatsapp.message.received` event exactly as before

#### Scenario: Echo with unknown phone_number_id

- **WHEN** an echo payload arrives with a `phone_number_id` that does not match any active config
- **THEN** the system SHALL return HTTP 404 with error code `unknown_phone_number`
