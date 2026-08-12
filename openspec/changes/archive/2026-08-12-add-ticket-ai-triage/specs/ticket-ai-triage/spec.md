## ADDED Requirements

### Requirement: AI ticket triage

The system SHALL provide an AI triage endpoint for a ticket that drafts an internal note and suggests a priority from the stored ticket title and description. The endpoint SHALL require a valid authenticated session with at least `ticket:view` permission and an org context, and SHALL only operate on tickets belonging to that org.

The system SHALL gate every triage call on AI credits: when the org has a credit cap and no credits remain, the endpoint SHALL return 402 with a machine-readable `ai_credits_exhausted` code; ledger-read failures SHALL fail open exactly like the agent analysis path.

The suggested priority SHALL be validated against the ticket domain's valid priority set. When the model returns an invalid priority, the system SHALL return the drafted note without a priority rather than failing the call.

Triage SHALL NOT mutate the ticket: the note is a draft and the priority is a suggestion. Applying either is a separate, explicit user action through the existing note/priority endpoints.

#### Scenario: Triage an existing ticket

- **WHEN** an authenticated user requests triage for a ticket in their org
- **THEN** the system returns 200 with `{"data": {"note": "<drafted internal note>", "priority": "alta" | null}, "success": true}`

#### Scenario: Invalid model priority is dropped

- **WHEN** the model returns a priority outside the valid set
- **THEN** the system returns 200 with the drafted note and `priority: null`

#### Scenario: Credits exhausted

- **WHEN** an authenticated user requests triage and the org has no remaining AI credits
- **THEN** the system returns 402 with `ai_credits_exhausted` and no LLM call is made

#### Scenario: Unauthenticated request

- **WHEN** an unauthenticated request hits the triage endpoint
- **THEN** the system returns 401

#### Scenario: Ticket not in the org

- **WHEN** an authenticated user requests triage for a ticket that does not exist or belongs to another org
- **THEN** the system returns 404

#### Scenario: Composer fills the note draft and priority suggestion

- **WHEN** the frontend receives a successful triage response
- **THEN** the note input is filled with the drafted note
- **AND** a priority suggestion is shown with an explicit Apply action that uses the existing priority endpoint
- **AND** nothing is saved automatically

#### Scenario: Triage failure leaves the form untouched

- **WHEN** the triage request fails (network, LLM, or 402)
- **THEN** the form keeps its current values
- **AND** a Spanish error toast is shown
