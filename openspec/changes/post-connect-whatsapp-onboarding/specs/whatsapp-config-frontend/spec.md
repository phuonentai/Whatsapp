## MODIFIED Requirements

### Requirement: WhatsApp settings view supports embedded signup entry

The system SHALL render the embedded-signup connect entry and, once a connection succeeds, the post-connect next-steps flow instead of terminating at the connected banner alone.

#### Scenario: Post-connect flow rendered after success

- **WHEN** the embedded signup exchange succeeds and the configuration becomes active
- **THEN** the WhatsApp settings view SHALL render the post-connect next-steps card alongside the connected state

#### Scenario: Connect entry preserved for inactive state

- **WHEN** no active configuration exists
- **THEN** the existing connect empty-state entry SHALL remain unchanged
