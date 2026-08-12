## ADDED Requirements

### Requirement: Optional trial seeding on signup
When trial is enabled by configuration, organization bootstrap SHALL create a local subscription row with status `trialing` and current period `[now, now + TRIAL_DAYS]`, plus a quota-tracking row granting NO quota (invoice processing blocked), so the new organization passes the paywall for the trial window. Seeding SHALL be idempotent and SHALL flow through a narrow, injected interface rather than coupling the organizations module to the billing repositories.

#### Scenario: Trial enabled seeds trialing row
- **WHEN** `TRIAL_ENABLED=true` and bootstrap completes
- **THEN** a subscription row with status `trialing` and `current_period_end = now + TRIAL_DAYS` SHALL exist
- **AND** the organization SHALL pass `RequireActiveSubscription` (trialing evaluates as active)
- **AND** a quota-tracking row SHALL exist with `invoice_count = 0` so invoice processing SHALL be blocked during the trial (no quota granted)

#### Scenario: Trial disabled leaves free tier
- **WHEN** `TRIAL_ENABLED` is unset or false
- **THEN** bootstrap SHALL create no subscription row
- **AND** the organization SHALL resolve to status `none` (free tier)

#### Scenario: Bootstrap retry does not duplicate trial rows
- **WHEN** bootstrap is retried for the same organization after a partial failure
- **THEN** no duplicate `subscriptions` or `quota_tracking` rows SHALL be created
- **AND** an existing provider subscription SHALL NOT be overwritten or reset by trial seeding (provider upserts remain authoritative)

### Requirement: Trial expiry is enforced at the status boundary
A `trialing` subscription whose `current_period_end` has passed SHALL be classified as expired (inactive) by the status path, so the paywall lazy guard can evaluate it against the provider before denying access. Local-only trials SHALL fail closed; provider-backed trials SHALL heal via provider refresh.

#### Scenario: Local-only trial expires
- **WHEN** a local-only trial's `current_period_end` passes and the organization accesses a paywalled route
- **THEN** the status path SHALL classify the subscription as expired (inactive)
- **AND** the lazy guard SHALL run and, finding no provider subscription, SHALL fail closed
- **AND** the response SHALL be HTTP 402 (`subscription_required` after classification, never indefinite access)

#### Scenario: Provider trial auto-converts after expiry
- **WHEN** a provider-backed trial's `current_period_end` passes but the provider reports the subscription active (auto-converted)
- **THEN** the lazy guard SHALL refresh from the provider and grant access
- **AND** the status SHALL be treated as active

#### Scenario: Refresh never calls the provider for synthetic trial rows
- **WHEN** the refresh path encounters a local-only trial row (synthetic `local-trial-*` identifiers)
- **THEN** the system SHALL NOT call the payment provider with the synthetic customer ID
- **AND** SHALL fail closed to the no-subscription classification

### Requirement: Late-payment remediation paths
The system SHALL give `past_due`/`unpaid` organizations a payment-method-update path distinct from resubscribing, and SHALL keep read features available (grace) while blocking writes. The payment-method-update path SHALL be provider-appropriate and SHALL NOT silently route a dunning organization into creating a new subscription.

#### Scenario: Past-due paywall response points to a real page
- **WHEN** a `past_due` organization hits a paywalled route
- **THEN** the 402 response SHALL include an `upgrade_url` that resolves to a real page
- **AND** the message SHALL instruct updating the payment method

#### Scenario: Grace keeps features readable
- **WHEN** an organization is `past_due`
- **THEN** the entitlement SHALL mark `IsGracePeriod=true`
- **AND** read features SHALL remain available while writes are blocked

#### Scenario: Polar payment-method update uses the customer portal
- **WHEN** a `past_due`/`unpaid` Polar organization is offered a payment-method update
- **THEN** the offer SHALL use the Polar customer portal URL when the subscription exposes one
- **AND** SHALL fall back to the plans modal only when no portal URL is available

#### Scenario: Mercado Pago payment-method update is honest and documented
- **WHEN** a `past_due`/`unpaid` Mercado Pago organization is offered remediation
- **THEN** the UI SHALL NOT present a new-checkout CTA as a payment-method update
- **AND** SHALL state the in-app limitation explicitly (resubscribe is labeled as creating a new subscription; provider auto-retry is surfaced), with the limitation documented for a follow-up

### Requirement: No-subscription orgs receive the subscription_required code
The billing status service SHALL classify quota-status lookup outcomes: a missing subscription row SHALL surface as the existing `domain.ErrSubscriptionNotFound` sentinel returned **unwrapped**, and all other lookup errors SHALL propagate. The paywall middleware SHALL map the no-subscription sentinel to 402 `subscription_required` and any other error to HTTP 500.

#### Scenario: No subscription returns subscription_required
- **WHEN** an organization with no subscription row accesses a paywalled route
- **THEN** the response SHALL be HTTP 402 with error `subscription_required` and status `none`

#### Scenario: Database failure is not a 402
- **WHEN** the quota-status lookup fails due to a database error
- **THEN** the response SHALL be HTTP 500
- **AND** SHALL NOT be a 402 with a subscription reason

#### Scenario: The billing status endpoint still returns none for no-subscription orgs
- **WHEN** an organization with no subscription row calls `GET /api/subscriptions/status`
- **THEN** the response SHALL be HTTP 200 with `none` semantics (the existing `ErrSubscriptionNotFound` branch of the handler)
- **AND** SHALL NOT regress to 500 after the service classification change

#### Scenario: The lazy-guard refresh path follows the same classification
- **WHEN** the provider refresh path encounters a missing subscription
- **THEN** the refresh SHALL classify the outcome with the same sentinel semantics
- **AND** SHALL NOT fabricate a misleading nil-error `none` status

### Requirement: Onboarding surfaces render upgrade prompts on 402
Frontend onboarding surfaces (WhatsApp config section, first-run checklist) SHALL recognize a 402 response and render an upgrade prompt — a CTA that opens the plans modal or the billing subscription view — instead of displaying the raw 402 error text.

#### Scenario: WhatsApp config 402 shows upgrade CTA
- **WHEN** a new organization without a subscription opens the WhatsApp configuration section
- **THEN** the section SHALL render an upgrade prompt with a path to the plans modal or subscription view
- **AND** SHALL NOT render the raw "active subscription is required" error as the primary surface

#### Scenario: First-run checklist does not dead-end on paywalled step
- **WHEN** a new organization's first-run checklist leads to a paywalled step (connect WhatsApp)
- **THEN** the checklist SHALL surface the plan/pricing choice before the paywalled step
- **AND** the user SHALL NOT be stranded on a raw 402

### Requirement: Trial-expiry population is observable
The system SHALL expose queries for trialing subscriptions whose period has ended: a global monitoring variant and an organization-scoped variant (tenant-safe — at most one row per org, since `organization_id` is unique).

#### Scenario: Expired trials are listable globally and per-org
- **WHEN** a monitoring or follow-up scanner queries expired trials
- **THEN** a query SHALL return subscriptions with `status = 'trialing'` and `current_period_end < now()` across organizations (monitoring)
- **AND** an organization-scoped variant SHALL return the expired trial for a single organization when used per-tenant
