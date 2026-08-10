## Purpose

Define the E2E behavior of the CRM activities (Actividad) module: creating note/call/task activities and filtering activities by type and linked entity through the browser UI.

## Requirements

### Requirement: Activity timeline CRUD via browser UI

The system SHALL support activity creation and display through the activity timeline UI, verified by E2E tests.

#### Scenario: Create a note activity
- **WHEN** a user navigates to the Actividad tab
- **AND** clicks "Nueva Actividad"
- **AND** selects type "Nota"
- **AND** fills in subject and content
- **THEN** the note SHALL appear in the activity timeline

#### Scenario: Create a call activity
- **WHEN** a user creates an activity with type "Llamada"
- **AND** fills in subject, content, and performed date
- **THEN** the call SHALL appear in the timeline

#### Scenario: Create a task activity
- **WHEN** a user creates an activity with type "Tarea"
- **AND** fills in subject, content, and due date
- **THEN** the task SHALL appear in the timeline

### Requirement: Activity filtering

The system SHALL support activity filtering by type and by linked entity, verified by E2E tests.

#### Scenario: Filter activities by type
- **WHEN** a user selects a type filter (e.g., "Nota")
- **THEN** the timeline SHALL display only activities of that type

#### Scenario: View activities linked to a contact
- **WHEN** a user views a contact detail
- **THEN** the activities section SHALL display only activities linked to that contact

#### Scenario: View activities linked to a deal
- **WHEN** a user views a deal detail
- **THEN** the activities section SHALL display only activities linked to that deal
