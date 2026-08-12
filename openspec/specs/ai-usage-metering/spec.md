## Purpose

Defines the AI usage ledger schema, idempotent usage recording, and token-to-credit conversion for metered LLM features.

## Requirements

### Requirement: AI usage ledger schema

The system SHALL persist per-organization AI token consumption in a dedicated ledger under the `subscription_billing` schema, with a running-totals table and an append-only event table, plus a period credit allowance column on `quota_tracking`.

#### Scenario: Running totals row per organization per period

- **WHEN** an organization first records AI usage in a billing period
- **THEN** the system SHALL create a `subscription_billing.ai_usage` row keyed by `(organization_id, period_start)`
- **AND** the row SHALL track `tokens_input`, `tokens_output`, `tokens_embedding`, and `credits_used` totals for that period

#### Scenario: Every usage event is appended to the audit ledger

- **WHEN** AI usage is recorded
- **THEN** the system SHALL append an immutable row to `subscription_billing.ai_usage_events` containing organization_id, feature, model, tokens (input/output/embedding), credits_consumed, request_id, and created_at
- **AND** the event row SHALL NOT be modified or deleted by any service

#### Scenario: Credit allowance stored on quota tracking

- **WHEN** a subscription is synced with product metadata containing the `ai_credits_max` key
- **THEN** the system SHALL store the period credit allowance in the `ai_credits_max` column of `subscription_billing.quota_tracking`
- **AND** organizations without the key SHALL have an allowance of zero

### Requirement: AiUsageLedger records usage idempotently

The system SHALL provide an `AiUsageLedger` service in the billing module that records token/credit consumption, returns current-period usage, and exposes a read-only credit check. Recording MUST be idempotent per `request_id`.

#### Scenario: Successful completion recording

- **WHEN** an LLM completion or embedding returns tokens used for an organization
- **THEN** the ledger SHALL increment the period `ai_usage` totals atomically
- **AND** SHALL append a corresponding `ai_usage_events` row

#### Scenario: Duplicate request_id is ignored

- **WHEN** a usage record arrives whose `request_id` already exists in `ai_usage_events` for the organization
- **THEN** the ledger SHALL NOT increment totals again and SHALL NOT append a second event row

#### Scenario: No allowance row yet

- **WHEN** usage is recorded for an organization with no prior `ai_usage` row in the current period
- **THEN** the ledger SHALL upsert a new row initialized with the recorded amounts and the current period bounds

#### Scenario: Current period usage is readable

- **WHEN** a caller asks for the organization's AI usage
- **THEN** the ledger SHALL return tokens (input/output/embedding), credits used, period bounds, and remaining credits computed as `ai_credits_max - credits_used`

### Requirement: Token-to-credit conversion

The system SHALL convert token consumption into credits using a central per-model conversion map, differentiated by input, output, and embedding tokens, with a documented fallback rate for unknown models.

#### Scenario: Known model conversion

- **WHEN** tokens are recorded for a model present in the conversion map
- **THEN** the system SHALL compute credits from that model's input/output/embedding rates

#### Scenario: Unknown model fallback

- **WHEN** tokens are recorded for a model absent from the conversion map
- **THEN** the system SHALL apply the documented default fallback rate
- **AND** SHALL log the unknown model for rate-table maintenance

### Requirement: All LLM invocations are instrumented

The system SHALL wrap the LLM client with a metered decorator so that every completion, streaming completion, and embedding call records its consumed tokens into the ledger.

#### Scenario: Chat completion records usage

- **WHEN** a RAG chat response completes successfully through the metered client
- **THEN** the system SHALL record the completion's `TokensUsed` against the requesting organization

#### Scenario: Streaming completion records usage

- **WHEN** a streaming completion finishes
- **THEN** the system SHALL record the final token usage of the aggregated response

#### Scenario: Embedding records usage

- **WHEN** a document or query embedding is generated
- **THEN** the system SHALL record the embedding's token usage against the organization

#### Scenario: Failed LLM call records nothing

- **WHEN** the underlying LLM client returns an error
- **THEN** the metered decorator SHALL return the error without recording usage

#### Scenario: Ledger failure does not break the response

- **WHEN** recording usage fails (e.g., database error)
- **THEN** the metered decorator SHALL log the failure and still return the successful LLM response to the caller

