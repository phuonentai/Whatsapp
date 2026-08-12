## ADDED Requirements

### Requirement: MercadoPago payment verification uses the MercadoPago provider
The system SHALL verify MercadoPago checkouts against the MercadoPago adapter end-to-end: poll the MP payments API, fetch the subscription from the MercadoPago preapproval search, and persist the mapped status and quota.

#### Scenario: Verify resolves subscription from MercadoPago
- **WHEN** `VerifyMPPayment` runs for an approved MP payment
- **THEN** the subscription SHALL be fetched from the MercadoPago provider, never the Polar adapter
- **AND** the stored status SHALL be the mapped domain status

### Requirement: MercadoPago subscription carries an invoice quota
The system SHALL persist a per-plan invoice quota for MP subscriptions, attached to the preapproval at checkout time as `metadata.invoice_count_max` and read back on verification, sync, and webhook processing.

#### Scenario: Checkout attaches plan quota
- **WHEN** `CreateMPCheckout` creates a preapproval for a plan with a configured invoice count
- **THEN** the preapproval SHALL carry `metadata.invoice_count_max` equal to the plan's configured quota

#### Scenario: Verified subscription seeds quota
- **WHEN** `VerifyMPPayment` persists an approved MP subscription
- **THEN** the local quota tracking SHALL be seeded with the preapproval's `invoice_count_max`
- **AND** `CanProcessInvoices` SHALL be true when the quota is positive and the status is active

### Requirement: MercadoPago checkout and cancellation resolve the organization context
The system SHALL resolve the organization id for MP checkout and cancellation from the Gin context keys populated by the auth middleware (`RequireOrganization`), never from `ctx.Value` on the raw request context.

#### Scenario: Checkout resolves org from Gin context
- **WHEN** `CreateMPCheckout` runs for an authenticated organization
- **THEN** the org id SHALL be read from the Gin context
- **AND** the checkout SHALL NOT fail with a configuration error for a missing request-context value

#### Scenario: Cancellation resolves org and persists locally
- **WHEN** `CancelMPSubscription` runs for an MP organization
- **THEN** the org id SHALL be read from the Gin context
- **AND** a local subscription row with status `canceled` and valid `CurrentPeriodStart/End` SHALL be persisted
- **AND** the organization SHALL be treated as inactive thereafter

### Requirement: MercadoPago cancellation is end-to-end persisted
The system SHALL persist MP cancellation locally with non-null period bounds, deriving them from the existing subscription row or the preapproval's schedule when the provider does not return them.

#### Scenario: Canceled MP org loses access locally
- **WHEN** an MP subscription is canceled and the webhook or verify path later queries the local status
- **THEN** the local status SHALL be `canceled`
- **AND** the paywall SHALL deny access

#### Scenario: Subscription-cancelled webhook without dates is idempotent
- **WHEN** an MP `subscription_cancelled` event arrives without `end_date`/`next_payment_date`
- **THEN** the event SHALL be processed without an error
- **AND** the local row SHALL retain its prior period bounds
