## ADDED Requirements

### Requirement: Outbound text message sending via WhatsApp Cloud API

The system SHALL support sending text messages through the WhatsApp Cloud API (`POST https://graph.facebook.com/{api_version}/{phone_number_id}/messages`) using the organization's stored credentials.

#### Scenario: Successful text message send

- **WHEN** a POST request is sent to `/crm/conversaciones/:id/mensajes` with body `{"content": "Hello, how can I help?"}`
- **AND** the organization has an active WhatsApp config with valid `access_token`, `phone_number_id`, `waba_id`, `graph_api_url`, and `api_version`
- **THEN** the system SHALL call the WhatsApp Cloud API with `{"messaging_product": "whatsapp", "recipient_type": "individual", "to": "<contact_phone>", "type": "text", "text": {"body": "Hello, how can I help?"}}`
- **AND** on successful API response (HTTP 200/201 with `{"messages": [{"id": "wamid.xxx"}]}`), SHALL persist the outbound message in `crm.messages` with `direction = 'outbound'`, `status = 'sent'`, and `whatsapp_message_id` set to the returned message ID
- **AND** return HTTP 200 with the created message

#### Scenario: WhatsApp config is missing or inactive

- **WHEN** the organization has no WhatsApp config or `is_active = false`
- **THEN** the system SHALL return HTTP 400 with error code `whatsapp_not_configured`

#### Scenario: Missing access_token

- **WHEN** the WhatsApp config exists but `access_token` is empty
- **THEN** the system SHALL return HTTP 400 with error code `whatsapp_no_access_token`

#### Scenario: 24-hour messaging window closed

- **WHEN** the conversation's `last_message_at` is more than 24 hours in the past
- **THEN** the system SHALL return HTTP 200 with the sent message
- **AND** SHALL include a `warning` field in the response: `"outside_24h_window"` (template message support not yet implemented)

#### Scenario: WhatsApp API returns an error

- **WHEN** the WhatsApp Cloud API returns an error response (4xx/5xx)
- **THEN** the system SHALL NOT persist the message
- **AND** SHALL return HTTP 502 with the WhatsApp API error details as `{"error": {"code": "whatsapp_api_error", "message": "...", "api_error": {...}}}`

### Requirement: WhatsApp Cloud API HTTP client

The system SHALL provide a reusable WhatsApp Cloud API HTTP client (`pkg/whatsapp/client.go`) with Bearer token authentication, configurable base URL, and circuit breaker.

#### Scenario: Client constructs correct request

- **WHEN** `client.SendTextMessage(ctx, "access_token", "https://graph.facebook.com", "v21.0", "12345", "+573001234567", "Hello")` is called
- **THEN** the client SHALL send `POST https://graph.facebook.com/v21.0/12345/messages` with header `Authorization: Bearer access_token` and Content-Type `application/json`
- **AND** the body SHALL contain the WhatsApp Cloud API text message payload

#### Scenario: Circuit breaker opens after repeated failures

- **WHEN** the WhatsApp Cloud API returns 5xx errors for 5 consecutive calls within a 10-second window
- **THEN** the circuit breaker SHALL open
- **AND** subsequent calls SHALL return an error immediately without making HTTP requests
- **AND** after 30 seconds, the circuit SHALL transition to half-open to probe recovery

#### Scenario: Success response is parsed

- **WHEN** the WhatsApp Cloud API returns `{"messaging_product": "whatsapp", "contacts": [{"input": "+57...", "wa_id": "57..."}], "messages": [{"id": "wamid.HBgNNTc..."}]}`
- **THEN** `SendTextMessage` SHALL return the message ID `"wamid.HBgNNTc..."` and `nil` error

### Requirement: Send message API endpoint

The system SHALL expose a `POST /crm/conversaciones/:id/mensajes` endpoint that sends an outbound text message via the WhatsApp Cloud API.

#### Scenario: Send text message to conversation contact

- **WHEN** an authenticated user sends `POST /crm/conversaciones/42/mensajes` with body `{"content": "Thank you for your inquiry"}`
- **AND** conversation 42 belongs to the user's organization and has a contact with a valid phone number
- **AND** the organization's WhatsApp config is active and has a valid `access_token`
- **THEN** the system SHALL send the message via WhatsApp Cloud API, persist it, and return HTTP 200

#### Scenario: Reply is rate-limited

- **WHEN** a user sends replies too rapidly (more than 10 messages in 10 seconds)
- **THEN** the system SHALL return HTTP 429 with error code `rate_limit`
