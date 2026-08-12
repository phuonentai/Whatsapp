# conversation-context-note Specification

## Purpose
TBD - created by archiving change add-context-to-note. Update Purpose after archive.
## Requirements
### Requirement: Save AI conversation context as a contact note

The system SHALL provide a one-click action on the AI conversation-context panel that saves the conversation summary (summary, detected intent, key facts) as a CRM activity of type `nota` linked to the conversation's contact.

The action SHALL be rendered only when full context is available (not loading, unavailable, consent-gated, or structural-only states) AND the contact id of the conversation is known. The action SHALL reuse the existing CRM activity creation endpoint and SHALL NOT introduce new backend endpoints. Nothing SHALL be persisted on failure; a successful save SHALL show a confirmation and SHALL NOT auto-repeat (the action is disabled after saving).

#### Scenario: Save context from full context

- **WHEN** a user opens a conversation whose context panel shows full context and clicks "Guardar como nota"
- **THEN** the system creates a CRM activity of type `nota` for the conversation's contact
- **AND** the note content contains the summary, detected intent, and key facts
- **AND** a confirmation toast is shown and the action is disabled

#### Scenario: Context not fully available

- **WHEN** the context panel shows loading, unavailable, consent-gated, or structural-only state
- **THEN** no save action is rendered

#### Scenario: Save fails

- **WHEN** the activity creation request fails
- **THEN** no activity is persisted
- **AND** an error toast is shown and the action remains available for retry

#### Scenario: Contact id unknown

- **WHEN** the panel is used in a context without a contact id (e.g., another mount point)
- **THEN** no save action is rendered

