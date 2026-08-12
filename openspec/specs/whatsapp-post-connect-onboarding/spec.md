# whatsapp-post-connect-onboarding Specification

## Purpose
TBD - created by archiving change post-connect-whatsapp-onboarding. Update Purpose after archive.
## Requirements
### Requirement: Post-connect next-steps flow

After a successful WhatsApp connect, the WhatsApp configuration surface SHALL render a next-steps card guiding the user through: sending a test message, understanding that only new messages arrive going forward, acknowledging Ley 1581 consent expectations, opening the inbox, and enabling the WhatsApp assistant. The card SHALL be dismissible, SHALL reappear if the configuration is deactivated and later reactivated, and SHALL link to the inbox, compliance, and agent-settings surfaces.

#### Scenario: Next-steps card renders after connect

- **WHEN** a WhatsApp configuration is active and the next-steps card has not been dismissed
- **THEN** the surface SHALL render the next-steps card with the five guidance items and their links

#### Scenario: Test-message guidance uses the inbound path

- **WHEN** the next-steps card renders the test-message item
- **THEN** the copy SHALL instruct the user to message the business number from their own phone so the message arrives in the inbox via the existing webhook pipeline
- **AND** the system SHALL NOT invoke a dedicated test-message send endpoint

#### Scenario: Go-forward expectations are explicit

- **WHEN** the next-steps card renders
- **THEN** the copy SHALL state that only new messages arriving from the moment of connection are received
- **AND** SHALL NOT imply historical chat import

#### Scenario: Consent expectations link to compliance

- **WHEN** the next-steps card renders the consent item
- **THEN** the copy SHALL summarize that contacts may receive an automatic data-treatment consent request (Ley 1581) and SHALL link to the compliance section

#### Scenario: Card reappears after reactivation

- **WHEN** a configuration is deactivated and later reactivated
- **THEN** the next-steps card SHALL render again even if previously dismissed

