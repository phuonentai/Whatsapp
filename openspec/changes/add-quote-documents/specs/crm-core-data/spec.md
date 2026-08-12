## ADDED Requirements

### Requirement: Deal activity on quote lifecycle

The system SHALL record deal activities (type `sistema`, Spanish messages) on quote lifecycle events: created, sent, approved, rejected, expired, revised. Activities SHALL reference the acting member when available.

#### Scenario: Quote events leave activity trail
- **WHEN** a quote is created, sent, approved, rejected, expired, or revised
- **THEN** the deal SHALL have a corresponding `sistema` activity in Spanish
- **AND** the activity SHALL reference the acting `stytch_member_id` when the action was member-initiated
