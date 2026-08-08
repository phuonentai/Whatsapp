## MODIFIED Requirements

### Requirement: Deal kanban board in Colombian Spanish

The system SHALL display negocios as cards arranged in pipeline etapa columns. Column headers SHALL use Spanish stage names. Dropdown labels SHALL be in Spanish. The kanban SHALL show a pipeline selector defaulting to the `es_predeterminado` pipeline ("Pipeline de Ventas") and SHALL allow moving a negocio between etapas by dragging the card onto a stage column, persisting the move via `PUT /api/crm/negocios/:id/etapa` with the source and target stage names.

#### Scenario: Kanban shows deals with Spanish UI

- **WHEN** the negocios view loads
- **THEN** etapa columns SHALL show Spanish names (Prospección, Calificado, Propuesta, etc.)
- **AND** deal cards SHALL show monto formatted as "$10.000.000 COP"
- **AND** the pipeline selector SHALL read "Pipeline de Ventas"

#### Scenario: Deal is moved by dragging to another stage column

- **WHEN** a user drags a deal card from one etapa column onto another etapa column and releases it
- **THEN** the system SHALL call `PUT /api/crm/negocios/:id/etapa` with the target `stage_id` and the source/target stage names
- **AND** the deal SHALL appear in the target column after the list refreshes

### Requirement: Pipeline editor with Spanish labels

The system SHALL provide a pipeline editor view at `?view=pipelines` (gated by the `crm_deals` feature) where pipeline owners can view pipelines, create a pipeline with its initial etapas, and edit a stage's nombre, color, or probabilidad. Labels and buttons SHALL be in Spanish ("Nuevo pipeline", "Agregar Etapa", "Guardar", "Cancelar"). The stage list SHALL render "Salida" when a stage's probabilidad is null.

#### Scenario: Editing a stage's details in Spanish

- **WHEN** a user edits a stage's nombre, color, or probabilidad and saves
- **THEN** the stage SHALL be updated via `PUT /api/crm/pipelines/:id/etapas/:stageId`
- **AND** buttons SHALL read "Guardar", "Cancelar", "Agregar Etapa"

#### Scenario: Stage with null probability renders as Salida

- **WHEN** a stage with `probabilidad = null` is displayed in the editor
- **THEN** the stage SHALL render the label "Salida" instead of a percentage

## ADDED Requirements

### Requirement: Deal creation form in Colombian Spanish

The system SHALL provide a "Nuevo negocio" button in the Negocios view that opens a create dialog with fields: `nombre` (required), `monto` (optional), `moneda` (default "COP"), an optional `empresa` select, an optional `contacto` select, and an optional `etapa` select populated from the active pipeline's stages. Labels, buttons, and validation messages SHALL be in Colombian Spanish.

#### Scenario: Create deal from kanban

- **WHEN** a user clicks "Nuevo negocio", fills the form, and submits
- **THEN** the deal SHALL be created via `POST /api/crm/negocios` with the selected pipeline_id
- **AND** the kanban SHALL refresh and SHALL display the new card in the corresponding etapa column

#### Scenario: Empty nombre shows Spanish validation error

- **WHEN** a user submits the deal form without a nombre
- **THEN** the system SHALL display a Spanish validation error without calling the API

### Requirement: Deal edit and delete from the kanban

The system SHALL provide per-card edit and delete actions. Editing opens the deal dialog prefilled with the deal's data and SHALL persist via `PUT /api/crm/negocios/:id`. Deleting SHALL require confirmation via a Spanish confirmation dialog and SHALL delete via `DELETE /api/crm/negocios/:id`.

#### Scenario: Deal edited from card menu

- **WHEN** a user opens a deal card's menu, selects "Editar", changes the monto, and saves
- **THEN** the deal SHALL be updated via `PUT /api/crm/negocios/:id`
- **AND** the kanban SHALL refresh and SHALL show the updated amount

#### Scenario: Deal deleted with confirmation

- **WHEN** a user opens a deal card's menu, selects "Eliminar", and confirms in the Spanish dialog
- **THEN** the deal SHALL be deleted via `DELETE /api/crm/negocios/:id`
- **AND** the card SHALL disappear from the kanban

### Requirement: Pipeline creation with stages in one dialog flow

The system SHALL provide a "Nuevo pipeline" dialog in the Pipelines view that collects the pipeline name and one or more stage rows (nombre, color). On submit, the system SHALL create the pipeline via `POST /api/crm/pipelines` and then create each stage via `POST /api/crm/pipelines/:id/etapas`.

#### Scenario: Create pipeline with stages

- **WHEN** a user clicks "Nuevo pipeline", enters a name, adds two stages with "Agregar Etapa", and submits
- **THEN** the pipeline SHALL be created via `POST /api/crm/pipelines`
- **AND** each stage SHALL be created via `POST /api/crm/pipelines/:id/etapas`
- **AND** the pipeline SHALL appear in the pipeline list with its stages
