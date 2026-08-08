## ADDED Requirements

### Requirement: Pipeline editor creates pipelines with stages in one dialog flow

The system SHALL allow creating a pipeline together with its initial etapas from the CRM pipelines editor, sequencing the existing endpoints `POST /api/crm/pipelines` followed by one `POST /api/crm/pipelines/:id/etapas` per etapa. No bulk creation endpoint is introduced.

#### Scenario: Pipeline and stages created sequentially

- **WHEN** a user creates a pipeline with two etapas from the editor dialog
- **THEN** the system SHALL call `POST /api/crm/pipelines` with the pipeline name
- **AND** SHALL call `POST /api/crm/pipelines/:id/etapas` once per etapa with the created pipeline's id
- **AND** the response of the pipeline list SHALL include the new pipeline with its etapas

#### Scenario: Stage creation failure leaves dialog open

- **WHEN** the pipeline is created but an etapa creation fails
- **THEN** the editor SHALL keep the dialog open
- **AND** SHALL display the error in Spanish
- **AND** a retry SHALL reuse the already-created pipeline id
