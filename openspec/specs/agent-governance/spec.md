## Purpose

Defines the guardrail governance contract where every side-effecting agent action is ruled, while escalation and drafting remain always allowed.

## Requirements

### Requirement: Every side-effecting action is guardrail-governed

The system SHALL route every agent action that can produce a side effect (sending a message) through a `GuardrailService` before the side effect is performed. The guardrail input SHALL be a pure snapshot containing the action, the draft, the organization settings, contact facts (including consent), the daily sent count, the autonomy flag, the approver member id, and the current time. Guardrails SHALL be deterministic, transport-free Go logic — no external policy engines.

#### Scenario: Send evaluated before execution

- **WHEN** an approved draft or an autonomous send is about to be sent
- **THEN** the system SHALL evaluate the `send_message` guardrail with a snapshot of settings, contact facts, and time
- **AND** SHALL only invoke the WhatsApp send API when the decision is `allow`

#### Scenario: Denied action performs no side effect

- **WHEN** the guardrail decision for an action is `deny`
- **THEN** the system SHALL NOT perform the side effect
- **AND** SHALL record the denial in the audit ledger with the reasons

### Requirement: Guardrail rule table for send_message

The system SHALL implement the following deterministic rules for `send_message`: kill switch (deny always, including human approvals); discount cap — drafts containing a percentage above `guardrails.never.max_discount_percent` are denied; forbidden terms — drafts containing `guardrails.never.forbidden_terms` are denied; escalation terms — drafts containing `guardrails.escalate.terms` are denied for autonomous sends; consent — autonomous sends require consent `granted` when `consent_required` is enabled; send window — autonomous sends only within the configured `autopilot_start`/`autopilot_end` window (timezone-aware, overnight windows supported); daily limit — autonomous sends stop once `max_daily_messages` is reached (0 = unlimited).

#### Scenario: Kill switch blocks send

- **WHEN** an org has `kill_switch = true` and a send is evaluated
- **THEN** the decision SHALL be `deny` with reason `kill_switch`

#### Scenario: Discount above cap denied

- **WHEN** a draft contains a percentage higher than the configured cap
- **THEN** the decision SHALL be `deny` with reason `discount_exceeds_cap`

#### Scenario: Forbidden term denied

- **WHEN** a draft contains a forbidden term
- **THEN** the decision SHALL be `deny` with reason `forbidden_term`

#### Scenario: Escalation topic never autonomous

- **WHEN** an autonomous send draft matches an escalation term
- **THEN** the decision SHALL be `deny` with reason `escalation_match`

#### Scenario: Consent not granted blocks autonomous send

- **WHEN** an autonomous send is evaluated for a contact without consent `granted` and consent is required
- **THEN** the decision SHALL be `deny` with reason `consent_required`

#### Scenario: Outside window blocks autonomous send

- **WHEN** an autonomous send is evaluated outside the configured send window
- **THEN** the decision SHALL be `deny` with reason `outside_send_window`

#### Scenario: Daily limit blocks autonomous send

- **WHEN** `max_daily_messages` is greater than zero and the daily count equals or exceeds it
- **THEN** the decision SHALL be `deny` with reason `daily_limit_reached`

#### Scenario: Human approval bypasses window/consent/limit

- **WHEN** a human approves a draft (autonomous = false)
- **THEN** the window, consent, and daily-limit rules SHALL NOT apply
- **AND** the kill switch and never/escalate rules SHALL still apply

### Requirement: Escalation and drafting are always allowed

The system SHALL consider `escalate_human` and `generate_draft` non-governable actions that are always allowed. Unknown actions and guardrail evaluation errors SHALL produce deny with reason `guardrail_error` (fail-safe direction).

#### Scenario: Escalation always allowed

- **WHEN** the action is `escalate_human` under any org settings, including kill switch on
- **THEN** the decision SHALL be `allow`

#### Scenario: Unknown action denied

- **WHEN** the action is not one of the known guardrail actions
- **THEN** the decision SHALL be `deny` with reason `guardrail_error`

### Requirement: Parameters as data, rules as code

The system SHALL store all tunable guardrail parameters in `agent_settings.guardrails` (JSONB) and in the settings row; guardrail Go code SHALL contain only logic. Default guardrails SHALL be: discount cap 10%, no forbidden terms, escalation terms [abogado, legal, garantía, demanda, superintendencia].

#### Scenario: Settings changes take effect immediately

- **WHEN** an org updates its guardrails or settings
- **THEN** subsequent guardrail evaluations SHALL observe the updated values without a deploy

### Requirement: Append-only agent action audit

The system SHALL append one row to `agent_actions` for every governance evaluation of a side-effecting action, containing the action, the decision (`allow`/`deny`/`skip`), the policy input snapshot, the reasons, the approving Stytch `member_id` (when applicable), the linked `whatsapp_message_id`, and the `request_id`. Rows SHALL be immutable.

#### Scenario: Approved send is audited

- **WHEN** a send is approved, evaluated, and executed
- **THEN** an `agent_actions` row SHALL be appended with decision `allow`, the policy input snapshot, the approving member id, and the sent message id

#### Scenario: Denied action is audited

- **WHEN** a guardrail evaluation denies an action
- **THEN** an `agent_actions` row SHALL be appended with decision `deny` and the reason list
- **AND** the row SHALL NOT be modified afterwards

### Requirement: Guardrail tests in CI

The system SHALL maintain Go unit tests covering every rule in the guardrail table (kill switch, discount cap, forbidden terms, escalation terms, consent, window, daily limit, autonomy semantics, fail-safe denial) and SHALL run them in CI.

#### Scenario: CI runs guardrail tests

- **WHEN** a change touches the guardrail service
- **THEN** CI SHALL run the guardrail unit tests and SHALL fail the build on any failing test
