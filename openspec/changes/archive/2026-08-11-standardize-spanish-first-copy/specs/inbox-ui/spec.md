## MODIFIED Requirements

### Requirement: Inbox handles empty and failure states

The system SHALL render inbox empty and failure states using the typed copy layer in Spanish-first voice, replacing the current English empty-state strings.

#### Scenario: Channel empty states render Spanish

- **WHEN** the conversation list is empty for a channel (WhatsApp or Instagram) or for all channels
- **THEN** the empty-state message SHALL be a Spanish string resolved from the copy layer that explains how to get started (connect the channel in Settings)

#### Scenario: Thread empty state renders Spanish

- **WHEN** an open conversation has no messages
- **THEN** the thread empty-state hint SHALL be a Spanish string resolved from the copy layer
