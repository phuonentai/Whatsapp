## ADDED Requirements

### Requirement: Quote send transition triggers document delivery

The system SHALL, as part of the `borrador → enviada` transition, trigger document delivery: render the branded PDF, store it as a file asset, host a shareable link, and send the link to the deal's contact via WhatsApp (as specified by the `quote-delivery` capability). If rendering or storage fails, the transition SHALL fail with an error and the quote SHALL remain `borrador`; if only the WhatsApp send fails, the transition SHALL succeed and the failure SHALL be logged.

#### Scenario: Send transition renders and delivers
- **WHEN** a `borrador` quote transitions to `enviada`
- **THEN** the system SHALL render + store the PDF and send the shareable link to the contact
- **AND** the quote status SHALL become `enviada`

#### Scenario: Render failure keeps quote in borrador
- **WHEN** the renderer fails during the transition
- **THEN** the transition SHALL fail
- **AND** the quote SHALL remain `borrador`
- **AND** an error SHALL be returned to the caller
