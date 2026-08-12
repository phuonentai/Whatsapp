## MODIFIED Requirements

### Requirement: AI audience builder with human approval

The system SHALL provide `POST /crm/campanas/segments/ai-build` that converts a natural-language audience description into a candidate `filter_spec` **and a candidate message draft (`message_draft`)**. The call SHALL go through the metered LLM client with the organization id in context, SHALL be recorded in the ai-usage ledger, and SHALL NOT include any contact PII in the prompt. The returned candidate spec SHALL be validated against the same whitelist as manual segments. The `message_draft` SHALL be a Spanish WhatsApp promotional draft consistent with the description, and SHALL be returned as an optional field: if the model output for the message is unparsable, the system SHALL still return the validated `filter_spec` with `message_draft` omitted. Nothing SHALL be persisted until the user explicitly saves the segment or creates the campaign.

#### Scenario: AI builder returns validated candidate with preview

- **WHEN** a user submits "clientes mayoristas que escribieron este mes"
- **THEN** the system SHALL return a candidate `filter_spec` (e.g. `[{"field":"tag_ids","op":"any","value":[3]},{"field":"recency_days","op":"lte","value":30}]`)
- **AND** a `message_draft` with promotional copy for the described audience
- **AND** a preview count

#### Scenario: AI message output fails to parse

- **WHEN** the LLM returns a valid filter spec but an unparsable message
- **THEN** the system SHALL return 200 with the validated `filter_spec` and preview
- **AND** `message_draft` SHALL be omitted

#### Scenario: AI output fails validation

- **WHEN** the LLM returns a spec containing an unknown field or operator
- **THEN** the system SHALL return HTTP 400 with a Spanish error

#### Scenario: AI usage is metered and org-tagged

- **WHEN** an AI audience build call completes
- **THEN** the tokens SHALL be recorded in the ai-usage ledger for the organization
- **AND** the prompt SHALL contain no contact names, phones, or document numbers

#### Scenario: AI credits exhausted

- **WHEN** the organization's AI credits are exhausted
- **THEN** the system SHALL return HTTP 402 with a Spanish error
- **AND** SHALL NOT call the LLM

### Requirement: Campaign draft lifecycle with idempotent audience snapshot

The system SHALL allow creating a campaign draft under `/crm/campanas` referencing exactly one segment and an **optional message body (`mensaje`)**, and launching it. Launch SHALL evaluate the segment with hard gates, snapshot the matching contacts into `campaign_recipients` with status `pending` (one row per contact, deduplicated by `(campaign_id, contact_id)`), set `recipient_count`, and transition the campaign `draft → ready`. Launch SHALL be idempotent: a campaign not in `draft` SHALL NOT be re-snapshotted.

#### Scenario: Launch snapshots the audience

- **WHEN** a user with `org:manage` permission launches a draft campaign referencing a segment matching 90 contacts
- **THEN** the system SHALL insert 90 `campaign_recipients` rows with status `pending`

#### Scenario: Campaign draft with message body

- **WHEN** a user creates a campaign draft with an optional `mensaje`
- **THEN** the system SHALL persist the `mensaje` with the draft
- **AND** the message SHALL NOT be sent at creation or launch

#### Scenario: Campaign draft without message body

- **WHEN** a user creates a campaign draft without `mensaje` (existing clients)
- **THEN** the system SHALL create the draft normally with `mensaje` null
