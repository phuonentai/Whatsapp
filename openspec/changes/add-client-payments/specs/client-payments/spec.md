## ADDED Requirements

### Requirement: One-shot payment link creation
The system SHALL create one-shot, customer-facing MercadoPago payment preferences (payment links) for a deal/invoice amount through the payments module. The link SHALL be created with the MercadoPago Checkout Preferences API (`POST /checkout/preferences`), SHALL include the platform commission as a markup on the payment amount, and SHALL reference the originating deal for tracking.

#### Scenario: Create payment link for invoice
- **WHEN** the invoicing module requests a payment link for an invoice with a positive amount
- **THEN** the system SHALL create a MercadoPago payment preference priced at amount plus platform commission
- **AND** SHALL persist a `client_payments` record with status `pending`, the preference id, and the deal reference
- **AND** SHALL return the checkout link (`init_point`) for inclusion in the WhatsApp invoice notification

#### Scenario: Payment link creation failure
- **WHEN** MercadoPago fails to create the payment preference
- **THEN** the system SHALL NOT fail the invoicing flow
- **AND** SHALL record the failure and send the invoice notification without a payment link

#### Scenario: Payment link request without deal context
- **WHEN** a payment link is requested for a deal that cannot be resolved
- **THEN** the system SHALL reject the request without persisting state

### Requirement: Payment state tracking
The system SHALL track the lifecycle of each client payment in the local `client_payments` table with states `pending`, `paid`, `failed`, and `expired`, keyed by MercadoPago payment id and preference id, and referenced to the organization (Stytch org foreign key) and deal. Local storage SHALL contain only MercadoPago payment/preference identifiers — never tokens, card data, or wallet credentials.

#### Scenario: Payment marked paid
- **WHEN** a verified payment approval is processed
- **THEN** the system SHALL transition the payment to `paid`
- **AND** SHALL record the payment timestamp and MercadoPago payment id

#### Scenario: Payment marked failed
- **WHEN** MercadoPago reports the payment as rejected or cancelled
- **THEN** the system SHALL transition the payment to `failed`

#### Scenario: Duplicate payment event
- **WHEN** the same payment id is processed again
- **THEN** the system SHALL apply the transition at most once (idempotent, transaction-isolated)

### Requirement: Payment event webhook dispatch
The system SHALL dispatch verified MercadoPago payment events (`payment_created`, `payment_updated`, `payment_approved`) to the client-payments module, where payment approval SHALL be verified against the MercadoPago Payments API (`GET /v1/payments/{id}`) before any state mutation. Subscription events SHALL continue to be handled by the billing service unchanged.

#### Scenario: Payment approved event dispatched
- **WHEN** a verified `payment_approved` event references a tracked client payment
- **THEN** the system SHALL verify the payment status via the MercadoPago Payments API
- **AND** SHALL transition the payment to `paid` on confirmation

#### Scenario: Payment verification fails
- **WHEN** the verification call to the MercadoPago Payments API fails or returns a non-approved status
- **THEN** the system SHALL NOT transition the payment
- **AND** SHALL leave the payment `pending` for retry or polling

#### Scenario: Payment event for untracked payment
- **WHEN** a verified payment event does not match any tracked client payment
- **THEN** the system SHALL acknowledge the event without persisting state

### Requirement: WhatsApp payment confirmation
The system SHALL notify the deal's contact inside WhatsApp when a tracked client payment becomes `paid`, reusing the existing outbound send path, and SHALL record a deal activity entry. The confirmation is transactional (required to fulfill the sale) and SHALL be sent even to contacts with `consent_status = 'withdrawn'`, SHALL NOT contain promotional content, and SHALL NOT fail the payment flow if sending fails (logged warning only).

#### Scenario: Payment confirmed in WhatsApp
- **WHEN** a tracked client payment transitions to `paid`
- **THEN** the system SHALL send a WhatsApp confirmation message to the deal's contact via the existing send path
- **AND** SHALL record a deal activity entry for the payment

#### Scenario: Confirmation send failure
- **WHEN** the WhatsApp confirmation send fails
- **THEN** the system SHALL log the failure
- **AND** SHALL keep the payment `paid` (send failure SHALL NOT revert or fail the payment)

### Requirement: Platform commission
The system SHALL apply a platform commission as a percentage markup on client payment amounts, sourced from backend configuration, with the commission amount recorded on the persisted payment record. The customer pays amount plus commission; the full original amount SHALL be recorded as the SME receivable.

#### Scenario: Commission applied to payment amount
- **WHEN** a payment link is created
- **THEN** the system SHALL price the MercadoPago preference at amount × (1 + commission rate)
- **AND** SHALL persist both the base amount and the commission amount

#### Scenario: Zero commission configured
- **WHEN** the commission rate is zero
- **THEN** the system SHALL create the payment preference at exactly the base amount
