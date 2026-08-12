## ADDED Requirements

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
