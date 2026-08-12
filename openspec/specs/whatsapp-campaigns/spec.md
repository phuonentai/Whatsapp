## Purpose

Specification for WhatsApp campaign management: contact segments with whitelisted filter specs, mandatory consent and phone hard gates, campaign draft and launch lifecycle, and AI-assisted audience building with human approval.

## Requirements

### Requirement: Segment CRUD with whitelisted filter specs

The system SHALL allow organizations to create, read, update, and delete saved contact segments under `/crm/campanas/segments`. A segment SHALL store a `filter_spec` JSON array with AND semantics, validated against a whitelist of fields and operators: `source` (eq), `lead_status` (eq), `company_id` (eq), `assigned_to` (eq), `tag_ids` (any), `recency_days` (lte), `search` (contains). Unknown fields, unknown operators, or empty values SHALL be rejected with HTTP 400 and a Spanish error message. Segments SHALL be org-scoped.

#### Scenario: Segment created with valid filters

- **WHEN** a user with `org:manage` permission creates a segment with `filter_spec` `[{"field":"lead_status","op":"eq","value":"cliente"},{"field":"tag_ids","op":"any","value":[3,7]}]`
- **THEN** the segment SHALL be persisted under the user's organization
- **AND** the system SHALL return HTTP 201 with the created segment

#### Scenario: Invalid filter field rejected

- **WHEN** a user creates a segment with `filter_spec` containing `{"field":"password","op":"eq","value":"x"}`
- **THEN** the system SHALL return HTTP 400 with a Spanish error
- **AND** SHALL NOT persist the segment

#### Scenario: Segment list is org-scoped

- **WHEN** a user with `org:view` permission lists segments
- **THEN** the system SHALL return only segments belonging to the user's organization

#### Scenario: User without permission is denied

- **WHEN** a user without `org:manage` permission attempts to create or update a segment
- **THEN** the system SHALL return HTTP 403 with a Spanish error message

### Requirement: Segment evaluation applies mandatory hard gates

The system SHALL evaluate a segment's `filter_spec` against the organization's contacts AND always append mandatory hard gates that cannot be expressed in the filter spec: `consent_status = 'granted'` and a valid E.164 `phone_number`. Contacts failing a gate SHALL be excluded from every evaluation result regardless of user filters.

#### Scenario: Contact without consent is excluded

- **WHEN** a segment matches a contact whose `consent_status` is `none` or `requested` or `withdrawn`
- **THEN** the contact SHALL NOT appear in the evaluation result
- **AND** the exclusion SHALL be reported in the preview response as a gate exclusion

#### Scenario: Contact with invalid phone is excluded

- **WHEN** a segment matches a contact whose `phone_number` is empty or not valid E.164
- **THEN** the contact SHALL NOT appear in the evaluation result

#### Scenario: User filters cannot override gates

- **WHEN** a user creates a segment whose `filter_spec` includes a filter attempting to include withdrawn-consent contacts
- **THEN** the validation SHALL reject the attempt (no such field/op exists in the whitelist)
- **AND** withdrawn contacts SHALL remain excluded at evaluation time

### Requirement: Segment preview count

The system SHALL provide `GET /crm/campanas/segments/:id/preview` (and an evaluation preview for candidate specs) that returns the live count of matching contacts after hard gates, plus the count of contacts excluded by gates. Preview SHALL NOT persist anything.

#### Scenario: Preview returns count and gate exclusions

- **WHEN** a user with `org:view` permission requests preview for a segment whose spec matches 120 contacts, of which 30 have non-granted consent
- **THEN** the system SHALL return `{"total": 90, "excluded_by_gates": 30}`

#### Scenario: Preview requires no mutation

- **WHEN** a preview is requested
- **THEN** the system SHALL NOT create or modify any segment, campaign, or recipient rows

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

### Requirement: Campaign routes are RBAC-protected and org-scoped

The system SHALL require `org:manage` for creating/updating segments, creating/launching campaigns, and AI build; `org:view` for listing segments, campaigns, and previews. All campaign routes SHALL sit behind the existing `auth` + `org_context` + `subscription` middleware and SHALL operate exclusively within the resolved organization.

#### Scenario: Read access for org viewers

- **WHEN** a user with `org:view` permission lists campaigns or requests a preview
- **THEN** the system SHALL return the data for the user's organization

#### Scenario: Write access denied without manage

- **WHEN** a user without `org:manage` permission attempts to launch a campaign
- **THEN** the system SHALL return HTTP 403 with a Spanish error message
