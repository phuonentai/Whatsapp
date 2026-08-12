## MODIFIED Requirements

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

#### Scenario: Client sends a template message

- **WHEN** `client.SendTemplateMessage(ctx, "access_token", "https://graph.facebook.com", "v21.0", "12345", "+573001234567", "confirmacion_pedido", "es", ["María", "Pedido #1234"])` is called with an approved template name, language, and parameters
- **THEN** the client SHALL POST the Cloud API messages endpoint with `"type": "template"` and the template components payload (`{"name": "confirmacion_pedido", "language": {"policy": "deterministic", "code": "es"}, "components": [{"type": "body", "parameters": [{"type": "text", "text": "María"}, {"type": "text", "text": "Pedido #1234"}]}]}`)
- **AND** on a successful response (HTTP 200/201 with `{"messages": [{"id": "wamid.xxx"}]}`) SHALL return the message ID `"wamid.xxx"` and `nil` error
