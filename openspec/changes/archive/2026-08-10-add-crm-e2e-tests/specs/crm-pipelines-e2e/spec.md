## ADDED Requirements

### Requirement: Pipeline editor CRUD via browser UI

The E2E tests SHALL verify pipeline management through the pipeline editor UI.

#### Scenario: View default pipeline
- **WHEN** a user navigates to the pipeline editor
- **THEN** the default "Pipeline de Ventas" SHALL be displayed with 6 stages (Prospección, Calificado, Propuesta, Negociación, Cerrado Ganado, Cerrado Perdido)

#### Scenario: Create a new pipeline
- **WHEN** a user clicks "Nuevo Pipeline"
- **AND** enters a pipeline name
- **AND** adds at least one stage with name and color
- **THEN** the new pipeline SHALL appear in the pipeline list

#### Scenario: Edit a pipeline name
- **WHEN** a user edits an existing pipeline name
- **THEN** the pipeline list SHALL reflect the updated name

### Requirement: Stage management in pipeline editor

The E2E tests SHALL verify stage creation and editing within a pipeline.

#### Scenario: Add a stage to a pipeline
- **WHEN** a user adds a new stage with name, color, and probability
- **THEN** the stage SHALL appear in the pipeline stage list

#### Scenario: Edit a stage
- **WHEN** a user edits a stage name and color
- **THEN** the stage SHALL reflect the updated values
