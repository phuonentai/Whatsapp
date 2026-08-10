## Purpose

Defines Instagram DM integration via the Meta Instagram Messaging API (Instagram Graph API product `messaging`): config management with token expiry, webhook ingress with org resolution by IG user ID, outbound sending, provider resilience, and the settings frontend.

## ADDED Requirements

### Requirement: Auth-gated GET endpoint returns organization's Instagram config

The system SHALL expose a `GET /api/v1/instagram/config` endpoint protected by the standard auth middleware chain (auth, org_context, subscription) that returns the authenticated organization's Instagram configuration, masking `webhook_secret`, `verify_token`, and `access_token` to show only the first 6 and last 4 characters (e.g., `"token_****abcd"`).

#### Scenario: Config exists

- **WHEN** an authenticated user with `org:manage` permission requests `GET /api/v1/instagram/config`
- **AND** the organization has an Instagram config record
- **THEN** the system SHALL return HTTP 200 with the config object, including `ig_user_id`, `ig_username`, `fb_page_id`, `access_token` (masked), `token_expires_at`, `api_version`, `graph_api_url`, and `is_active`

#### Scenario: No config exists

- **WHEN** an authenticated user with `org:manage` permission requests `GET /api/v1/instagram/config`
- **AND** the organization has no Instagram config record
- **THEN** the system SHALL return HTTP 404 with error code `config_not_found`

#### Scenario: User lacks org:manage permission

- **WHEN** an authenticated user without `org:manage` permission requests `GET /api/v1/instagram/config`
- **THEN** the system SHALL return HTTP 403

### Requirement: Auth-gated PUT endpoint upserts Instagram config

The system SHALL expose a `PUT /api/v1/instagram/config` endpoint that creates or updates (upsert) the Instagram config for the authenticated organization, with partial update semantics preserving existing secret values when the request body omits them or sends masked/empty values.

#### Scenario: Create a new config

- **WHEN** an authenticated user with `org:manage` permission sends `PUT /api/v1/instagram/config` with a valid body containing `ig_user_id`, `access_token`, `webhook_secret`, and `verify_token`
- **AND** the organization has no existing Instagram config
- **THEN** the system SHALL insert a new row into `whatsapp.instagram_configs` scoped to the organization, including optional fields `ig_username`, `fb_page_id`, `token_expires_at`, `api_version`, and `graph_api_url` if provided
- **AND** the system SHALL return HTTP 200 with the created config (secrets and access_token masked)

#### Scenario: Update an existing config

- **WHEN** an authenticated user with `org:manage` permission sends `PUT /api/v1/instagram/config` with updated field values
- **AND** the organization already has an Instagram config
- **THEN** the system SHALL update the existing row using partial update semantics (only changed fields are updated; omitted fields retain their current values)
- **AND** empty or masked secret values SHALL preserve the existing stored secret

#### Scenario: Duplicate ig_user_id

- **WHEN** a `PUT /api/v1/instagram/config` request contains an `ig_user_id` that already belongs to a different organization
- **THEN** the system SHALL return HTTP 409 with error code `ig_user_id_conflict`

#### Scenario: Validation fails

- **WHEN** a `PUT /api/v1/instagram/config` request body is missing required fields (`ig_user_id`, `access_token`, `webhook_secret`, `verify_token`) for a new config
- **THEN** the system SHALL return HTTP 400 with a validation error message

### Requirement: Auth-gated PATCH endpoint toggles config active state

The system SHALL expose a `PATCH /api/v1/instagram/config/toggle` endpoint that toggles the `is_active` field of the organization's Instagram configuration.

#### Scenario: Toggle config off

- **WHEN** an authenticated user with `org:manage` permission sends `PATCH /api/v1/instagram/config/toggle`
- **AND** the config is currently active
- **THEN** the system SHALL set `is_active` to `false`
- **AND** return HTTP 200 with the updated config

#### Scenario: Toggle config on

- **WHEN** an authenticated user with `org:manage` permission sends `PATCH /api/v1/instagram/config/toggle`
- **AND** the config is currently inactive
- **THEN** the system SHALL set `is_active` to `true`
- **AND** return HTTP 200 with the updated config

