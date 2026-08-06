## ADDED Requirements

### Requirement: WhatsApp section appears in settings overview

The workspace settings overview at `/dashboard/settings` SHALL display a "Messaging" (WhatsApp) card visible to users with `org:manage` permission.

#### Scenario: User has org:manage and config exists

- **WHEN** a user with `org:manage` permission views the settings overview
- **AND** the organization has an active WhatsApp config
- **THEN** the overview SHALL show a card with title "Messaging", the connected phone number as the value, and "Active" as the status

#### Scenario: User has org:manage and no config exists

- **WHEN** a user with `org:manage` permission views the settings overview
- **AND** the organization has no WhatsApp config
- **THEN** the overview SHALL show a card with title "Messaging", value "Not connected", and helper text "Connect WhatsApp to receive messages"

#### Scenario: User lacks org:manage permission

- **WHEN** a user without `org:manage` permission views the settings overview
- **THEN** the overview SHALL NOT display the WhatsApp card

### Requirement: WhatsApp config detail view with form

The `whatsapp` settings view SHALL display a form for viewing and editing the organization's WhatsApp configuration.

#### Scenario: Config exists — form is pre-populated

- **WHEN** a user navigates to `/dashboard/settings?view=whatsapp`
- **AND** the organization has a WhatsApp config
- **THEN** the form SHALL display the current `phone_number_id`, `business_phone`, `app_id`, and `is_active` toggle
- **AND** the `webhook_secret` and `verify_token` fields SHALL be masked as password inputs with a note "Leave blank to keep current value"
- **AND** a "Save" button SHALL be enabled when changes are made

#### Scenario: No config exists — empty form with onboarding guidance

- **WHEN** a user navigates to `/dashboard/settings?view=whatsapp`
- **AND** the organization has no WhatsApp config
- **THEN** the system SHALL display an empty state with guidance text explaining how to get the required values from the Meta WhatsApp Business dashboard
- **AND** the form fields SHALL be empty and editable

#### Scenario: Save config updates successfully

- **WHEN** a user fills in or edits config fields and clicks "Save"
- **THEN** the system SHALL send a `PUT /api/v1/whatsapp/config` request
- **AND** on success, SHALL show a success toast and refresh the displayed values (secrets remain masked)

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
