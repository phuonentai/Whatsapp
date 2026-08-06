## ADDED Requirements

### Requirement: Pipeline entity with organization scoping

The system SHALL store sales pipelines (pipelines de ventas) in `crm.pipelines` scoped by `organization_id`, with `nombre`, `es_predeterminado`, and `orden` fields.

#### Scenario: Pipeline created for an organization

- **WHEN** a pipeline is created with a name for an organization
- **THEN** the pipeline SHALL be persisted and scoped to that organization

### Requirement: PipelineStage entity with Spanish names and display properties

The system SHALL store pipeline stages (etapas) in `crm.pipeline_stages` linked to a pipeline, with `nombre`, `orden`, `color` (hex), and `probabilidad` (integer 0-100).

#### Scenario: Stages created with Spanish names

- **WHEN** stages are created for a pipeline with nombres like "Calificado", "Propuesta"
- **THEN** the stages SHALL be persisted with the specified Spanish names
- **AND** stages SHALL be returned in `orden` ascending

#### Scenario: Stage with no probability (etapa de salida)

- **WHEN** a stage is created with probabilidad=NULL (e.g., "Cerrado Perdido")
- **THEN** the stage SHALL be persisted with null probabilidad

### Requirement: Default pipeline auto-seeded with Colombian Spanish stages

The system SHALL automatically create a default pipeline when a tenant first accesses pipeline data if no pipeline exists for that organization. Default stages SHALL use Colombian Spanish.

#### Scenario: First access creates default pipeline in Spanish

- **WHEN** a GET request is made to `/api/crm/pipelines` and no pipelines exist for the organization
- **THEN** the system SHALL create a default "Pipeline de Ventas" with stages:
  - Prospección (orden: 1, color: #6B7280, probabilidad: 10%)
  - Calificado (orden: 2, color: #3B82F6, probabilidad: 25%)
  - Propuesta (orden: 3, color: #8B5CF6, probabilidad: 50%)
  - Negociación (orden: 4, color: #F59E0B, probabilidad: 75%)
  - Cerrado Ganado (orden: 5, color: #10B981, probabilidad: 100%)
  - Cerrado Perdido (orden: 6, color: #EF4444, probabilidad: 0%)
- **AND** SHALL return the seeded pipeline with its etapas

#### Scenario: Subsequent access returns existing pipeline without duplication

- **WHEN** a GET request is made to `/api/crm/pipelines` and pipelines already exist
- **THEN** the system SHALL return the existing pipelines without creating duplicates

### Requirement: Pipeline list includes stages

The system SHALL include the pipeline's etapas when listing or retrieving pipelines.

#### Scenario: List pipelines with etapas

- **WHEN** a GET request is made to `/api/crm/pipelines`
- **THEN** each pipeline in the response SHALL include its array of etapas ordered by orden

### Requirement: Pipeline stage can be created within a pipeline

The system SHALL allow creating new stages within an existing pipeline via `POST /api/crm/pipelines/:id/etapas`.

#### Scenario: New etapa added to pipeline

- **WHEN** an etapa is created for a pipeline with nombre, orden, color, and probabilidad
- **THEN** the etapa SHALL be added to the pipeline

### Requirement: Pipeline deletion restricted when deals exist

The system SHALL use `ON DELETE RESTRICT` for the deal-to-pipeline foreign key, preventing pipeline deletion while negocios reference it.

#### Scenario: Pipeline with active negocios cannot be deleted

- **WHEN** a user attempts to delete a pipeline that has associated negocios
- **THEN** the system SHALL return error: "No se puede eliminar el pipeline porque tiene negocios activos."

### Requirement: Pipeline management is RBAC-protected

The system SHALL require `pipeline:view` permission for read operations and `pipeline:manage` permission for create/update operations.

#### Scenario: Admin manages pipelines

- **WHEN** a user with `pipeline:manage` permission creates or updates a pipeline
- **THEN** the operation SHALL succeed

#### Scenario: Member without pipeline:manage sees Spanish error

- **WHEN** a user with `pipeline:view` but not `pipeline:manage` attempts to create a pipeline
- **THEN** the system SHALL return HTTP 403 with message "No tienes permiso para gestionar pipelines."