#### Scenario: No config to toggle

- **WHEN** a user sends `PATCH /api/v1/instagram/config/toggle` but the organization has no config
- **THEN** the system SHALL return HTTP 404 with error code `config_not_found`

### Requirement: Token expiry exposure and manual refresh

The system SHALL expose `token_expires_at` on every config response, SHALL surface a warning when the token expires within 7 days, and SHALL provide a `POST /api/v1/instagram/config/refresh` endpoint that exchanges the current access token for a new long-lived token via the Meta `fb_exchange_token` grant.

#### Scenario: Token near expiry surfaces warning

- **WHEN** `GET /api/v1/instagram/config` returns a config whose `token_expires_at` is less than 7 days in the future (or null)
- **THEN** the response SHALL include `token_expiry_warning: true`

#### Scenario: Refresh succeeds

- **WHEN** an authenticated user with `org:manage` permission sends `POST /api/v1/instagram/config/refresh`
- **AND** the Meta token-exchange call succeeds
- **THEN** the system SHALL update `access_token` and `token_expires_at` in `whatsapp.instagram_configs`
- **AND** return HTTP 200 with the updated config (access_token masked)

#### Scenario: Refresh fails

- **WHEN** the Meta `fb_exchange_token` call fails (invalid or non-exchangeable token)
- **THEN** the system SHALL return HTTP 502 with error code `instagram_token_refresh_failed` including the Meta error details
- **AND** SHALL NOT modify the stored token

### Requirement: Webhook endpoint accepts Instagram Graph API payloads

The system SHALL expose a `POST /api/v1/webhooks/instagram` endpoint that accepts Instagram Messaging API webhook JSON payloads, and a `GET /api/v1/webhooks/instagram` endpoint for the subscription handshake.

#### Scenario: Valid webhook with text message

- **WHEN** a POST request arrives at `/api/v1/webhooks/instagram` with a valid `x-hub-signature-256` header and a well-formed payload whose `entry[].changes[].value` contains a `messages` array with a text message
- **THEN** the system SHALL return HTTP 200 with no body
- **AND** the system SHALL durably enqueue an `instagram.message.received` event by persisting it to the outbox in the same database transaction as the webhook log row
- **AND** the event SHALL be dispatched asynchronously after the HTTP response is committed

#### Scenario: Webhook verification challenge (hub.mode=subscribe)

- **WHEN** a GET request arrives at `/api/v1/webhooks/instagram?hub.mode=subscribe&hub.verify_token=<token>&hub.challenge=<challenge>`
- **THEN** the system SHALL compare `hub.verify_token` against the platform-configured `INSTAGRAM_WEBHOOK_VERIFY_TOKEN`, and additionally against any active config's `verify_token`
- **AND** if valid, return HTTP 200 with the `hub.challenge` value as the response body
- **AND** if invalid, return HTTP 403

#### Scenario: Missing or invalid HMAC signature

- **WHEN** a POST request arrives at `/api/v1/webhooks/instagram` without a valid `x-hub-signature-256` header or with a signature that does not match the computed HMAC-SHA256 of the raw body
- **THEN** the system SHALL return HTTP 401 with error code `invalid_signature`

#### Scenario: Unknown recipient ig_user_id

- **WHEN** a POST request arrives with a valid signature but the `recipient.id` in the payload does not match any active entry in `whatsapp.instagram_configs`
- **THEN** the system SHALL return HTTP 404 with error code `unknown_ig_user`

### Requirement: HMAC-SHA256 signature validation

The system SHALL verify Instagram webhook signatures using HMAC-SHA256 by computing `sha256(body)` with the resolved organization's `webhook_secret` and comparing against the `x-hub-signature-256` header value in constant time.

#### Scenario: Valid signature passes verification

- **WHEN** the signature header value is `sha256=<hex_digest>` and the computed HMAC-SHA256 of the raw request body matches
- **THEN** the system SHALL proceed with request processing

#### Scenario: Invalid signature is rejected

- **WHEN** the computed HMAC-SHA256 does not match the signature header value
- **THEN** the system SHALL return HTTP 401 with error code `invalid_signature`

### Requirement: Organization resolution from webhook payload

