## MODIFIED Requirements

### Requirement: Settings shows WhatsApp configuration

The system SHALL render the WhatsApp configuration section using the typed copy layer in Spanish-first voice. Primary connect copy SHALL use plain language; Meta developer tokens SHALL be confined to the collapsed advanced panel.

#### Scenario: WhatsApp config section renders Spanish copy

- **WHEN** a user opens the settings WhatsApp section
- **THEN** the section title, description, connect empty-state, status labels, and connect button SHALL be Spanish strings resolved from the copy layer

#### Scenario: Connected state renders Spanish

- **WHEN** a WhatsApp configuration is active
- **THEN** the connected banner and message-receiving status SHALL render the Spanish strings from the copy layer
