## ADDED Requirements

### Requirement: AI actions in the command palette

The system SHALL expose AI entry points in the ⌘K command palette as a dedicated "IA" group rendered above the navigation destinations. AI actions SHALL be discoverable with the same fuzzy filtering as navigation commands, SHALL execute when selected, and SHALL close the palette.

The system SHALL provide at least the following actions: open the AI knowledge assistant, start a new AI chat, and open the AI campaign audience builder. Starting a new AI chat SHALL reset the knowledge chat to a fresh conversation (not merely navigate). AI actions SHALL NOT modify the existing navigation destinations or the `g <key>` shortcuts.

#### Scenario: Palette shows the IA group

- **WHEN** the user opens the palette and types "asistente"
- **THEN** the IA group appears with the matching AI actions

#### Scenario: Open the AI assistant

- **WHEN** the user selects "Preguntar al asistente"
- **THEN** the app navigates to the knowledge assistant page and closes the palette

#### Scenario: Start a new AI chat

- **WHEN** the user selects "Nueva conversación de IA"
- **THEN** the app navigates to the knowledge page
- **AND** the knowledge chat resets to a fresh conversation (messages cleared, no session selected)

#### Scenario: Open the AI audience builder

- **WHEN** the user selects the campaign audience AI action
- **THEN** the app navigates to the CRM campaigns view and closes the palette

#### Scenario: Navigation commands unchanged

- **WHEN** the user selects any existing navigation destination
- **THEN** behavior is identical to before (URL navigation, `g <key>` shortcuts unaffected)
