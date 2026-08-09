## ADDED Requirements

### Requirement: Embedded signup bootstrap endpoint

The system SHALL expose a `GET /api/v1/whatsapp/signup/meta-config` endpoint protected by the standard auth middleware chain (auth, org_context, subscription) with `org:manage` permission that returns the Meta Embedded Signup bootstrap values (`app_id`, `config_id`, `redirect_uri`) required to initialize the Meta Business JS SDK in the browser.

#### Scenario: Authenticated request with org:manage

- **WHEN** an authenticated user with `org:manage` permission requests `GET /api/v1/whatsapp/signup/meta-config`
- **THEN** the system SHALL return HTTP 200 with `app_id`, `config_id`, and `redirect_uri` values from validated environment configuration

#### Scenario: Unauthenticated request

- **WHEN** a request arrives without a valid auth token
- **THEN** the system SHALL return HTTP 401

#### Scenario: User lacks org:manage permission

- **WHEN** an authenticated user without `org:manage` permission requests the endpoint
- **THEN** the system SHALL return HTTP 403

### Requirement: Embedded signup exchange endpoint

The system SHALL expose a `POST /api/v1/whatsapp/signup/exchange` endpoint protected by the standard auth middleware chain (auth, org_context, subscription) with `org:manage` permission that accepts the OAuth authorization `code` returned by the Meta Embedded Signup popup and provisions the organization's WhatsApp connection.

#### Scenario: Successful exchange provisions the connection

- **WHEN** an authenticated user with `org:manage` permission sends `POST /api/v1/whatsapp/signup/exchange` with a valid `{code}`
- **AND** the Meta token exchange, system-user provisioning, webhook registration, and test-echo validation all succeed
- **THEN** the system SHALL create or update the organization's `whatsapp_configs` row with the resolved `phone_number_id`, `business_phone`, `waba_id`, system-user `access_token`, and server-generated `webhook_secret`/`verify_token`
- **AND** the system SHALL mark the signup flow `connected` and the config `is_active`
- **AND** the system SHALL return HTTP 200 with the config summary (secrets masked) and `status: "connected"`

#### Scenario: Missing or malformed code

- **WHEN** the request body is missing `code` or contains an empty string
- **THEN** the system SHALL return HTTP 400 with a validation error

#### Scenario: Exchange already in progress

- **WHEN** a signup flow for the organization is already mid-flight (`exchanging`, `registering`, or `verifying`)
- **THEN** the system SHALL return HTTP 409 with error code `signup_in_progress`

#### Scenario: Organization already connected

- **WHEN** a signup flow for the organization is already `connected`
- **THEN** the system SHALL return HTTP 409 with error code `signup_already_connected`

#### Scenario: Meta API failure after retries

- **WHEN** a Meta Graph API step fails after the configured retry/backoff attempts
- **THEN** the system SHALL mark the flow `failed` with the Meta error code
- **AND** the system SHALL create a high-priority ticket in the tickets module with the error code and flow details
- **AND** the system SHALL return HTTP 502 with error code `signup_failed` and the recorded `error_code`

#### Scenario: Tickets module disabled

- **WHEN** a signup fails terminally and the tickets module is feature-disabled
- **THEN** the system SHALL log the failure and surface it via `GET /api/v1/whatsapp/signup/status` instead of creating a ticket

### Requirement: Signup status endpoint

The system SHALL expose a `GET /api/v1/whatsapp/signup/status` endpoint protected by the standard auth middleware chain (auth, org_context, subscription) with `org:manage` permission that returns the current signup flow state for the organization.

#### Scenario: Flow exists

- **WHEN** an authenticated user with `org:manage` permission requests `GET /api/v1/whatsapp/signup/status`
- **AND** a signup flow exists for the organization
- **THEN** the system SHALL return HTTP 200 with `status`, `step`, and, when failed, the `error_code`

#### Scenario: No flow exists

- **WHEN** the organization has no signup flow row
- **THEN** the system SHALL return HTTP 404 with error code `signup_not_found`

### Requirement: Server-generated webhook secrets

The system SHALL generate `webhook_secret` and `verify_token` values server-side (cryptographically random, at least 32 characters) during embedded-signup provisioning instead of requiring the user to supply them.

#### Scenario: Generated secrets stored and masked

- **WHEN** the exchange flow provisions a config
- **THEN** the stored `webhook_secret` and `verify_token` SHALL be server-generated values
- **AND** all API responses SHALL return them masked (first 6 + last 4 characters, `****` in between)

#### Scenario: Generated verify token used for subscription handshake

- **WHEN** the system registers app subscriptions with Meta
- **THEN** the generated `verify_token` SHALL be the value configured as the subscription `verify_token` on `/{app_id}/subscriptions`

### Requirement: Webhook subscription auto-registration

The system SHALL register the app's webhook subscription with Meta during the signup flow, targeting the organization's WABA and the platform webhook callback URL with the generated `verify_token`.

#### Scenario: Registration succeeds

- **WHEN** the exchange flow reaches the registration step
- **THEN** the system SHALL call `POST /{waba_id}/subscribed_apps` and `POST /{app_id}/subscriptions` with `object=whatsapp_business_account`, the platform callback URL, the generated `verify_token`, and fields including `messages` and `statuses`

#### Scenario: Registration fails transiently

- **WHEN** a registration call fails
- **THEN** the system SHALL retry with exponential backoff up to 3 attempts before marking the flow `failed`

### Requirement: Test-echo validation before connected

The system SHALL validate the provisioned connection by sending a test text message to the organization's own `business_phone` via the Cloud API before marking the flow `connected`.

#### Scenario: Test message succeeds

- **WHEN** the test message API call succeeds
- **THEN** the flow SHALL transition to `connected` and the config SHALL be `is_active`

#### Scenario: Test message fails

- **WHEN** the test message API call fails after retries
- **THEN** the flow SHALL transition to `failed` with the recorded error code and trigger the HITL ticket path

### Requirement: Token lifecycle constraints

The system SHALL only persist the permanent Meta system-user token and SHALL NOT persist transient OAuth authorization codes or user access tokens.

#### Scenario: Transient tokens not persisted

- **WHEN** the exchange flow processes an authorization `code` and user token
- **THEN** the system SHALL hold them in request memory only
- **AND** the only token written to the database SHALL be the permanent system-user token stored in the `whatsapp_configs.access_token` column
- **AND** the stored token SHALL be masked in all API responses

### Requirement: Coexistence outcome recorded

The system SHALL record whether the provisioned number is a coexistence number (an active WhatsApp Business app number on the client's phone) in the signup flow and config metadata.

#### Scenario: Coexistence number

- **WHEN** the number selected in the Embedded Signup popup is an active WhatsApp Business app number
- **THEN** the flow SHALL record `coexistence: true` in `metadata`
- **AND** the config metadata SHALL include `coexistence: true`

#### Scenario: API-only number

- **WHEN** the number is not an active WhatsApp Business app number
- **THEN** the flow SHALL record `coexistence: false` in `metadata`
- **AND** the system SHALL NOT require or perform server-side OTP handling in either case
