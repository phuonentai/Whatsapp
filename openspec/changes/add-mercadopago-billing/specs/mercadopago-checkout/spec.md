## ADDED Requirements

### Requirement: MercadoPago checkout endpoint
The system SHALL provide a `POST /api/subscriptions/create-mp-checkout` endpoint that creates a MercadoPago preference and returns a redirect URL for the hosted Checkout Pro flow.

#### Scenario: Create checkout preference
- **WHEN** an authenticated user with `org:manage` permission calls the endpoint with a valid plan ID
- **THEN** the system SHALL create a MercadoPago preapproval with the plan's `auto_recurring` settings
- **AND** set `external_reference` to the Stytch organization ID
- **AND** set `back_url` to the application's verify-payment page
- **AND** return the `init_point` URL for the MercadoPago hosted checkout

#### Scenario: Checkout requires valid plan
- **WHEN** the endpoint is called with an invalid or non-existent plan ID
- **THEN** the system SHALL return HTTP 400 with an error message

#### Scenario: Checkout requires active session
- **WHEN** the endpoint is called without a valid Stytch session
- **THEN** the system SHALL return HTTP 401

### Requirement: MercadoPago payment verification endpoint
The system SHALL provide a `POST /api/subscriptions/verify-mp-payment` endpoint that verifies a MercadoPago payment after checkout redirect.

#### Scenario: Verify successful payment
- **WHEN** the endpoint is called with a valid MercadoPago `payment_id`
- **THEN** the system SHALL poll the MercadoPago API for payment status
- **AND** on `approved` status, create/update the local subscription and quota records
- **AND** set the organization's `billing_provider` to `"mercadopago"`
- **AND** return `BillingStatus` with `HasActiveSubscription: true`

#### Scenario: Payment still pending
- **WHEN** the payment status is `pending` or `in_process` after the polling window
- **THEN** the system SHALL return `BillingStatus` with `HasActiveSubscription: false` and reason `"payment_pending"`

#### Scenario: Payment failed or rejected
- **WHEN** the payment status is `rejected` or `cancelled`
- **THEN** the system SHALL NOT create any subscription records
- **AND** return `BillingStatus` with `HasActiveSubscription: false` and reason `"payment_failed"`

### Requirement: Dual-provider checkout UI
The frontend SHALL display both payment provider options in the plan selection modal, allowing users to choose between international cards (Polar) and Colombian payment methods (MercadoPago).

#### Scenario: Provider selection in plans modal
- **WHEN** a user views the plans/pricing modal
- **THEN** each plan SHALL show two checkout options:
  - "International credit/debit card" (triggers Polar checkout)
  - "PSE / Nequi / Colombian card" (triggers MercadoPago checkout)

#### Scenario: MercadoPago checkout flow
- **WHEN** user selects "PSE / Nequi / Colombian card" option
- **THEN** the frontend SHALL call `createMercadoPagoCheckout(planId)` server action
- **AND** redirect the browser to the MercadoPago Checkout Pro `init_point` URL

#### Scenario: Post-payment redirect and verification
- **WHEN** the user returns from MercadoPago checkout to the application's callback URL
- **THEN** the frontend SHALL call `verifyMercadoPagoPayment(paymentId)` server action
- **AND** on success, redirect to the dashboard
- **AND** on failure, display an appropriate error message
