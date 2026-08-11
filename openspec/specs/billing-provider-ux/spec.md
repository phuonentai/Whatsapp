# Spec: billing-provider-ux

## Purpose

TBD. Define how billing surfaces (subscription tab, paywall) behave and render copy based on the enabled billing provider (MercadoPago or Polar).

## Requirements

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
The system SHALL render provider-appropriate copy on billing surfaces (plans modal, subscription paywall, subscription tab) using the typed copy layer in Spanish-first voice, replacing the current mixed English/Spanish hardcoded strings. The plan price caption SHALL name the active provider instead of defaulting to "Billed via Polar", and the paywall inactive state SHALL show MercadoPago (PSE/Nequi) messaging when MercadoPago is enabled and Polar messaging otherwise. Payment method explanations (Polar international card, MercadoPago PSE / Nequi / Colombian card) SHALL be expressed in plain Spanish.

#### Scenario: MP enabled shows MP copy
- **WHEN** MercadoPago is enabled and a user views the subscription tab or an inactive paywall
- **THEN** the copy identifies MercadoPago as the billing provider

#### Scenario: Polar default shows Polar copy
- **WHEN** MercadoPago is not enabled and a user views the subscription tab or an inactive paywall
- **THEN** the copy identifies Polar as the billing provider

#### Scenario: Plans modal renders Spanish copy

- **WHEN** a user opens the plans modal
- **THEN** the modal heading, payment-method explanation, plan descriptions, and action labels SHALL be Spanish strings resolved from the copy layer

#### Scenario: Active-subscription notice renders Spanish

- **WHEN** the plans modal shows the active-subscription blocking notice
- **THEN** the notice SHALL be the Spanish string ("Suscripción activa" with an instruction to cancel the current subscription before switching)

### Requirement: Plan-switch blocking uses custom dialog, not window.alert

When an active subscription prevents switching plans, the billing UI SHALL render the blocking notice in the custom dialog component (or an inline banner) and SHALL NOT use `window.alert`.

#### Scenario: Active subscription blocks plan switch with dialog

- **WHEN** a user with an active subscription attempts to switch plans
- **THEN** the blocking notice SHALL render in the custom dialog component
- **AND** no `window.alert` SHALL be invoked
