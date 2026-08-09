## ADDED Requirements

### Requirement: Inbound message triggers the agent pipeline

The system SHALL start or advance a conversation flow whenever a `whatsapp.message.received` event is published, by subscribing to the same event consumed by the CRM message listener. The pipeline SHALL resolve the contact and active conversation idempotently (independent of the CRM listener's event ordering) and SHALL NOT block the webhook HTTP response (the eventbus dispatches handlers asynchronously).

#### Scenario: Inbound message starts a flow

- **WHEN** a `whatsapp.message.received` event arrives for a contact
- **THEN** the system SHALL create or reuse an active `conversation_flows` row in `running` status
- **AND** SHALL NOT modify the webhook ingress response

#### Scenario: Redelivered webhook does not double-process

- **WHEN** a retried webhook delivers a message whose `whatsapp_message_id` already has a pending suggestion
- **THEN** the system SHALL skip processing for that message
- **AND** SHALL NOT create a second suggestion or send

#### Scenario: Non-text messages are ignored

- **WHEN** a message event has a non-text type or empty content
- **THEN** the pipeline SHALL return without analysis or suggestions

### Requirement: Agent analysis is LLM-powered and metered

The system SHALL run a consolidated analysis LLM call (structured intent, sentiment, and suggested reply) through the existing metered client so every agent LLM call is recorded in the ai-usage ledger. Exhausted AI credits SHALL escalate instead of burning unmetered tokens.

#### Scenario: Analysis records AI consumption

- **WHEN** an analysis LLM call completes with the organization id in context
- **THEN** the tokens SHALL be recorded in the ai-usage ledger via the metered client
- **AND** the pipeline SHALL continue with the structured result

#### Scenario: Credits exhausted escalate to a human

- **WHEN** the organization's AI credits are exhausted before analysis
- **THEN** the system SHALL insert an `escalation` suggestion with reason `ai_credits_exhausted`
- **AND** SHALL mark the flow `awaiting_human`

### Requirement: Copilot mode requires human approval (L1)

In copilot mode the system SHALL NOT send outbound messages autonomously. Every draft SHALL be saved as a pending `reply` suggestion, superseding older pending replies for the same conversation.

#### Scenario: Draft saved as pending suggestion

- **WHEN** the pipeline produces a draft for an organization in copilot mode
- **THEN** the system SHALL insert an `agent_suggestions` row with status `pending`, source `copilot`, and the draft body
- **AND** SHALL NOT call the WhatsApp send API

#### Scenario: Older pending replies are superseded

- **WHEN** a new draft is saved for a conversation that already has pending replies
- **THEN** the existing pending replies SHALL be marked `superseded`

#### Scenario: Rep approves a draft

- **WHEN** a rep approves a pending suggestion via `POST /api/agent/suggestions/:id/approve`
- **THEN** the system SHALL evaluate the send guardrails
- **AND** if allowed, SHALL send the message via the CRM outbound service (circuit-breakered WhatsApp client)
- **AND** SHALL mark the suggestion `approved` with the acting Stytch `member_id`

#### Scenario: Rep edits before sending

- **WHEN** a rep edits the draft body (`edited_body`) and approves
- **THEN** the system SHALL send the edited body, not the original draft

#### Scenario: Rep rejects a draft

- **WHEN** a rep rejects a pending suggestion via `POST /api/agent/suggestions/:id/reject`
- **THEN** the system SHALL mark the suggestion `rejected`
- **AND** SHALL audit the decision as `deny` with reason `human_rejection`

### Requirement: Autopilot mode is guardrail-bounded

In autopilot mode the system SHALL evaluate the send guardrails with autonomous semantics before sending. Denied sends SHALL fall back to pending suggestions.

#### Scenario: Autopilot send allowed

- **WHEN** guardrails allow an autonomous send
- **THEN** the system SHALL send the draft and mark the flow `succeeded`

#### Scenario: Autopilot send denied falls back

- **WHEN** guardrails deny an autonomous send for a non-escalation reason
- **THEN** the system SHALL insert a pending suggestion with source `autopilot_fallback`
- **AND** SHALL mark the flow `awaiting_human`

#### Scenario: Escalation-match topics always escalate

- **WHEN** a draft matches an escalation term in autopilot mode
- **THEN** the system SHALL NOT send
- **AND** SHALL insert an `escalation` suggestion and mark the flow `awaiting_human`

### Requirement: Consent state machine (Ley 1581)

The system SHALL track contact consent (`none`, `requested`, `granted`, `withdrawn`) on `crm.contacts`. When `consent_required` is enabled, a new contact with consent `none` SHALL receive the consent template as the first outbound message and SHALL NOT receive analysis-driven replies until consent is granted.

#### Scenario: New contact receives consent template

- **WHEN** an inbound message arrives from a contact with consent `none` and the org requires consent
- **THEN** the system SHALL send the configured consent template (or the built-in Ley 1581 default)
- **AND** SHALL mark the contact consent `requested`
- **AND** SHALL audit `consent_request`

#### Scenario: Affirmative reply grants consent

- **WHEN** a contact with consent `requested` replies affirmatively (e.g. "sí", "acepto", "autorizo")
- **THEN** the system SHALL mark the contact consent `granted` with a timestamp
- **AND** SHALL audit `consent_grant` and continue analysis

#### Scenario: Consent not granted blocks autonomous sends

- **WHEN** an autonomous send is evaluated for a contact whose consent is not `granted`
- **THEN** the guardrails SHALL deny with reason `consent_required`

### Requirement: Escalation to a human is always allowed

The system SHALL treat escalation as a non-governable action: `escalate_human` SHALL always be allowed by the guardrail layer, so the agent can never trap a lead in automation.

#### Scenario: Escalation under any settings

- **WHEN** an escalation is evaluated under any org settings, including kill switch on
- **THEN** the decision SHALL be `allow`

### Requirement: Flow lifecycle

The system SHALL track flow status (`running`, `awaiting_human`, `succeeded`, `failed`, `cancelled`) and SHALL cancel an in-progress flow when the org's kill switch is enabled.

#### Scenario: Kill switch cancels in-progress flow

- **WHEN** a message arrives and the org's kill switch is enabled
- **THEN** the flow SHALL be marked `cancelled`
- **AND** the action SHALL be audited as `skip` with reason `kill_switch`

### Requirement: Agent routes are RBAC-protected

The system SHALL require the existing `org:manage` permission for approvals, settings updates, and compliance operations, and `org:view` for reading suggestions, settings, and flow debug. Agent routes SHALL sit behind `auth` + `org_context` + `subscription` middleware like other CRM routes.

#### Scenario: Member without permission is denied

- **WHEN** a member without `org:manage` attempts to approve a suggestion, update settings, or run compliance operations
- **THEN** the system SHALL return HTTP 403 with a Spanish error message

### Requirement: Data subject rights (Ley 1581 export/forget)

The system SHALL expose contact data export and anonymization. Exports SHALL mask PII when consent is withdrawn; forget SHALL be idempotent.

#### Scenario: Export returns full data bundle

- **WHEN** an `org:manage` member calls `GET /api/agent/compliance/export/:contactId`
- **THEN** the system SHALL return the contact plus its conversations and messages

#### Scenario: Export masks withdrawn-consent PII

- **WHEN** the contact's consent is `withdrawn`
- **THEN** the exported phone, name, email, and document fields SHALL be masked

#### Scenario: Forget anonymizes idempotently

- **WHEN** an `org:manage` member calls `POST /api/agent/compliance/forget/:contactId`
- **THEN** the contact's personal fields SHALL be scrubbed and consent set to `withdrawn`
- **AND** a repeated call SHALL succeed without further changes
