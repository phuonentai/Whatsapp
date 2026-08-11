## ADDED Requirements

### Requirement: Conversation context endpoint

The system SHALL expose `GET /api/agent/conversations/:id/context` returning AI-derived context for a conversation: a summary, a detected intent/topic, key facts, a generation cursor, and generation status. The endpoint SHALL sit behind the existing agent middleware (`auth` + `org_context` + `subscription`) and SHALL require `org:view` to read. Context generation SHALL be org-scoped and SHALL only reference the requesting organization's conversation.

#### Scenario: Context returned for a conversation

- **WHEN** a member with `org:view` calls `GET /api/agent/conversations/:id/context` for an org-scoped conversation
- **THEN** the system SHALL return the conversation id, summary, detected intent, key facts, source cursor, generation timestamp, consent-gated flag, and status
- **AND** SHALL NOT expose context of conversations belonging to other organizations

#### Scenario: Unauthorized read denied

- **WHEN** a member without `org:view` calls the context endpoint
- **THEN** the system SHALL return HTTP 403 with a Spanish error message

### Requirement: Metered context generation

Context generation SHALL run through the existing metered LLM client so every generation is recorded in the AI usage ledger. When the organization's AI credits are exhausted, the endpoint SHALL return context status `unavailable` and SHALL NOT perform an unmetered generation.

#### Scenario: Generation records AI consumption

- **WHEN** a context LLM call completes with the organization id in context
- **THEN** the consumed tokens SHALL be recorded in the ai-usage ledger via the metered client

#### Scenario: Exhausted credits return unavailable

- **WHEN** the organization's AI credits are exhausted before context generation
- **THEN** the endpoint SHALL return status `unavailable`
- **AND** SHALL NOT call the LLM unmetered

### Requirement: Context caching and regeneration

The system SHALL persist generated context in `agent.conversation_contexts` keyed by `conversation_id` (FK to `crm.conversations(id)` ON DELETE CASCADE), with summary, key facts, detected intent, source cursor, consent-gated flag, and timestamps. The endpoint SHALL serve cached context when the cursor matches the conversation's latest message and the cache is fresh (TTL 5 minutes); otherwise SHALL regenerate and upsert idempotently.

#### Scenario: Fresh cache served

- **WHEN** cached context exists whose source cursor equals the latest message id and is within TTL
- **THEN** the endpoint SHALL return the cached context without an LLM call

#### Scenario: Stale cache regenerated

- **WHEN** the conversation has messages newer than the cached source cursor or the cache is outside TTL
- **THEN** the system SHALL regenerate context and upsert the row keyed by `conversation_id`

### Requirement: Consent-gated context (Ley 1581)

When the organization requires consent (`consent_required`) and the contact has not granted it, the endpoint SHALL return structural-only context (message count, first and last message dates, channel) with `consent_gated: true` and SHALL NOT run LLM analysis on the conversation. If a contact withdraws consent after context was generated, the read SHALL return masked/structural-only context.

#### Scenario: Consent required and not granted

- **WHEN** the org requires consent and the contact's consent status is not `granted`
- **THEN** the endpoint SHALL return structural-only context with `consent_gated: true`
- **AND** SHALL NOT perform LLM analysis

#### Scenario: Consent withdrawn after generation

- **WHEN** a contact's consent status is `withdrawn` and a cached context row exists
- **THEN** the endpoint SHALL return structural-only context and SHALL NOT expose previously generated facts

### Requirement: PII masking before generation

Message bodies passed to the context LLM SHALL be masked per the existing PII masking path before any LLM call.

#### Scenario: Masked input to LLM

- **WHEN** context generation prepares message bodies for the LLM
- **THEN** PII (phones, names, emails, document numbers) SHALL be masked before the call per `whatsapp-compliance`