### Requirement: AI usage is exposed in tenant entitlement context

The system SHALL surface AI usage and credit state through the existing `FeatureProvider.GetEntitlement` result so every handler sees it in tenant context.

#### Scenario: Entitlement carries AI usage

- **WHEN** `GetEntitlement` is called for an organization
- **THEN** `Entitlement.Usage` SHALL include `ai_tokens_input`, `ai_tokens_output`, `ai_tokens_embedding`, `ai_credits_used`, and `ai_credits_remaining`

#### Scenario: Entitlement carries AI quota

- **WHEN** `GetEntitlement` is called for an organization with an `ai_credits_max` allowance
- **THEN** `Entitlement.Quotas["ai_credits"]` SHALL equal the period allowance

### Requirement: AI route credit enforcement

The system SHALL guard AI-facing routes so an organization with exhausted credits receives a structured rejection before the LLM is called.

#### Scenario: Credits remain, request proceeds

- **WHEN** an organization has remaining credits for the period and calls a guarded AI route
- **THEN** the request SHALL proceed to the handler

#### Scenario: Credits exhausted returns 402

- **WHEN** an organization has zero remaining credits (`ai_credits_max > 0` and `credits_used >= ai_credits_max`) and calls a guarded AI route
- **THEN** the system SHALL return HTTP 402 with JSON body `{"error": "ai_credits_exhausted"}`
- **AND** SHALL abort the request before any LLM invocation

#### Scenario: No allowance configured allows access

- **WHEN** an organization has `ai_credits_max = 0` (no allowance configured)
- **THEN** the credit guard SHALL NOT block the request (recording still applies)

### Requirement: Metered LLM invocations require an active subscription

On non-paywalled inbound paths (WhatsApp webhook → agent), the system SHALL require an active or trialing subscription before invoking a metered LLM call: organizations without one SHALL be refused the AI analysis and SHALL NOT accrue billed credit consumption.

#### Scenario: Subscriptionless org is refused metered analysis

- **WHEN** an inbound message arrives for an organization with no active subscription
- **THEN** the AI analysis SHALL be refused
- **AND** no usage SHALL be recorded against the AI usage ledger

#### Scenario: Active org analysis is metered as before

- **WHEN** an inbound message arrives for an organization with an active or trialing subscription
- **THEN** the metered analysis SHALL run
- **AND** usage SHALL be recorded idempotently per the existing ledger rules

### Requirement: Billing provider AI meter ingestion

The system SHALL ingest AI consumption to the billing provider as a best-effort background meter event, and SHALL apply provider meter-grant events for the AI meter to the local credit allowance.

#### Scenario: Meter event ingested after recording

- **WHEN** AI usage is recorded successfully
- **THEN** the system SHALL asynchronously (best-effort, logged on failure) call `BillingProvider.IngestMeterEvent` with the organization's external customer ID, meter slug `ai.tokens.consumed`, and the credits consumed

#### Scenario: MercadoPago organizations keep local ledger

- **WHEN** the organization's billing provider is MercadoPago (whose adapter logs a no-op for meter events)
- **THEN** usage SHALL remain recorded in the local ledger and the no-op SHALL be logged without error

#### Scenario: AI meter grant refreshes allowance

- **WHEN** a `meter.grant.created`/`meter.grant.updated` webhook arrives with meter slug `ai.tokens`
- **THEN** the system SHALL update the organization's `ai_credits_max` allowance with the granted credits

#### Scenario: Unrelated meter grants are ignored

- **WHEN** a meter grant webhook arrives for a slug other than `ai.tokens` (or the invoice meter)
- **THEN** the system SHALL ignore it for the AI allowance

### Requirement: AI usage endpoint

The system SHALL expose the organization's current-period AI usage through an authenticated API endpoint.

#### Scenario: Usage returned for authenticated organization

- **WHEN** an authenticated organization requests `GET /api/crm/usage/ai`
- **THEN** the response SHALL include `tokens_input`, `tokens_output`, `tokens_embedding`, `credits_used`, `credits_max`, `credits_remaining`, `period_start`, and `period_end`

#### Scenario: No usage yet

- **WHEN** an organization has never consumed AI tokens
- **THEN** the endpoint SHALL return zeroed totals with the current period bounds
