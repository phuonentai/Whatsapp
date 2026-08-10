## ADDED Requirements

### Requirement: Subscription cancellation branches by billing provider
The subscription tab SHALL branch the cancellation flow by the enabled billing provider: when MercadoPago is enabled (`NEXT_PUBLIC_MERCADOPAGO_PLAN_ID` present), cancellation SHALL call the MercadoPago cancel server action posting to `POST /api/subscriptions/mp-cancel`; otherwise cancellation SHALL keep the existing Polar `cancelSubscription` flow.

#### Scenario: MercadoPago enabled cancels via MP route
- **WHEN** MercadoPago is enabled and a member with subscription-management permission cancels the subscription
- **THEN** the system calls the MercadoPago cancel server action and the backend route `/api/subscriptions/mp-cancel` receives the request

#### Scenario: Polar default keeps existing cancel flow
- **WHEN** MercadoPago is not enabled and a member cancels the subscription
- **THEN** the system keeps the existing Polar `cancelSubscription` flow unchanged

#### Scenario: Cancel without permission is rejected
- **WHEN** a member without subscription-management permission attempts to cancel
- **THEN** the cancel server action rejects the request before any outbound call, per Stytch B2B RBAC

### Requirement: Billing surfaces render provider-appropriate copy
The subscription tab and paywall SHALL render provider-appropriate billing copy: the plan price caption SHALL name the active provider instead of defaulting to "Billed via Polar", and the paywall inactive state SHALL show MercadoPago (PSE/Nequi) messaging when MercadoPago is enabled and Polar messaging otherwise.

#### Scenario: MP enabled shows MP copy
- **WHEN** MercadoPago is enabled and a user views the subscription tab or an inactive paywall
- **THEN** the copy identifies MercadoPago as the billing provider

#### Scenario: Polar default shows Polar copy
- **WHEN** MercadoPago is not enabled and a user views the subscription tab or an inactive paywall
- **THEN** the copy identifies Polar as the billing provider
