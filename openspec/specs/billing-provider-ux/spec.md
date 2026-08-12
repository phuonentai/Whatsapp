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

### Requirement: Subscription state resolution is provider-aware
The frontend SHALL resolve subscription state for MercadoPago organizations from the backend status endpoint: when MP is enabled and the Polar SDK reports no active subscription, `resolveCurrentSubscription` SHALL consult `GET /api/subscriptions/status` before declaring the organization inactive.

#### Scenario: MP org with active preapproval resolves active
- **WHEN** a MercadoPago organization's Polar lookup finds no subscription and the backend reports `has_active_subscription=true`
- **THEN** the resolved subscription state SHALL be `isActive=true`
- **AND** the paywall and alerts SHALL reflect an active plan

#### Scenario: Backend reports past due
- **WHEN** the backend status reports an inactive subscription with reason `past_due`
- **THEN** the resolved state SHALL carry `status=past_due` and `reason=NO_ACTIVE_SUBSCRIPTION` so the existing dunning alert renders

#### Scenario: Billing not configured for MP
- **WHEN** MP is enabled but the checkout plan id is unset
- **THEN** the state SHALL carry reason `MP_UNCONFIGURED`

### Requirement: Late-payment remediation paths surface a payment-method update
The frontend SHALL present `past_due`/`unpaid` states with a payment-method-update path distinct from subscribing anew.

#### Scenario: Past-due alert offers update path
- **WHEN** a user with a `past_due` subscription views the dashboard
- **THEN** the billing alert SHALL offer a payment-method update action in addition to plan browsing

### Requirement: Layout-level plans modal renders the MercadoPago option
The plans modal opened from the dashboard layout ("Subscribe now" entry point) SHALL receive `mercadopagoEnabled` and SHALL render the MercadoPago checkout option when enabled.

#### Scenario: MP-enabled layout modal shows MP option
- **WHEN** MercadoPago is enabled and the dashboard layout opens its plans modal
- **THEN** the modal SHALL render the MercadoPago checkout option
- **AND** the modal SHALL NOT present Polar as the only CTA

#### Scenario: MP-only deployment promotes MP CTA
- **WHEN** MercadoPago is enabled and Polar is unconfigured
- **THEN** the MercadoPago option SHALL be the primary checkout CTA

### Requirement: Cancellation and resume branch by provider with accurate copy
The subscription tab SHALL branch cancellation and resume by the enabled provider: under MP, resume SHALL use the MP resume path and the cancellation dialog SHALL state that cancellation is immediate; under Polar, resume and end-of-period cancellation copy SHALL keep the existing behavior.

#### Scenario: MP resume uses MP path
- **WHEN** MercadoPago is enabled and a member resumes a pending cancellation
- **THEN** the resume action SHALL call the MercadoPago resume path, never Polar's `cancelSubscription`

#### Scenario: MP cancellation copy is immediate
- **WHEN** MercadoPago is enabled and a member opens the cancellation dialog
- **THEN** the dialog SHALL state access ends immediately on cancellation

### Requirement: MercadoPago checkout returns to the application origin
The MP checkout action SHALL supply a return URL rooted at the application origin so the `back_url` never falls back to the `localhost` default in production.

#### Scenario: Checkout carries app-origin return URL
- **WHEN** a member triggers MercadoPago checkout from a deployed environment
- **THEN** the backend `back_url` SHALL point at the application origin (not `http://localhost:3000`)
- **AND** the returning user SHALL land on the subscription view

### Requirement: Dunning remediation branches by billing provider
For `past_due`/`unpaid` states, the system SHALL offer a payment-method-update path distinct from subscribing anew, branched by the enabled billing provider, and SHALL NOT present a new-checkout CTA as a payment-method update for either provider.

#### Scenario: Polar dunning offers customer-portal payment update
- **WHEN** a `past_due`/`unpaid` Polar organization is shown the dunning alert
- **THEN** the alert SHALL offer an "Update payment method" action using the Polar customer portal URL when the subscription exposes one
- **AND** SHALL fall back to the plans modal only when no portal URL is available

#### Scenario: Mercado Pago dunning states the in-app limitation honestly
- **WHEN** a `past_due`/`unpaid` Mercado Pago organization is shown the dunning alert
- **THEN** the alert SHALL NOT present a new-checkout CTA as a payment-method update
- **AND** SHALL state that in-app payment-method update is not yet available for Mercado Pago
- **AND** SHALL surface provider auto-retry messaging and label any resubscribe action as creating a new subscription

#### Scenario: Dunning alert renders without dead links
- **WHEN** a paywalled organization receives a 402 with `upgrade_url`
- **THEN** the alert SHALL resolve `upgrade_url` to a real page
- **AND** SHALL NOT route the user to a route that does not exist

### Requirement: Grace-period state is surfaced on billing surfaces
The system SHALL surface grace-period state (`IsGracePeriod`) in the settings subscription tab with provider-appropriate copy, while reads remain available and writes stay blocked.

#### Scenario: Past-due member sees grace copy
- **WHEN** a member with a `past_due` subscription opens the settings subscription tab
- **THEN** the tab SHALL show grace-period copy ("your plan is in a grace period; features stay readable, writes blocked")
- **AND** SHALL offer the provider-appropriate payment-method-update path