The system SHALL resolve the `organization_id` by extracting `recipient.id` from `entry[].changes[].value` in the webhook JSON payload and looking it up in `whatsapp.instagram_configs`.

#### Scenario: Known ig_user_id resolves to organization

- **WHEN** the payload's `value.recipient.id` matches a row in `whatsapp.instagram_configs` with `is_active = true`
- **THEN** the system SHALL set the resolved `organization_id` for subsequent processing

#### Scenario: Inactive config returns no organization

- **WHEN** the `recipient.id` matches a row with `is_active = false`
- **THEN** the system SHALL return HTTP 404 with error code `unknown_ig_user`

### Requirement: Raw webhook payload logging

The system SHALL store the raw webhook request body, headers, and processing metadata in `whatsapp.instagram_webhook_logs` BEFORE dispatching any event, atomically with the outbox entries in a single database transaction.

#### Scenario: Successful webhook logged and enqueued atomically

- **WHEN** a webhook passes signature validation and organization resolution
- **THEN** the system SHALL insert a row into `whatsapp.instagram_webhook_logs` with `status = 'received'`, the raw payload, and the resolved `organization_id`
- **AND** in the same transaction SHALL insert the corresponding outbox entries for the `instagram.message.received` events
- **AND** the system SHALL commit the transaction before returning HTTP 200

#### Scenario: Transaction failure prevents HTTP 200

- **WHEN** the webhook log or outbox insert fails and the transaction rolls back
- **THEN** the system SHALL return a non-2xx response so the provider retries delivery

#### Scenario: Failed webhook still logged

- **WHEN** a webhook fails signature validation or organization resolution
- **THEN** the system SHALL insert a row into `whatsapp.instagram_webhook_logs` with `status = 'failed'` and the error message

### Requirement: Duplicate webhook deliveries do not create duplicate CRM messages

The system SHALL ensure that a retried or duplicated Instagram webhook delivery does not create a duplicate `crm.messages` row. Message persistence for inbound Instagram messages SHALL be idempotent on `(organization_id, 'instagram', instagram_message_id)` using `INSERT ... ON CONFLICT DO NOTHING` as the primary operation, with a subsequent fetch of the existing message when the insert is a no-op.

#### Scenario: Retried webhook for an already-stored message

- **WHEN** a webhook delivery is processed for an `instagram_message_id` (the IG `mid`) already stored for the organization
- **THEN** the system SHALL reuse the existing message row
- **AND** SHALL NOT return an error and SHALL NOT create a second row

#### Scenario: Concurrent deliveries of the same message

- **WHEN** two deliveries of the same message are processed concurrently
- **THEN** exactly one `crm.messages` row SHALL be persisted
- **AND** the processing SHALL complete without a unique-violation error

### Requirement: Webhook delivery deduplication and replay

The system SHALL deduplicate Instagram webhook deliveries before dispatch using a `delivery_key` on `whatsapp.instagram_webhook_logs`, and SHALL support replaying dead-lettered events from stored raw payloads.

#### Scenario: Duplicate delivery acknowledged without re-dispatch

- **WHEN** a webhook delivery is processed whose delivery key was already processed
- **THEN** the system SHALL return HTTP 200
- **AND** SHALL NOT create new outbox entries or re-dispatch events

#### Scenario: Replay of a dead-lettered message event

- **WHEN** an operator triggers replay for a dead-lettered Instagram message event
- **THEN** the system SHALL re-enqueue the event from its stored raw payload
- **AND** SHALL record the replay action in the webhook log metadata

### Requirement: Echo messages persist as outbound CRM messages

Messages arriving with `is_echo = true` (sent from Meta Business Suite or elsewhere) SHALL NOT be published as inbound `instagram.message.received` events; instead the system SHALL publish an `instagram.message.echo` event that persists them as `direction = 'outbound'` CRM messages, idempotent on `(organization_id, 'instagram', instagram_message_id)`.

#### Scenario: Echo message persisted as outbound

- **WHEN** a webhook contains a message with `message.is_echo = true`
- **THEN** the system SHALL publish `instagram.message.echo`
- **AND** the CRM echo listener SHALL persist a `crm.messages` row with `direction = 'outbound'` and the IG `mid` as `provider_message_id`

