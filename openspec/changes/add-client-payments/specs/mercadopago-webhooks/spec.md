## MODIFIED Requirements

### Requirement: Webhook dispatch to existing services
The system SHALL dispatch verified webhooks to the existing billing services without routing through the provider router: Polar events to `ProcessWebhookEvent` and MercadoPago events to `ProcessMPWebhookEvent`. MercadoPago `subscription_authorized` and `subscription_updated` events SHALL upsert the subscription and `subscription_cancelled` SHALL mark it canceled. MercadoPago payment events (`payment_created`, `payment_updated`, `payment_approved`) SHALL be dispatched to the client-payments module, where approval SHALL be verified against the MercadoPago Payments API before persisting payment state; the subscription `verify-mp-payment` polling endpoint remains the fallback for checkout verification.

#### Scenario: Polar subscription events persisted
- **WHEN** a verified `subscription.created` or `subscription.updated` event is received
- **THEN** the system SHALL upsert the subscription record via `ProcessWebhookEvent`

#### Scenario: MercadoPago subscription authorized persisted
- **WHEN** a verified `subscription_authorized` event is received
- **THEN** the system SHALL upsert the subscription record with the mapped status

#### Scenario: MercadoPago subscription cancelled persisted
- **WHEN** a verified `subscription_cancelled` event is received
- **THEN** the system SHALL mark the subscription as `"canceled"`

#### Scenario: MercadoPago payment events dispatched to client payments
- **WHEN** a verified `payment_created`, `payment_updated`, or `payment_approved` event is received
- **THEN** the system SHALL dispatch the event to the client-payments module
- **AND** SHALL NOT mutate subscription state
- **AND** subscription checkout verification SHALL continue via the `verify-mp-payment` polling endpoint

#### Scenario: Webhooks bypass provider router
- **WHEN** any webhook event is processed
- **THEN** it SHALL be dispatched directly to the provider-specific service
- **AND** SHALL NOT be routed through `ProviderRouter`
