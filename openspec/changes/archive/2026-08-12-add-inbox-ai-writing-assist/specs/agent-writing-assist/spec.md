## ADDED Requirements

### Requirement: Composer writing assist

The system SHALL provide an AI text-transformation endpoint for outbound drafts with modes `rephrase`, `formal`, `casual`, and `summarize`. Requests SHALL require a valid authenticated session with at least `org:view` permission and an org context. Every call SHALL be org-scoped and metered through the existing LLM token ledger.

The system SHALL gate every call on AI credits: when the org has a credit cap and no credits remain, the endpoint SHALL return 402 with a machine-readable code; ledger-read failures SHALL fail open (proceed) exactly like the existing agent analysis path.

The endpoint SHALL transform only the text the user submitted. It MUST NOT send messages, mutate conversations, or persist drafts. The returned text is a suggestion for the user to review.

#### Scenario: Rephrase a draft

- **WHEN** an authenticated user submits `{ "text": "hola cuando llega mi pedido", "mode": "rephrase" }`
- **THEN** the system returns 200 with `{"data": {"text": "<rewritten draft>"}, "success": true}`

#### Scenario: Tone change

- **WHEN** an authenticated user submits `{ "text": "llega hoy", "mode": "formal" }` or `"casual"`
- **THEN** the system returns 200 with the text transformed to the requested tone

#### Scenario: Summarize

- **WHEN** an authenticated user submits `{ "text": "<long draft>", "mode": "summarize" }`
- **THEN** the system returns 200 with a condensed version of the draft

#### Scenario: Invalid mode

- **WHEN** an authenticated user submits an unknown `mode`
- **THEN** the system returns 400 with a Spanish error message and no LLM call is made

#### Scenario: Unauthenticated request

- **WHEN** an unauthenticated request hits the rephrase endpoint
- **THEN** the system returns 401

#### Scenario: Credits exhausted

- **WHEN** an authenticated user submits a rephrase request and the org has no remaining AI credits
- **THEN** the system returns 402 with a machine-readable `ai_credits_exhausted` code and no LLM call is made

#### Scenario: Composer applies the result

- **WHEN** the frontend receives a successful rephrase response
- **THEN** the composer replaces the current draft text with the returned text
- **AND** the user reviews it before sending
- **AND** nothing is sent automatically

#### Scenario: Composer failure leaves draft intact

- **WHEN** the rephrase request fails (network, LLM, or 402)
- **THEN** the composer keeps the current draft text unchanged
- **AND** shows a Spanish error toast
