## Purpose

Defines the WhatsApp settings frontend: overview section, config detail form, and active and inactive toggle.
## Requirements
### Requirement: WhatsApp section appears in settings overview

The workspace settings overview at `/dashboard/settings` SHALL display a "Messaging" (WhatsApp) card and an "Instagram" card, each visible to users with `org:manage` permission. The Instagram card SHALL link to the Instagram settings view (`/dashboard/settings?view=instagram`).

#### Scenario: User has org:manage and config exists

- **WHEN** a user with `org:manage` permission views the settings overview
- **AND** the organization has an active WhatsApp config
- **THEN** the overview SHALL show a card with title "Messaging", the connected phone number as the value, and "Active" as the status

#### Scenario: User has org:manage and no WhatsApp config exists

- **WHEN** a user with `org:manage` permission views the settings overview
- **AND** the organization has no WhatsApp config
- **THEN** the overview SHALL show a card with title "Messaging", value "Not connected", and helper text "Connect WhatsApp to receive messages"

#### Scenario: Instagram card mirrors WhatsApp card

- **WHEN** a user with `org:manage` permission views the settings overview
- **THEN** an "Instagram" card SHALL be displayed with the connected IG username (or "Not connected") and status, per the instagram-messaging spec
- **AND** clicking it SHALL navigate to the Instagram settings view

#### Scenario: User lacks org:manage permission

- **WHEN** a user without `org:manage` permission views the settings overview
- **THEN** the overview SHALL NOT display the WhatsApp or Instagram cards

### Requirement: WhatsApp config detail view with form

The `whatsapp` settings view SHALL display a form for viewing and editing the organization's WhatsApp configuration.

#### Scenario: Config exists — form is pre-populated

- **WHEN** a user navigates to `/dashboard/settings?view=whatsapp`
- **AND** the organization has a WhatsApp config
- **THEN** the form SHALL display the current `phone_number_id`, `business_phone`, `app_id`, `waba_id`, `api_version`, `graph_api_url`, and `is_active` toggle
- **AND** the `webhook_secret`, `verify_token`, and `access_token` fields SHALL be masked as password inputs with a note "Leave blank to keep current value"
- **AND** a "Save" button SHALL be enabled when changes are made

#### Scenario: No config exists — empty form with onboarding guidance

- **WHEN** a user navigates to `/dashboard/settings?view=whatsapp`
- **AND** the organization has no WhatsApp config
- **THEN** the system SHALL display an empty state with guidance text explaining how to get the required values from the Meta WhatsApp Business dashboard
- **AND** the form fields SHALL be empty and editable

#### Scenario: Save config updates successfully

- **WHEN** a user fills in or edits config fields and clicks "Save"
- **THEN** the system SHALL send a `PUT /api/v1/whatsapp/config` request
- **AND** on success, SHALL show a success toast and refresh the displayed values (secrets and access_token remain masked)

#### Scenario: Save config fails with validation error

- **WHEN** a user submits the form with missing required fields
- **THEN** the system SHALL display an inline error message describing the validation failure

### Requirement: Active/inactive toggle on config detail view

The WhatsApp config detail view SHALL include a toggle control for the `is_active` status.

#### Scenario: Toggle deactivates the config

- **WHEN** a user toggles the active switch from on to off
- **THEN** the system SHALL send a `PATCH /api/v1/whatsapp/config/toggle` request
- **AND** on success, SHALL update the toggle state and show a confirmation toast

#### Scenario: Toggle activates the config

- **WHEN** a user toggles the active switch from off to on
- **THEN** the system SHALL send a `PATCH /api/v1/whatsapp/config/toggle` request
- **AND** on success, SHALL update the toggle state and show a confirmation toast

### Requirement: Settings view stack includes WhatsApp view

The settings page SHALL support `?view=whatsapp` in the URL query parameter, matching the existing pattern for profile, members, and subscription.

#### Scenario: Direct navigation to WhatsApp view

- **WHEN** a user navigates to `/dashboard/settings?view=whatsapp`
- **AND** the user has `org:manage` permission
- **THEN** the system SHALL display the WhatsApp config detail view with a "Back" button returning to the overview

#### Scenario: WhatsApp view redirects to overview if permission is missing

- **WHEN** a user navigates to `/dashboard/settings?view=whatsapp`
- **AND** the user lacks `org:manage` permission
- **THEN** the system SHALL redirect to the settings overview

### Requirement: Loading and error states for WhatsApp config

The WhatsApp config section SHALL handle loading, empty, and error states gracefully.

#### Scenario: Config is loading

- **WHEN** the config query is in a loading state
- **THEN** the system SHALL display a skeleton placeholder in the config detail view

#### Scenario: Config fetch fails

