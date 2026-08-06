## ADDED Requirements

### Requirement: WhatsApp config supports WABA ID and access token for outbound API

The `whatsapp.whatsapp_configs` table SHALL support `waba_id` (VARCHAR, nullable), `access_token` (VARCHAR, nullable), `api_version` (VARCHAR, default `v21.0`), and `graph_api_url` (VARCHAR, default `https://graph.facebook.com`) columns for outbound WhatsApp Cloud API communication.

#### Scenario: Config with new fields is returned

- **WHEN** a config exists with `waba_id`, `access_token`, `api_version`, and `graph_api_url` set
- **AND** `GET /api/v1/whatsapp/config` is called
- **THEN** the response SHALL include all four fields, with `access_token` masked (first 6 + last 4 chars, `****` in between)

#### Scenario: Config with default values for optional fields

- **WHEN** a config is created without specifying `api_version` and `graph_api_url`
- **THEN** the system SHALL default `api_version` to `v21.0` and `graph_api_url` to `https://graph.facebook.com`

#### Scenario: Update config with new fields

- **WHEN** `PUT /api/v1/whatsapp/config` is called with new values for `waba_id`, `access_token`, `api_version`, or `graph_api_url`
- **THEN** the system SHALL update the corresponding fields

## MODIFIED Requirements

### Requirement: Auth-gated GET endpoint returns organization's WhatsApp config

The system SHALL expose a `GET /api/v1/whatsapp/config` endpoint protected by the standard auth middleware chain (auth, org_context, subscription) that returns the authenticated organization's WhatsApp configuration.

#### Scenario: Config exists

- **WHEN** an authenticated user with `org:manage` permission requests `GET /api/v1/whatsapp/config`
- **AND** the organization has a WhatsApp config record
- **THEN** the system SHALL return HTTP 200 with the config object, masking `webhook_secret`, `verify_token`, and `access_token` to show only the first 6 and last 4 characters (e.g., `"whsec_****abcd"`)

#### Scenario: No config exists

- **WHEN** an authenticated user with `org:manage` permission requests `GET /api/v1/whatsapp/config`
- **AND** the organization has no WhatsApp config record
- **THEN** the system SHALL return HTTP 404 with error code `config_not_found`

#### Scenario: Unauthenticated request

- **WHEN** a request arrives without a valid auth token
- **THEN** the system SHALL return HTTP 401

#### Scenario: User lacks org:manage permission

- **WHEN** an authenticated user without `org:manage` permission requests `GET /api/v1/whatsapp/config`
- **THEN** the system SHALL return HTTP 403

### Requirement: Auth-gated PUT endpoint upserts WhatsApp config

The system SHALL expose a `PUT /api/v1/whatsapp/config` endpoint that creates a new config or updates the existing one (upsert) for the authenticated organization.

#### Scenario: Create a new config

- **WHEN** an authenticated user with `org:manage` permission sends `PUT /api/v1/whatsapp/config` with a valid body containing `phone_number_id`, `business_phone`, `webhook_secret`, and `verify_token`
- **AND** the organization has no existing WhatsApp config
- **THEN** the system SHALL insert a new row into `whatsapp.whatsapp_configs` scoped to the organization, including optional fields `waba_id`, `access_token`, `api_version`, and `graph_api_url` if provided
- **AND** the system SHALL return HTTP 200 with the created config (secrets and access_token masked)

#### Scenario: Update an existing config

- **WHEN** an authenticated user with `org:manage` permission sends `PUT /api/v1/whatsapp/config` with updated field values
- **AND** the organization already has a WhatsApp config
- **THEN** the system SHALL update the existing row using partial update semantics (only changed fields are updated; omitted fields retain their current values)
- **AND** the system SHALL return HTTP 200 with the updated config (secrets and access_token masked)

#### Scenario: Secret field is empty on update

- **WHEN** a `PUT /api/v1/whatsapp/config` request body has an empty `webhook_secret`, `verify_token`, or `access_token`
- **THEN** the system SHALL preserve the existing stored secret value (do not overwrite with empty string)

#### Scenario: Secret field contains masked value on update

- **WHEN** a `PUT /api/v1/whatsapp/config` request body has a masked `webhook_secret` (e.g., `"whsec_****abcd"`), `verify_token`, or `access_token`
- **THEN** the system SHALL preserve the existing stored secret value

#### Scenario: Validation fails

- **WHEN** a `PUT /api/v1/whatsapp/config` request body is missing required fields (`phone_number_id`, `business_phone`, `webhook_secret`, `verify_token`) for a new config
- **THEN** the system SHALL return HTTP 400 with a validation error message

#### Scenario: Duplicate phone_number_id

- **WHEN** a `PUT /api/v1/whatsapp/config` request contains a `phone_number_id` that already belongs to a different organization
- **THEN** the system SHALL return HTTP 409 with error code `phone_number_id_conflict`

### Requirement: Auth-gated PATCH endpoint toggles config active state

The system SHALL expose a `PATCH /api/v1/whatsapp/config/toggle` endpoint that toggles the `is_active` field of the organization's WhatsApp configuration.

#### Scenario: Toggle config off

- **WHEN** an authenticated user with `org:manage` permission sends `PATCH /api/v1/whatsapp/config/toggle`
- **AND** the config is currently active
- **THEN** the system SHALL set `is_active` to `false`
- **AND** return HTTP 200 with the updated config

#### Scenario: Toggle config on

- **WHEN** an authenticated user with `org:manage` permission sends `PATCH /api/v1/whatsapp/config/toggle`
- **AND** the config is currently inactive
- **THEN** the system SHALL set `is_active` to `true`
- **AND** return HTTP 200 with the updated config

#### Scenario: No config to toggle

- **WHEN** a user sends `PATCH /api/v1/whatsapp/config/toggle` but the organization has no config
- **THEN** the system SHALL return HTTP 404 with error code `config_not_found`
