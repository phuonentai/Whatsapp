## ADDED Requirements

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
