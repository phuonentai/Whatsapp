## ADDED Requirements

### Requirement: MercadoPago HTTP client
The system SHALL provide an HTTP client for the MercadoPago API at `internal/platform/mercadopago/` that authenticates via Bearer token and supports GET, POST, and PUT operations against `https://api.mercadopago.com`.

#### Scenario: Authenticated API request
- **WHEN** a MercadoPago API request is made with a valid `MERCADOPAGO_ACCESS_TOKEN`
- **THEN** the request SHALL include an `Authorization: Bearer <token>` header

#### Scenario: Missing credentials
- **WHEN** `MERCADOPAGO_ACCESS_TOKEN` is not set
- **THEN** the client factory SHALL return an error during DI initialization

### Requirement: MercadoPago configuration
The system SHALL load MercadoPago configuration from environment variables including `MERCADOPAGO_ACCESS_TOKEN`, `MERCADOPAGO_BASE_URL` (default `https://api.mercadopago.com`), and `MERCADOPAGO_WEBHOOK_SECRET`.

#### Scenario: Config loaded from environment
- **WHEN** the application starts
- **THEN** MercadoPago config SHALL be populated from environment variables via viper

### Requirement: MercadoPago subscription adapter
The system SHALL provide an `MPAdapter` struct that implements `domain.BillingProvider` and translates MercadoPago subscription API calls to domain types.

#### Scenario: Get subscription by external reference
- **WHEN** `GetSubscription(ctx, externalCustomerID)` is called
- **THEN** the adapter SHALL call `GET /preapproval/search?external_reference=<externalCustomerID>`
- **AND** map the response to `domain.Subscription` with correct status, period dates, and product info

#### Scenario: Subscription not found
- **WHEN** the `/preapproval/search` response has zero results
- **THEN** the adapter SHALL return `domain.ErrSubscriptionNotFound`

#### Scenario: Get checkout session (payment verification)
- **WHEN** `GetCheckoutSession(ctx, paymentID)` is called
- **THEN** the adapter SHALL call `GET /v1/payments/{id}`
- **AND** map the payment status to `domain.CheckoutSessionResponse` where `approved` maps to `"succeeded"`

#### Scenario: Checkout session polling
- **WHEN** `GetCheckoutSessionWithPolling(ctx, paymentID)` is called
- **THEN** the adapter SHALL poll `GET /v1/payments/{id}` every 2 seconds for up to 10 seconds
- **AND** return immediately when status is `approved`
- **AND** return error on timeout or non-retryable status

#### Scenario: Ingest meter event (no-op)
- **WHEN** `IngestMeterEvent(ctx, externalCustomerID, meterSlug, amount)` is called
- **THEN** the adapter SHALL return nil without making any API call
- **AND** log a debug message noting the no-op

### Requirement: MercadoPago webhook parsing
The system SHALL parse MercadoPago IPN webhook payloads and normalize them to `domain.SubscriptionEventData` for processing by existing service handlers.

#### Scenario: Subscription authorized webhook
- **WHEN** a webhook with topic `subscription_authorized` is received
- **THEN** the parser SHALL extract `preapproval_id`, `external_reference`, `status`, `auto_recurring` dates
- **AND** produce a `domain.SubscriptionEventData` with status mapped to a valid subscription status

#### Scenario: Subscription cancelled webhook
- **WHEN** a webhook with topic `subscription_cancelled` is received
- **THEN** the parser SHALL extract the subscription ID and external reference
- **AND** produce a `domain.SubscriptionEventData` with status `"canceled"`

#### Scenario: Payment created webhook
- **WHEN** a webhook with topic `payment_created` is received
- **THEN** the parser SHALL extract `payment_id`, `external_reference`, and `status`
- **AND** produce a `domain.SubscriptionEventData` suitable for checkout verification

#### Scenario: Webhook signature verification
- **WHEN** a MercadoPago webhook is received
- **THEN** the system SHALL verify the `x-signature` header against the webhook secret using HMAC-SHA256
- **AND** reject requests with invalid or missing signatures with HTTP 401

### Requirement: MercadoPago DI wiring
The system SHALL register MercadoPago dependencies in the uber-go/dig container, including the platform client, `MPAdapter`, and webhook parser.

#### Scenario: DI container registration
- **WHEN** `mercadopago.Init(container)` is called
- **THEN** the container SHALL provide `mercadopago.Config`, `mercadopago.Client`, and `MPAdapter` (as `domain.BillingProvider`)
