## MODIFIED Requirements

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

## ADDED Requirements

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
