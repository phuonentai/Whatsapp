## ADDED Requirements

### Requirement: Template-driven branded quote PDF rendering

The system SHALL render a quote as a branded PDF through a `DocumentRenderer` interface using a named template (`cotizacion`) from a `TemplateRegistry`. Rendering SHALL resolve the organization's branding via `DocumentBrandingProvider` using the quote's branding snapshot key, and SHALL produce PDF bytes containing: the org logo, letterhead/colors, quote number/date/validity, client identification, itemized lines with quantities/prices/discounts, subtotal/IVA/total, and terms footer. Rendering SHALL support Spanish text (embedded Unicode-capable fonts).

#### Scenario: Quote rendered to branded PDF
- **WHEN** the renderer renders an `enviada` quote for an organization with branding configured
- **THEN** the PDF bytes SHALL contain the org logo, branded header, all line items, and correct totals
- **AND** SHALL reflect the branding active at the quote's snapshot key

#### Scenario: Rendering without branding falls back to defaults
- **WHEN** the renderer renders a quote for an organization without branding
- **THEN** the PDF SHALL render with default styling and the org company name/NIT

#### Scenario: Render failure does not corrupt quote state
- **WHEN** PDF rendering fails
- **THEN** the system SHALL log the failure
- **AND** the quote status SHALL remain unchanged (transition aborts with an error)

### Requirement: Quote PDF stored as file asset with shareable link

The system SHALL store rendered quote PDFs via the file-asset manager with a quote purpose/category and SHALL expose a shareable link usable by the client without org authentication (unguessable token; may expire with the quote's validity). The link SHALL be retrievable by `org:view` members.

#### Scenario: Rendered PDF hosted with link
- **WHEN** a quote PDF is rendered and stored
- **THEN** the file asset SHALL be linked to the quote
- **AND** a shareable link SHALL be available for the client

#### Scenario: Shareable link respects quote validity
- **WHEN** a shareable link is generated for a quote with `valid_until`
- **THEN** the link SHALL stop working after `valid_until` passes (or the view SHALL indicate the quote is expired)

### Requirement: WhatsApp delivery of quote link on send

The system SHALL, when a quote transitions to `enviada`, render and host the quote PDF and send a WhatsApp message with the shareable link to the deal's contact through the existing outbound send path, at most once per transition (notified-status guard). Send failure SHALL be logged as a warning and SHALL NOT revert the quote state; the link SHALL be re-sendable from the deal page.

#### Scenario: Sending a quote notifies the client
- **WHEN** a member transitions a quote to `enviada` and rendering/storage succeed
- **THEN** the system SHALL send a WhatsApp message with the quote link to the contact
- **AND** SHALL mark the quote as notified for the transition

#### Scenario: Repeat send transition does not re-notify
- **WHEN** the same `enviada` transition event is processed again
- **THEN** the system SHALL NOT send a duplicate message

#### Scenario: Send failure is non-fatal
- **WHEN** the WhatsApp send fails after the quote is `enviada`
- **THEN** the quote status SHALL remain `enviada`
- **AND** the failure SHALL be logged
- **AND** a member SHALL be able to re-send the link from the deal page

### Requirement: Document renderer/sender seams for future templates

The system SHALL implement `DocumentRenderer` and `DocumentSender` as domain interfaces such that future document templates (e.g., invoice templates) and future media-based sending can be added by registering new templates/senders without modifying the quotes domain service.

#### Scenario: New template registers without domain change
- **WHEN** a future change adds a new template to the registry (e.g., `cuenta_cobro`)
- **THEN** the quotes domain service SHALL remain unchanged
- **AND** the new template SHALL render through the same `DocumentRenderer` interface

#### Scenario: Media sender swaps in via DI
- **WHEN** a future change provides a document-type WhatsApp sender
- **THEN** it SHALL be wired via dependency injection in place of the link sender
- **AND** the quotes service SHALL require no code change