- **WHEN** the config query returns an error (not 404)
- **THEN** the system SHALL display an error message with a "Retry" button

### Requirement: WhatsApp config form includes outbound API fields

The WhatsApp config detail view SHALL include fields for WABA ID, permanent access token, API version, and Graph API URL in addition to the existing webhook fields.

#### Scenario: New outbound API fields are displayed

- **WHEN** a user navigates to `/dashboard/settings?view=whatsapp`
- **AND** the organization has a WhatsApp config with `waba_id` and `access_token` set
- **THEN** the form SHALL display WABA ID, a masked access token field (password input), API version, and Graph API URL
- **AND** the access token field SHALL show placeholder "Leave blank to keep current value" when a config exists

#### Scenario: Saving includes new fields

- **WHEN** a user fills in `waba_id`, `access_token`, `api_version`, and `graph_api_url` and clicks "Save"
- **THEN** the system SHALL include these fields in the `PUT /api/v1/whatsapp/config` request body

### Requirement: Webhook callback URL display

The WhatsApp config detail view SHALL display the webhook callback URL that must be configured in Meta's WhatsApp Business dashboard.

#### Scenario: Callback URL is shown

- **WHEN** a user navigates to `/dashboard/settings?view=whatsapp`
- **THEN** the system SHALL display a read-only field or info box showing the callback URL in the format `https://<domain>/api/v1/webhooks/whatsapp`
- **AND** a "Copy" button SHALL be available next to the URL

#### Scenario: Callback URL adapts to the deployment domain

- **WHEN** the application is deployed at different domains
- **THEN** the callback URL SHALL use the current `window.location.origin` as the base

### Requirement: Webhook health indicator

The config detail view SHALL display a webhook health indicator showing recent webhook log activity.

#### Scenario: Recent successful webhooks

- **WHEN** the organization has received webhooks in the last 24 hours
- **THEN** the system SHALL display a green indicator with "Webhooks active — last received {relative time}"

#### Scenario: No recent webhooks but config is active

- **WHEN** the config is active but no webhooks have been received in the last 24 hours
- **THEN** the system SHALL display a yellow indicator with "No webhooks received in the last 24 hours"

#### Scenario: Config not connected

- **WHEN** the organization has no WhatsApp config
- **THEN** the system SHALL NOT display a webhook health indicator

### Requirement: Setup guidance for Meta Business Dashboard

The config view SHALL include a setup guide or info box that explains how to obtain the required values from Meta's WhatsApp Business Dashboard.

#### Scenario: New config state shows setup guide

- **WHEN** a user views the WhatsApp config page and the organization has no config
- **THEN** the system SHALL display a step-by-step guide explaining:
  - How to create a Meta Business App
  - Where to find the Phone Number ID and WABA ID
  - How to generate a permanent access token
  - Where to configure the webhook callback URL and verify token

#### Scenario: Connected state shows compact info

- **WHEN** a user views the WhatsApp config page and the organization has an active config
- **THEN** the setup guide SHALL be collapsed or shown as a collapsible section

### Requirement: WhatsApp settings view supports embedded signup entry

When the organization has no WhatsApp config, the `whatsapp` settings view SHALL surface the embedded signup connect flow (Meta SDK login → code exchange → status polling) as the primary entry point, in addition to the existing manual configuration form. Once a connection succeeds, the view SHALL render the post-connect next-steps flow instead of terminating at the connected banner alone.

#### Scenario: No config exists — connect flow offered first

- **WHEN** a user with `org:manage` permission opens `/dashboard/settings?view=whatsapp`
- **AND** the organization has no WhatsApp config
- **THEN** the view SHALL display a "Connect WhatsApp" action that launches the Meta SDK embedded signup flow
- **AND** the manual config form SHALL remain available (e.g., under "Advanced settings")

#### Scenario: Embedded signup in progress

- **WHEN** the embedded signup flow is exchanging, registering, or verifying
- **THEN** the view SHALL display progress feedback and poll the signup status until it reaches `connected` or `failed`

#### Scenario: Embedded signup succeeds

- **WHEN** the signup status reaches `connected`
- **THEN** the view SHALL refetch and display the new WhatsApp config and show a success toast

#### Scenario: Embedded signup fails

- **WHEN** the signup status reaches `failed` or the exchange returns an error
- **THEN** the view SHALL display the error with a "Try again" action that restarts the flow

#### Scenario: Post-connect flow rendered after success

- **WHEN** the embedded signup exchange succeeds and the configuration becomes active
- **THEN** the WhatsApp settings view SHALL render the post-connect next-steps card alongside the connected state

#### Scenario: Connect entry preserved for inactive state

- **WHEN** no active configuration exists
- **THEN** the existing connect empty-state entry SHALL remain unchanged