#### Scenario: Non-echo message published as inbound

- **WHEN** a webhook contains a message without `is_echo`
- **THEN** the system SHALL publish `instagram.message.received` as normal

### Requirement: Outbound text message sending via Instagram Graph API

The system SHALL support sending text messages through the Instagram Graph API (`POST https://graph.facebook.com/{api_version}/{ig_user_id}/messages`) using the organization's stored credentials, routed from the existing send endpoint `POST /crm/conversaciones/:id/mensajes` when the conversation's `channel` is `instagram`.

#### Scenario: Successful text message send

- **WHEN** a POST request is sent to `/crm/conversaciones/:id/mensajes` with body `{"content": "Hello, how can I help?"}` for an Instagram-channel conversation
- **AND** the organization has an active Instagram config with valid `access_token` and `ig_user_id`
- **THEN** the system SHALL call the Instagram Graph API with `{"recipient": {"id": "<sender_ig_user_id>"}, "message": {"text": "Hello, how can I help?"}}`
- **AND** on successful API response, SHALL persist the outbound message in `crm.messages` with `direction = 'outbound'`, `status = 'sent'`, and `provider_message_id` set to the returned message id
- **AND** return HTTP 200 with the created message

#### Scenario: Instagram config is missing or inactive

- **WHEN** the conversation's channel is `instagram` and the organization has no Instagram config or `is_active = false`
- **THEN** the system SHALL return HTTP 400 with error code `instagram_not_configured`

#### Scenario: Missing access_token

- **WHEN** the Instagram config exists but `access_token` is empty
- **THEN** the system SHALL return HTTP 400 with error code `instagram_no_access_token`

#### Scenario: 24-hour messaging window closed

- **WHEN** the conversation's `last_message_at` is more than 24 hours in the past
- **THEN** the system SHALL return HTTP 200 with the sent message
- **AND** SHALL include a `warning` field in the response: `"outside_24h_window"`

#### Scenario: Instagram API returns an error

- **WHEN** the Instagram Graph API returns an error response (4xx/5xx)
- **THEN** the system SHALL NOT persist the message
- **AND** SHALL return HTTP 502 with the API error details as `{"error": {"code": "instagram_api_error", "message": "...", "api_error": {...}}}`

### Requirement: Instagram Graph API HTTP client

The system SHALL provide a reusable Instagram Graph API HTTP client with Bearer token authentication, configurable base URL, and circuit breaker (threshold 5, timeout 10s, half-open probe 2), exposed behind a mockable interface.

#### Scenario: Client constructs correct request

- **WHEN** the client's `SendTextMessage` is called with an access token, base URL, api version, `ig_user_id`, recipient id, and text
- **THEN** the client SHALL send `POST {base_url}/{api_version}/{ig_user_id}/messages` with header `Authorization: Bearer <access_token>` and Content-Type `application/json`
- **AND** the body SHALL contain the Instagram Messaging API text message payload

#### Scenario: Circuit breaker opens after repeated failures

- **WHEN** the Instagram Graph API returns 5xx errors for 5 consecutive calls within a 10-second window
- **THEN** the circuit breaker SHALL open
- **AND** subsequent calls SHALL return an error immediately without making HTTP requests
- **AND** after 30 seconds, the circuit SHALL transition to half-open to probe recovery

#### Scenario: GetIGUser resolves profile metadata

- **WHEN** the client's `GetIGUser` is called with an `ig_user_id` and access token
- **THEN** the client SHALL call `GET {base_url}/{api_version}/{ig_user_id}?fields=username,profile_picture_url`
- **AND** return the IG username and profile picture URL

#### Scenario: Token refresh constructs correct request

- **WHEN** the client's `RefreshToken` is called with app id, app secret, and the current access token
- **THEN** the client SHALL call `GET {base_url}/{api_version}/oauth/access_token?grant_type=fb_exchange_token&client_id=<app_id>&client_secret=<app_secret>&fb_exchange_token=<token>`
- **AND** return the new access token and expiry

### Requirement: Instagram management routes are registered in API provider

The system SHALL register Instagram routes in `internal/api/provider.go` so they are reachable at runtime.

