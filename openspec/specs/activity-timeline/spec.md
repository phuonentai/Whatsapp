## Purpose

Defines the activity entity with organization scoping, typed activity categories, performed_by attribution, and tenant-safe storage for CRM history tracking.

## Requirements

### Requirement: Activity entity with organization scoping

The system SHALL store activities in `crm.activities` scoped by `organization_id`, with type, content, and optional associations to contact, company, deal, and conversation.

#### Scenario: Activity created with minimum fields

- **WHEN** an activity is created with type='note', content, and organization_id
- **THEN** the activity SHALL be persisted
- **AND** optional entity references SHALL be NULL

#### Scenario: Activity linked to multiple entities

- **WHEN** an activity is created with contact_id, deal_id, and company_id
- **THEN** all associations SHALL be persisted
- **AND** the activity SHALL appear in the timeline for each linked entity

### Requirement: Activity types

The system SHALL support activity types: `note`, `call`, `email`, `meeting`, `task`, `whatsapp_message`, and `system`, enforced by a CHECK constraint.

#### Scenario: Note activity

- **WHEN** an activity of type 'note' is created with content
- **THEN** the activity SHALL be persisted with the note content

#### Scenario: Task activity with due date

- **WHEN** an activity of type 'task' is created with due_date and status='pending'
- **THEN** the due_date and status SHALL be persisted

#### Scenario: Invalid activity type rejected

- **WHEN** an activity is created with an invalid type value
- **THEN** the system SHALL return a validation error

### Requirement: Activity has a performed_by field

The system SHALL store `performed_by` referencing `organizations.accounts(id)` and `performed_at` (timestamp) for each activity.

#### Scenario: Activity created by a team member

- **WHEN** an activity is created with `performed_by` set to a valid account ID
- **THEN** the activity SHALL record who performed it

#### Scenario: System activity has no performed_by

- **WHEN** a system-generated activity is created (e.g., deal stage change)
- **THEN** `performed_by` SHALL be NULL
- **AND** `performed_at` SHALL be set to the current timestamp

### Requirement: Activity timeline for a contact

The system SHALL provide `GET /api/crm/actividades/contacto/:id` returning all activities associated with the contact, ordered by `performed_at DESC`.

#### Scenario: Contact timeline with mixed activity types

- **WHEN** a GET request is made to `/api/crm/actividades/contacto/1`
- **THEN** the system SHALL return all activities where `contact_id = 1`
- **AND** activities SHALL be ordered by performed_at descending
- **AND** the response SHALL include pagination

### Requirement: Activity timeline for a deal

The system SHALL provide `GET /api/crm/actividades/negocio/:id` returning all activities associated with the deal, ordered by `performed_at DESC`.

#### Scenario: Deal timeline includes stage changes

- **WHEN** a GET request is made to `/api/crm/actividades/negocio/1`
- **THEN** the response SHALL include system-generated activities for stage transitions
- **AND** SHALL include manually created notes, calls, etc.

### Requirement: Global activity list

The system SHALL provide `GET /api/crm/actividades` returning all activities for the organization, with optional filters for `tipo`, `entity_type`, and `entity_id`.

#### Scenario: Filter activities by type

- **WHEN** a GET request is made to `/api/crm/actividades?tipo=call`
- **THEN** the system SHALL return only activities with `type = 'call'`

#### Scenario: Filter activities by contact

- **WHEN** a GET request is made to `/api/crm/actividades?entity_type=contact&entity_id=1`
- **THEN** the system SHALL return only activities linked to contact ID 1

### Requirement: Activity creation is RBAC-protected

The system SHALL require `contact:manage` permission to create activities manually.

#### Scenario: User creates a note activity

- **WHEN** a user with `contact:manage` permission creates a note
- **THEN** the activity SHALL be persisted

#### Scenario: User without permission cannot create

- **WHEN** a user without `contact:manage` permission attempts to create an activity
- **THEN** the system SHALL return HTTP 403 Forbidden

### Requirement: Deal stage transitions create system activities

The system SHALL create an activity of type `system` (stored as `sistema`) when a deal moves between stages, associated with the deal, with `performed_by` NULL and `performed_at` set to the current timestamp.

#### Scenario: Moving a deal logs a system activity

- **WHEN** a deal is moved from one etapa to another via `PUT /api/crm/negocios/:id/etapa`
- **THEN** a system activity SHALL be created and associated with the deal
- **AND** the activity SHALL appear in the deal's activity timeline

### Requirement: Task activity fields with due date and estado

The system SHALL persist `fecha_vencimiento` (due date) and `estado` for activities of type `task` created via `POST /api/crm/actividades`.

#### Scenario: Task activity created with due date and estado

- **WHEN** an activity of type `task` is created with `fecha_vencimiento` and `estado='pendiente'`
- **THEN** both fields SHALL be persisted with the activity
- **AND** the response SHALL include them
