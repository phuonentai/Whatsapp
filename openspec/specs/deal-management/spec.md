## Purpose

Defines the deal entity with organization scoping, a Spanish status lifecycle, and financial fields in Colombian pesos.

## Requirements

### Requirement: Deal entity with organization scoping

The system SHALL store deals (negocios) in `crm.deals` scoped by `organization_id`, with required associations to a pipeline and optional associations to a contact and company. Default currency SHALL be COP (pesos colombianos).

#### Scenario: Negocio created with minimum required fields

- **WHEN** a negocio is created with nombre, pipeline_id, and organization_id
- **THEN** the negocio SHALL be persisted with `estado = 'abierto'` and `moneda = 'COP'`

#### Scenario: Negocio created with full associations

- **WHEN** a negocio is created with contact_id, company_id, pipeline_id, stage_id, monto=5000000, moneda='COP', and fecha_cierre_esperada
- **THEN** all fields SHALL be persisted

### Requirement: Deal has a status lifecycle in Spanish

The system SHALL support negocio status values of `abierto`, `ganado`, `perdido`, and `abandonado`, enforced by a CHECK constraint. API responses SHALL use Spanish status names.

#### Scenario: Negocio starts as abierto

- **WHEN** a negocio is created
- **THEN** its estado SHALL default to 'abierto'

#### Scenario: Negocio can be marked ganado

- **WHEN** a negocio is updated with `estado = 'ganado'`
- **THEN** the negocio estado SHALL be changed to 'ganado'
- **AND** the system SHALL create an Activity of type 'sistema' documenting "Negocio ganado"

### Requirement: Deal has financial fields in Colombian pesos

The system SHALL store `monto` (DECIMAL 12,2), `moneda` (default 'COP'), and `probabilidad` (integer percentage) for each negocio.

#### Scenario: Negocio with COP amount

- **WHEN** a negocio is created with monto=10000000 and moneda='COP'
- **THEN** both values SHALL be persisted representing 10.000.000 pesos colombianos

#### Scenario: Negocio uses default COP currency

- **WHEN** a negocio is created without specifying moneda
- **THEN** the system SHALL default to 'COP'

### Requirement: Deal stage transition via dedicated endpoint

The system SHALL provide `PUT /api/crm/negocios/:id/etapa` to move a negocio to a different pipeline stage, validating that the target stage belongs to the negocio's pipeline.

#### Scenario: Negocio moved to a valid stage in same pipeline

- **WHEN** a negocio is moved to a stage that belongs to its pipeline
- **THEN** the negocio's `stage_id` SHALL be updated
- **AND** the system SHALL publish a `crm.negocio.etapa_cambiada` event
- **AND** the system SHALL create an Activity record of type `sistema` documenting "Negocio movido de [etapa_anterior] a [etapa_nueva]"

#### Scenario: Negocio moved to stage in different pipeline is rejected

- **WHEN** a negocio is moved to a stage belonging to a different pipeline
- **THEN** the system SHALL return a validation error: "La etapa no pertenece al pipeline del negocio."

### Requirement: Deal list supports filtering

The system SHALL support filtering the negocio list by `pipeline_id`, `stage_id`, and `estado` query parameters. API paths SHALL use Spanish names.

#### Scenario: Negocios filtered by estado

- **WHEN** a GET request is made to `/api/crm/negocios?estado=abierto`
- **THEN** the system SHALL return only negocios with `estado = 'abierto'`

### Requirement: Deal CRUD is RBAC-protected

The system SHALL require `deal:view` permission for read operations and `deal:manage` permission for create/update operations. Error messages SHALL be in Spanish.

#### Scenario: Member without deal:manage sees Spanish error

- **WHEN** a user with `deal:view` but not `deal:manage` permission attempts to create a negocio
- **THEN** the system SHALL return HTTP 403 with message "No tienes permiso para gestionar negocios."

### Requirement: Deal retains association when contact or company is deleted

The system SHALL use `ON DELETE SET NULL` for contact_id and company_id foreign keys on deals.

#### Scenario: Contact deleted while linked to a negocio

- **WHEN** a contact that is associated with a negocio is deleted
- **THEN** the negocio's `contact_id` SHALL be set to NULL
- **AND** the negocio SHALL NOT be deleted
