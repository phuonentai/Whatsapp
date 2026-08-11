# Delta Spec: inbox-ui — ai-ux-polish

## MODIFIED Requirements

### Requirement: User approves a pending agent suggestion
The system SHALL render pending agent suggestions for the selected conversation, allow the user to approve a pending suggestion, and SHALL remove it from the pending list on approval via `POST /api/agent/suggestions/:id/approve`. While pending suggestions load, the panel SHALL render a skeleton placeholder instead of disappearing. Pending state SHALL be tracked per suggestion so an in-flight approve/reject disables only that suggestion's actions. The user SHALL be able to expand the conversation context a suggestion was based on before deciding.

#### Scenario: Approving a suggestion removes it from pending
- **WHEN** a pending suggestion exists for the conversation and user clicks approve
- **THEN** the suggestion is removed from the pending suggestions panel

#### Scenario: Rejecting a suggestion dismisses it
- **WHEN** a pending suggestion exists for the conversation and user clicks reject
- **THEN** the suggestion is dismissed from the pending suggestions panel

#### Scenario: Panel loads with skeleton
- **WHEN** the pending-suggestions query is loading
- **THEN** the panel SHALL render a skeleton placeholder and SHALL NOT flash out of existence

#### Scenario: Per-suggestion pending state
- **WHEN** the user approves one of several pending suggestions
- **THEN** only that suggestion's actions SHALL show pending state and the others SHALL remain interactive

#### Scenario: Context expansion shows conversation excerpt
- **WHEN** the user expands the context control on a suggestion
- **THEN** the panel SHALL show a read-only excerpt of the conversation the suggestion was based on
