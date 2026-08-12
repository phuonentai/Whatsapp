## ADDED Requirements


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
