## ADDED Requirements

### Requirement: Per-provider webhook endpoints in Go
The system SHALL provide webhook ingress endpoints in the Go backend for both billing providers, registered under `/api/v1/webhooks/`: `POST /api/v1/webhooks/polar` and `POST /api/v1/webhooks/mercadopago`. Both endpoints SHALL be signature-only (no session or organization context required) and SHALL be the sole authority for webhook-triggered database mutations.

#### Scenario: Polar webhook endpoint receives events
- **WHEN** a Polar.sh webhook is delivered to `POST /api/v1/webhooks/polar` with a valid signature
- **THEN** the system SHALL process the event and persist the resulting subscription state

#### Scenario: MercadoPago webhook endpoint receives events
- **WHEN** a MercadoPago IPN notification is delivered to `POST /api/v1/webhooks/mercadopago` with a valid signature
- **THEN** the system SHALL process the event and persist the resulting subscription state

#### Scenario: Next.js webhook route retired
- **WHEN** the Go webhook endpoints are live and the Polar dashboard is re-pointed
- **THEN** the Next.js `/api/billing/webhook` route SHALL no longer receive or process billing webhooks

### Requirement: Polar webhook signature verification
The system SHALL verify Polar.sh webhook signatures in Go before processing. Verification SHALL use the Svix-style scheme: the `webhook-id`, `webhook-timestamp`, and `webhook-signature` headers, an HMAC-SHA256 of the signing input `msg_id.msg_timestamp.payload` keyed with the webhook secret, constant-time comparison, and a timestamp tolerance window.

#### Scenario: Valid Polar signature
- **WHEN** a request to `POST /api/v1/webhooks/polar` carries headers and body that verify against the `POLAR_WEBHOOK_SECRET`
- **THEN** the system SHALL accept and process the webhook

#### Scenario: Invalid or missing Polar signature
- **WHEN** a request to `POST /api/v1/webhooks/polar` has a missing, malformed, or mismatched signature
- **THEN** the system SHALL reject the request with HTTP 401
- **AND** SHALL NOT mutate any database state

#### Scenario: Expired timestamp rejected
- **WHEN** a request's `webhook-timestamp` falls outside the tolerance window
- **THEN** the system SHALL reject the request with HTTP 401

### Requirement: MercadoPago webhook signature verification
The system SHALL verify MercadoPago IPN signatures in Go before processing, using HMAC-SHA256 of the request body keyed with `MERCADOPAGO_WEBHOOK_SECRET` against the `x-signature` header, with constant-time comparison.

#### Scenario: Valid MercadoPago signature
- **WHEN** a request to `POST /api/v1/webhooks/mercadopago` carries a valid `x-signature`
- **THEN** the system SHALL accept and process the webhook

#### Scenario: Invalid or missing MercadoPago signature
- **WHEN** a request to `POST /api/v1/webhooks/mercadopago` has a missing or invalid `x-signature`
- **THEN** the system SHALL reject the request with HTTP 401
- **AND** SHALL NOT mutate any database state

### Requirement: Webhook dispatch to existing services
The system SHALL dispatch verified webhooks to the existing billing services without routing through the provider router: Polar events to `ProcessWebhookEvent` and MercadoPago events to `ProcessMPWebhookEvent`. MercadoPago `subscription_authorized` and `subscription_updated` events SHALL upsert the subscription, `subscription_cancelled` SHALL mark it canceled, and payment events SHALL be ignored (checkout verification is handled by the `verify-mp-payment` polling endpoint).

#### Scenario: Polar subscription events persisted
- **WHEN** a verified `subscription.created` or `subscription.updated` event is received
- **THEN** the system SHALL upsert the subscription record via `ProcessWebhookEvent`

#### Scenario: MercadoPago subscription authorized persisted
- **WHEN** a verified `subscription_authorized` event is received
- **THEN** the system SHALL upsert the subscription record with the mapped status

#### Scenario: MercadoPago subscription cancelled persisted
- **WHEN** a verified `subscription_cancelled` event is received
- **THEN** the system SHALL mark the subscription as `"canceled"`

#### Scenario: MercadoPago payment events ignored
- **WHEN** a verified `payment_created`, `payment_updated`, or `payment_approved` event is received
- **THEN** the system SHALL acknowledge the event without persisting state
- **AND** checkout verification SHALL continue via the `verify-mp-payment` polling endpoint

#### Scenario: Webhooks bypass provider router
- **WHEN** any webhook event is processed
- **THEN** it SHALL be dispatched directly to the provider-specific service
- **AND** SHALL NOT be routed through `ProviderRouter`
