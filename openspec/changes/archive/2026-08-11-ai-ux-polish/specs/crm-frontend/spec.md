# Delta Spec: crm-frontend — ai-ux-polish

## ADDED Requirements

### Requirement: AI audience builder renders structured results

The CRM campaign manager SHALL accept a natural-language audience description, call the AI build mutation, and render the result as a structured card (segment criteria as labeled chips/lists, estimated audience size, consent-exclusion notice) with accept, edit, and regenerate actions. The result SHALL be presented as structured UI, NOT as a raw JSON `<pre>` block. Labels and errors SHALL be in Colombian Spanish.

#### Scenario: Natural-language input builds an audience

- **WHEN** a user types an audience description and submits it
- **THEN** the builder SHALL call the AI build mutation and render a pending state while it runs

#### Scenario: Result renders as structured card

- **WHEN** the AI build returns a result
- **THEN** the result SHALL render criteria as labeled chips/lists, the audience size, and the consent-exclusion notice
- **AND** the raw JSON payload SHALL NOT be the primary display

#### Scenario: Build failure shows Spanish error

- **WHEN** the AI build mutation fails
- **THEN** a Spanish error message SHALL render inline and the builder SHALL remain usable

#### Scenario: Accept creates the segment

- **WHEN** the user accepts the AI-built audience
- **THEN** the segment SHALL be created via the existing segment creation path and SHALL appear in the segments list
