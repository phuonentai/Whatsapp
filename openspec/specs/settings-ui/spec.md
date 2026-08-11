# Spec: settings-ui

## Purpose

Defines the settings UI: member invitation with roles, module toggling, and playbook guion management.

## Requirements

### Requirement: Admin invites a member with a role
The system SHALL render an invite-member form in the settings dashboard, SHALL accept a member email and role selection, and SHALL submit the invite through the member API for workspace admins.

#### Scenario: Invite member form renders with role options
- **WHEN** an admin opens the settings member section
- **THEN** an invite form with email input and role selector is visible

#### Scenario: Submitting an invite creates an invitation
- **WHEN** an admin enters a member email, selects a role, and submits
- **THEN** the invite is submitted to the member API

### Requirement: Admin toggles a module
The system SHALL render a switch for each module in the modules section, reflecting the module's enabled state, and SHALL persist a toggle change to the module settings API.

#### Scenario: Module switch reflects and persists enabled state
- **WHEN** an admin toggles a module switch
- **THEN** the switch state updates and the change is persisted

### Requirement: Settings shows playbook guiones
The system SHALL render the playbooks section listing playbooks and their guiones for the workspace.

#### Scenario: Playbook guiones render in settings
- **WHEN** an applied playbook with guiones exists and user opens the playbooks section
- **THEN** the playbook and its guiones are visible

### Requirement: User edits profile
The system SHALL render the profile section with editable fields and SHALL persist profile updates.

#### Scenario: Profile update persists
- **WHEN** user edits a profile field and saves
- **THEN** the updated value is shown and persisted

### Requirement: Settings shows subscription plan
The system SHALL render the subscription tab reflecting the workspace's plan.

#### Scenario: Subscription tab shows plan
- **WHEN** user opens the subscription tab
- **THEN** the current plan is displayed

### Requirement: Settings shows WhatsApp configuration
The system SHALL render the whatsapp-config section with workspace WhatsApp configuration controls, using the typed copy layer in Spanish-first voice. Primary connect copy SHALL use plain language; Meta developer tokens SHALL be confined to the collapsed advanced panel.

#### Scenario: WhatsApp config section renders
- **WHEN** user opens the settings whatsapp section
- **THEN** the WhatsApp configuration controls are visible

#### Scenario: WhatsApp config section renders Spanish copy

- **WHEN** a user opens the settings WhatsApp section
- **THEN** the section title, description, connect empty-state, status labels, and connect button SHALL be Spanish strings resolved from the copy layer

#### Scenario: Connected state renders Spanish

- **WHEN** a WhatsApp configuration is active
- **THEN** the connected banner and message-receiving status SHALL render the Spanish strings from the copy layer

### Requirement: Admin sees member list with role controls
The system SHALL render the member list showing each member's role and SHALL expose role controls for members the admin can manage.

#### Scenario: Member list shows roles
- **WHEN** an admin opens the member list
- **THEN** each member's role is displayed and manageable members expose role controls

### Requirement: Non-admin members see restricted settings
The system SHALL hide the invite form and member role controls from members without ORG_MANAGE, SHALL disable module toggles for them, and SHALL reject privileged settings API calls with 403.

#### Scenario: Member sees no invite form
- **WHEN** a member identity opens the settings member section
- **THEN** the invite form is not visible

#### Scenario: Member module toggles are disabled
- **WHEN** a member identity opens the modules section
- **THEN** the module switches render disabled and cannot be toggled

#### Scenario: Member sees no role controls
- **WHEN** a member identity opens the member list
- **THEN** member role controls are not visible

#### Scenario: Member invite call is rejected
- **WHEN** a member identity calls the member invite API
- **THEN** the API responds 403 and no invitation is created

### Requirement: Settings handles empty and failure states
The system SHALL render an empty state when the member list has no members, and SHALL surface a duplicate-invite rejection without creating a duplicate invitation.

#### Scenario: Empty member list renders empty state
- **WHEN** a workspace has no members and an admin opens the member list
- **THEN** an empty-state message is shown

#### Scenario: Duplicate invite is rejected
- **WHEN** an admin submits an invite for an email that is already a member or has a pending invite
- **THEN** the system surfaces a rejection and does not create a duplicate invitation

#### Scenario: Double-submit creates a single invitation
- **WHEN** an admin submits an invite twice rapidly
- **THEN** only one invitation is created

### Requirement: Destructive member and compliance actions use custom dialogs

Member role changes, member removal, and the compliance forget flow SHALL confirm via the custom `ConfirmDialog` component and SHALL NOT use native `window.confirm`.

#### Scenario: Member role change confirms via custom dialog

- **WHEN** an admin changes a member's role
- **THEN** confirmation SHALL render in the custom `ConfirmDialog`
- **AND** no `window.confirm` SHALL be invoked

#### Scenario: Member removal confirms via custom dialog

- **WHEN** an admin removes a member
- **THEN** confirmation SHALL render in the custom `ConfirmDialog`
- **AND** the "last admin" guard SHALL show an inline error, not a native dialog

#### Scenario: Compliance forget confirms via custom dialog

- **WHEN** a user triggers the compliance forget flow
- **THEN** confirmation SHALL render in the custom `ConfirmDialog`
- **AND** no `window.confirm` SHALL be invoked

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