#### Scenario: Instagram management routes respond to requests

- **WHEN** the API server starts
- **THEN** `GET /api/v1/instagram/config`, `PUT /api/v1/instagram/config`, `PATCH /api/v1/instagram/config/toggle`, and `POST /api/v1/instagram/config/refresh` SHALL be reachable with auth middleware applied and `org:manage` permission
- **AND** `GET /api/v1/webhooks/instagram` and `POST /api/v1/webhooks/instagram` SHALL be reachable without auth middleware

### Requirement: Instagram settings section in frontend

The workspace settings SHALL display an "Instagram" section at `/dashboard/settings?view=instagram` with a manual config form, visible to users with `org:manage` permission.

#### Scenario: Config exists — form is pre-populated

- **WHEN** a user navigates to `/dashboard/settings?view=instagram`
- **AND** the organization has an Instagram config
- **THEN** the form SHALL display `ig_user_id`, `ig_username`, `fb_page_id`, `token_expires_at` (with a warning when expiring within 7 days), `api_version`, `graph_api_url`, and `is_active` toggle
- **AND** the `webhook_secret`, `verify_token`, and `access_token` fields SHALL be masked as password inputs with a note "Leave blank to keep current value"
- **AND** a "Save" button SHALL be enabled when changes are made

#### Scenario: No config exists — empty form with onboarding guidance

- **WHEN** a user navigates to `/dashboard/settings?view=instagram`
- **AND** the organization has no Instagram config
- **THEN** the system SHALL display guidance explaining how to obtain the required values from Meta (Instagram Graph API access token with `instagram_manage_messages`, IG user ID, webhook verify token)
- **AND** the form fields SHALL be empty and editable

#### Scenario: Save config updates successfully

- **WHEN** a user fills in or edits config fields and clicks "Save"
- **THEN** the system SHALL send a `PUT /api/v1/instagram/config` request
- **AND** on success, SHALL show a success toast and refresh the displayed values (secrets and access_token remain masked)

#### Scenario: Refresh token button

- **WHEN** a config exists with `token_expires_at` and the user clicks "Refresh token"
- **THEN** the system SHALL send `POST /api/v1/instagram/config/refresh`
- **AND** on success, SHALL show a success toast and update the displayed expiry
- **AND** on failure, SHALL display an inline error

### Requirement: Instagram card in settings overview

The workspace settings overview at `/dashboard/settings` SHALL display an "Instagram" card visible to users with `org:manage` permission.

#### Scenario: User has org:manage and config exists

- **WHEN** a user with `org:manage` permission views the settings overview
- **AND** the organization has an active Instagram config
- **THEN** the overview SHALL show a card with title "Instagram", the connected IG username as the value, and "Active" as the status

#### Scenario: User has org:manage and no config exists

- **WHEN** a user with `org:manage` permission views the settings overview
- **AND** the organization has no Instagram config
- **THEN** the overview SHALL show a card with title "Instagram", value "Not connected", and helper text "Connect Instagram to receive DMs"

#### Scenario: User lacks org:manage permission

- **WHEN** a user without `org:manage` permission views the settings overview
- **THEN** the overview SHALL NOT display the Instagram card

### Requirement: Webhook callback URL and health indicator for Instagram

The Instagram config view SHALL display the webhook callback URL (`https://<origin>/api/v1/webhooks/instagram`) with a copy button, and a webhook health indicator based on recent `whatsapp.instagram_webhook_logs` activity.

#### Scenario: Callback URL is shown

- **WHEN** a user navigates to `/dashboard/settings?view=instagram`
- **THEN** the system SHALL display a read-only field showing the callback URL using the current `window.location.origin`
- **AND** a "Copy" button SHALL be available next to the URL

#### Scenario: Recent successful webhooks

- **WHEN** the organization has received Instagram webhooks in the last 24 hours
- **THEN** the system SHALL display a green indicator with "Webhooks active — last received {relative time}"

#### Scenario: No recent webhooks but config is active

- **WHEN** the config is active but no Instagram webhooks have been received in the last 24 hours
- **THEN** the system SHALL display a yellow indicator with "No webhooks received in the last 24 hours"
