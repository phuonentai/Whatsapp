## ADDED Requirements

### Requirement: AI chat streaming over existing endpoint

The system SHALL support token-streaming responses on the existing `POST /example_cognitive/chat` endpoint. A request that signals streaming (via `Accept: text/event-stream` header or a `stream: true` request field) SHALL receive an SSE response streaming content tokens incrementally; a request without the signal SHALL receive the existing single-JSON response. Both modes SHALL traverse the identical middleware chain (`auth`, `org_context`, `subscription`, the `ai_assistant` feature gate, and the AI credit guard).

#### Scenario: Streaming chat request

- **WHEN** an authorized request is sent to `POST /example_cognitive/chat` with streaming signaled
- **THEN** the response SHALL use `Content-Type: text/event-stream`
- **AND** the response SHALL emit a `data:` SSE event per generated token chunk
- **AND** the response SHALL emit a final `data: {"done": true, ...}` event carrying `session_id` and `message_id`
- **AND** the assistant message SHALL be persisted with the full final content and total tokens used

#### Scenario: Non-streaming chat request unchanged

- **WHEN** an authorized request is sent to `POST /example_cognitive/chat` without the streaming signal
- **THEN** the response SHALL be the existing single JSON body
- **AND** no SSE framing SHALL be used

### Requirement: Metered token integrity during streaming

The system SHALL record AI usage tokens for streamed responses through the same metered LLM client path used for non-streaming completions. Token recording SHALL happen only after a successful stream completes; a stream that fails mid-way SHALL record nothing.

#### Scenario: Successful stream records usage

- **WHEN** a streamed response completes successfully
- **THEN** the total consumed tokens SHALL be recorded in the org's ai-usage ledger via the metered client
- **AND** the persisted assistant message SHALL carry the total `TokensUsed`

#### Scenario: Failed stream records nothing

- **WHEN** the underlying LLM stream errors before completion
- **THEN** the system SHALL emit an SSE `event: error` frame
- **AND** SHALL NOT record token usage for the incomplete stream

### Requirement: Credit guard rejects before stream start

The system SHALL evaluate the AI credit guard before opening the SSE stream. If the org's period credits are exhausted, the request SHALL be rejected with HTTP 402 and the `ai_credits_exhausted` error shape, with no stream opened.

#### Scenario: Exhausted credits rejected before streaming

- **WHEN** an org's period credits are exhausted and a streaming chat request arrives
- **THEN** the system SHALL return HTTP 402
- **AND** SHALL NOT open an SSE stream or call the LLM provider

### Requirement: Frontend streams chat responses

The frontend SHALL consume the streaming endpoint and render tokens incrementally, replacing the single-response mutation for the chat UI. The consumer SHALL fall back to the non-streaming JSON response when streaming is unavailable.

#### Scenario: Chat renders tokens incrementally

- **WHEN** a user sends a message in the Knowledge Base chat and the backend streams
- **THEN** the UI SHALL append token chunks to the assistant message as they arrive
- **AND** the UI SHALL render the final message once the `done` event arrives

#### Scenario: Streaming unavailable falls back

- **WHEN** the streaming request fails or returns a non-SSE response
- **THEN** the UI SHALL render the non-streaming JSON response
- **AND** the chat session history SHALL remain consistent
