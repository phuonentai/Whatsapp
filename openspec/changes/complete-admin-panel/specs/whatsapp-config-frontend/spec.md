## ADDED Requirements

### Requirement: WhatsApp settings view supports embedded signup entry

When the organization has no WhatsApp config, the `whatsapp` settings view SHALL surface the embedded signup connect flow (Meta SDK login → code exchange → status polling) as the primary entry point, in addition to the existing manual configuration form.

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
