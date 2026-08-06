## ADDED Requirements

### Requirement: Webhook endpoint accepts WhatsApp Cloud API payloads

The system SHALL expose a `POST /api/v1/webhooks/whatsapp` endpoint that accepts WhatsApp Cloud API webhook JSON payloads.

#### Scenario: Valid webhook with text message

- **WHEN** a POST request arrives at `/api/v1/webhooks/whatsapp` with a valid `x-hub-signature-256` header and a well-formed WhatsApp Cloud API JSON body containing a text message
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

### Requirement: HMAC-SHA256 signature validation

The system SHALL verify WhatsApp webhook signatures using HMAC-SHA256 by computing `sha256(body)` with the organization's webhook secret and comparing against the `x-hub-signature-256` header value in constant time.

#### Scenario: Valid signature passes verification

- **WHEN** the signature header value is `sha256=<hex_digest>` and the computed HMAC-SHA256 of the raw request body matches
- **THEN** the system SHALL proceed with request processing

#### Scenario: Invalid signature is rejected

- **WHEN** the computed HMAC-SHA256 does not match the signature header value
- **THEN** the system SHALL return HTTP 401 with error code `invalid_signature`

### Requirement: Organization resolution from webhook metadata

The system SHALL resolve the `organization_id` by extracting `phone_number_id` from `entry[].changes[].value.metadata.phone_number_id` in the webhook JSON payload and looking it up in `whatsapp.whatsapp_configs`.

#### Scenario: Known phone_number_id resolves to organization

- **WHEN** the payload contains `metadata.phone_number_id` matching a row in `whatsapp.whatsapp_configs` with `is_active = true`
- **THEN** the system SHALL set the resolved `organization_id` for subsequent processing

#### Scenario: Inactive config returns no organization

- **WHEN** the `phone_number_id` matches a row with `is_active = false`
- **THEN** the system SHALL return HTTP 404 with error code `unknown_phone_number`

### Requirement: Webhook verification challenge (subscription handshake)

The system SHALL support WhatsApp's webhook subscription verification flow via GET requests containing `hub.mode`, `hub.verify_token`, and `hub.challenge` query parameters.

#### Scenario: Valid verify_token returns challenge

- **WHEN** a GET request arrives with `hub.mode=subscribe`, a `hub.verify_token` matching the org's configured `verify_token`, and a `hub.challenge` value
- **THEN** the system SHALL return HTTP 200 with the exact `hub.challenge` string as the response body

#### Scenario: Invalid verify_token is rejected

- **WHEN** a GET request arrives with a `hub.verify_token` that does not match any active config
- **THEN** the system SHALL return HTTP 403

### Requirement: Raw webhook payload logging

The system SHALL store the raw webhook request body, headers, and processing metadata in `whatsapp.webhook_logs` BEFORE publishing the event to the eventbus.

#### Scenario: Successful webhook logged and processed

- **WHEN** a webhook passes signature validation and organization resolution
- **THEN** the system SHALL insert a row into `whatsapp.webhook_logs` with `status = 'received'`, the raw payload, and the resolved `organization_id`
- **AND** then SHALL publish the `whatsapp.message.received` event

#### Scenario: Failed webhook still logged

- **WHEN** a webhook fails signature validation or organization resolution
- **THEN** the system SHALL insert a row into `whatsapp.webhook_logs` with `status = 'failed'` and the error message
