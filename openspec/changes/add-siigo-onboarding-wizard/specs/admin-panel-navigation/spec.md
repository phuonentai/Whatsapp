## ADDED Requirements

### Requirement: Admin onboarding overview with assisted provisioning

The system SHALL render an onboarding overview in the admin surface listing organizations with their invoicing connection state, numeración snapshot (prefijo, next number, confirmed date), last import run (counts, timestamp), and last connection error. The view SHALL be restricted to members with the admin role. For organizations in `awaiting_setup`, the view SHALL render an assisted-provisioning form (client id, client secret) that submits through the admin-provisioning endpoint.

#### Scenario: Admin sees onboarding overview

- **WHEN** an admin opens the onboarding overview
- **THEN** the system SHALL display each organization's connection state, numeración snapshot, last import run, and last error

#### Scenario: Admin provisions credentials for an awaiting organization

- **WHEN** an admin submits the assisted-provisioning form for an organization in `awaiting_setup`
- **THEN** the system SHALL call the admin-provisioning endpoint
- **AND** SHALL refresh the row to the new state on success
- **AND** SHALL display the server error verbatim on failure

#### Scenario: Non-admin denied onboarding overview

- **WHEN** a member without the admin role requests the onboarding overview
- **THEN** the system SHALL deny access and SHALL NOT render the view
