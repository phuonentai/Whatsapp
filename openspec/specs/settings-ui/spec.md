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
The system SHALL render the whatsapp-config section with workspace WhatsApp configuration controls.

#### Scenario: WhatsApp config section renders
- **WHEN** user opens the settings whatsapp section
- **THEN** the WhatsApp configuration controls are visible

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
