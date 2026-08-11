## ADDED Requirements

### Requirement: Siigo onboarding wizard e2e happy path

The system SHALL provide a Playwright e2e scenario driving the full onboarding wizard against the offline stack (mock auth + mock Siigo provider): connect with mock credentials → confirm numeration → preview + confirm customer import → create sandbox test invoice → activate invoicing. The scenario SHALL use web-first assertions (no fixed sleeps) and SHALL assert the UI reflects each state transition driven by backend responses.

#### Scenario: Wizard completes from none to live

- **WHEN** a member of the Siigo e2e organization opens the Siigo settings section with no connection
- **THEN** the connect form SHALL be visible
- **AND** after submitting mock credentials, the numeration step SHALL become visible
- **AND** after confirming numeration, the import preview SHALL render counts
- **AND** after confirming the import, the sandbox test step SHALL be visible
- **AND** after the test invoice reaches a valid status, activation SHALL succeed and the section SHALL show the active banner

### Requirement: Siigo kill-switch e2e

The system SHALL provide an e2e scenario toggling pause and resume from the Siigo settings section for a live organization, asserting the paused notice and the active banner respectively.

#### Scenario: Pause and resume from settings

- **WHEN** an organization is `live` and a member clicks Pausar
- **THEN** the section SHALL show the paused notice
- **AND** clicking Reanudar SHALL restore the active banner

### Requirement: Siigo assisted setup e2e

The system SHALL provide an e2e scenario covering the assisted path: a client organization requests assisted setup (awaiting_setup), an admin provisions credentials through the admin view, and the client's section reflects the provisioned connection.

#### Scenario: Request assisted then admin provisions

- **WHEN** a member requests assisted setup
- **THEN** the section SHALL show "Tu equipo está configurando tu facturación"
- **AND** after an admin provisions credentials via the admin view, the section SHALL advance past the awaiting state to the connected state

### Requirement: Siigo admin onboarding view e2e

The system SHALL provide an e2e scenario opening the admin onboarding view, asserting the per-organization table (status, NIT, numeration, last import run) and the assisted-provisioning form for awaiting organizations.

#### Scenario: Admin sees and provisions rows

- **WHEN** an admin opens the Onboarding Siigo view
- **THEN** the organization table SHALL render connection rows
- **AND** awaiting_setup rows SHALL expose the provisioning form

### Requirement: Deal-stage invoicing gating e2e

The system SHALL provide an e2e scenario creating a deal, moving it to the `facturado` stage before onboarding is live (no invoice, activity recorded), then completing onboarding and moving another deal to `facturado` (invoice created and status resolved via the mock provider webhook).

#### Scenario: No invoice before live, invoice after live

- **WHEN** a deal reaches `facturado` while the organization is not `live`
- **THEN** no invoice SHALL be created and a "Facturación no activa" activity SHALL be recorded
- **WHEN** the organization is `live` and a second deal reaches `facturado`
- **THEN** an invoice SHALL be created and SHALL reach a valid status through the mock provider

### Requirement: Cross-organization isolation e2e

The system SHALL provide an e2e scenario asserting that onboarding one organization does not change the connection state of another organization.

#### Scenario: One org's onboarding leaves another untouched

- **WHEN** the Siigo e2e organization completes onboarding to `live`
- **THEN** a second seeded organization SHALL still report `none` and SHALL show the connect invitation
