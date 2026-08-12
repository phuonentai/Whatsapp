## ADDED Requirements

### Requirement: Conversation context generation in the agent pipeline

The agent pipeline SHALL provide conversation-context generation as a sibling to analysis: a domain `ConversationContextService` that reads org-scoped conversation history, applies consent gating and PII masking, runs a metered LLM generation, and persists to `agent.conversation_contexts`. Consent semantics SHALL mirror the existing agent consent state machine (`none`/`requested`/`granted`/`withdrawn`); contacts without granted consent (when required) receive structural-only context.

#### Scenario: Context service gates on consent

- **WHEN** a context generation is requested for a conversation whose contact has not granted consent and the org requires consent
- **THEN** the service SHALL produce structural-only context with `consent_gated: true`
- **AND** SHALL NOT invoke the analysis LLM

#### Scenario: Context service masks PII

- **WHEN** a context generation runs an LLM call
- **THEN** the message bodies SHALL be masked per `whatsapp-compliance` before the call
