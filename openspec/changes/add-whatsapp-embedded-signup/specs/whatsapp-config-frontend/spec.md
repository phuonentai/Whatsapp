## MODIFIED Requirements

### Requirement: WhatsApp config detail view with form

The `whatsapp` settings view SHALL display the organization's WhatsApp connection: a primary embedded-signup connect flow when no config exists, and a form for viewing and editing the configuration once connected.

#### Scenario: Config exists — form is pre-populated

- **WHEN** a user navigates to `/dashboard/settings?view=whatsapp`
- **AND** the organization has a WhatsApp config
- **THEN** the view SHALL display the current `phone_number_id`, `business_phone`, `app_id`, and `is_active` toggle
- **AND** the `webhook_secret` and `verify_token` fields SHALL be masked as password inputs with a note "Leave blank to keep current value"
- **AND** a "Save" button SHALL be enabled when changes are made
- **AND** the form SHALL be presented under an "Advanced" disclosure

#### Scenario: No config exists — embedded connect flow

- **WHEN** a user navigates to `/dashboard/settings?view=whatsapp`
- **AND** the organization has no WhatsApp config
- **THEN** the system SHALL display a "Connect WhatsApp" button as the primary action
- **AND** SHALL NOT auto-open the manual form
- **AND** the manual form SHALL remain available under an "Advanced" disclosure

#### Scenario: Save config updates successfully

- **WHEN** a user fills in or edits config fields and clicks "Save"
- **THEN** the system SHALL send a `PUT /api/v1/whatsapp/config` request
- **AND** on success, SHALL show a success toast and refresh the displayed values (secrets remain masked)

#### Scenario: Save config fails with validation error

- **WHEN** a user submits the form with missing required fields
- **THEN** the system SHALL display an inline error message describing the validation failure

## ADDED Requirements

### Requirement: Embedded signup connect flow

The `whatsapp` settings view SHALL support connecting WhatsApp through Meta Embedded Signup: a "Connect WhatsApp" button that opens the Meta popup, submits the returned authorization code to the backend, and reports the outcome.

#### Scenario: Successful connect

- **WHEN** a user clicks "Connect WhatsApp" and completes the Meta popup flow
- **THEN** the system SHALL submit the returned authorization code to `POST /api/v1/whatsapp/signup/exchange`
- **AND** on success, SHALL show a success toast and display the connected config summary (secrets masked)

#### Scenario: Popup cancelled or failed

- **WHEN** the user cancels the Meta popup or the SDK returns an error
- **THEN** the system SHALL show the error state with the Meta error message and allow retrying the connect flow

#### Scenario: Backend exchange fails

- **WHEN** `POST /api/v1/whatsapp/signup/exchange` returns an error (including `signup_failed`, `signup_in_progress`, `signup_already_connected`)
- **THEN** the system SHALL display the returned error message and, for `signup_failed`, a support CTA referencing the signup status

### Requirement: In-flight micro-status

The system SHALL display a live micro-status during the embedded-signup provisioning so the user sees progress while the backend exchanges tokens, registers webhooks, and validates the connection.

#### Scenario: Micro-status states render during exchange

- **WHEN** the exchange request is in flight
- **THEN** the view SHALL render the sequence: "Connecting your WhatsApp..." → "Verifying Coexistence session..." → "Establishing secure token & webhooks..." → "All set! Your WhatsApp is live." on completion

#### Scenario: Status polling after reload

- **WHEN** a user reloads the settings page while a signup flow is in progress
- **THEN** the system SHALL query `GET /api/v1/whatsapp/signup/status` and render the current state and step
