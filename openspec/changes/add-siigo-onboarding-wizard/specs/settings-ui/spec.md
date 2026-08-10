## ADDED Requirements

### Requirement: Settings shows Siigo integration section with status banner

The system SHALL render a "Integración Siigo" section in the settings dashboard reflecting the organization's connection state from `GET /api/v1/org/siigo/status`. The section SHALL show a status banner for every state: `none` (invitation to connect), `awaiting_setup` (assisted pending message), `connected`/`numeracion_ok`/`sandbox_ok` (wizard progress), `paused` (paused notice), `invoicing_disabled` (single-line "Facturación desactivada — activa con Siigo"), and `live` (active confirmation). The banner SHALL never be empty or silently absent.

#### Scenario: Non-connected organization sees connect invitation

- **WHEN** an organization's connection state is `none` and a member opens the settings dashboard
- **THEN** the system SHALL render the Siigo section with a banner inviting the member to connect Siigo

#### Scenario: Assisted setup shows pending message

- **WHEN** an organization's connection state is `awaiting_setup`
- **THEN** the system SHALL render the banner "Tu equipo está configurando tu facturación"
- **AND** SHALL NOT render the self-serve connect form

#### Scenario: Manual-path organization sees explicit disabled state

- **WHEN** an organization's connection state is `invoicing_disabled`
- **THEN** the system SHALL render the single-line notice "Facturación desactivada — activa con Siigo"

### Requirement: Siigo onboarding wizard with server-persisted progress

The system SHALL render a 5-step wizard in the Siigo settings section: (1) Conectar Siigo, (2) Numeración, (3) Importar clientes, (4) Prueba en sandbox, (5) Activar. Each step SHALL be enabled only when the backend connection state permits it, SHALL submit through the change-1/change-2 endpoints, SHALL display server errors verbatim, and SHALL be resumable (progress is read from backend state, not local storage).

#### Scenario: Wizard steps follow connection state

- **WHEN** an organization is in state `connected`
- **THEN** step 1 is complete, step 2 (Numeración) is enabled, and steps 3–5 are locked

#### Scenario: Numeración confirmation advances the wizard

- **WHEN** a member confirms the numeración presented in step 2
- **THEN** the system SHALL submit the confirmation endpoint
- **AND** SHALL advance the wizard to step 3 when the backend state is `numeracion_ok`

#### Scenario: Wizard shows server error verbatim

- **WHEN** a step submission fails (e.g., NIT mismatch, sandbox test failure)
- **THEN** the system SHALL display the server error message inline at the failed step
- **AND** SHALL NOT advance the wizard

### Requirement: Import preview and confirmation UI

The system SHALL render the customer-import step with a preview (counts: nuevos, existentes, duplicados por NIT, sin NIT) fetched from the import-preview endpoint, SHALL require explicit user confirmation before the confirm endpoint is called, and SHALL display the recorded import run result (counts + timestamp) after confirmation.

#### Scenario: Preview shown before any write

- **WHEN** a member opens the import step and the backend returns preview counts
- **THEN** the system SHALL display the counts
- **AND** SHALL NOT call the confirm endpoint until the member confirms

#### Scenario: Confirmed import shows result

- **WHEN** a member confirms the import
- **THEN** the system SHALL display the completed run's counts and timestamp from the backend

### Requirement: Sandbox test step and go-live activation

The system SHALL render the sandbox test step: a button invoking the test-invoice endpoint, a status indicator while awaiting validity, and a success state when the connection advances to `sandbox_ok`. The final activation step SHALL become available when the backend state is `sandbox_ok` and SHALL display the `factura_lista` WhatsApp template approval status (approved / pending) sourced from existing WhatsApp config data.

#### Scenario: Sandbox test success unlocks activation

- **WHEN** the sandbox test invoice reaches a valid status
- **THEN** the system SHALL mark the test step complete
- **AND** SHALL enable the activation step

#### Scenario: Activation step shows template status

- **WHEN** the activation step is displayed
- **THEN** the system SHALL show whether the `factura_lista` WhatsApp template is approved or pending approval

### Requirement: Invoicing kill-switch toggle in settings

The system SHALL render a pause/resume toggle in the Siigo section when the organization is `live` or `paused`, invoking the pause/resume endpoints, and SHALL reflect the backend state immediately after the call.

#### Scenario: Live organization pauses invoicing

- **WHEN** a member toggles pause while the connection is `live`
- **THEN** the system SHALL call the pause endpoint
- **AND** SHALL show the paused banner

#### Scenario: Paused organization resumes

- **WHEN** a member toggles resume while the connection is `paused`
- **THEN** the system SHALL call the resume endpoint
- **AND** SHALL show the live banner
