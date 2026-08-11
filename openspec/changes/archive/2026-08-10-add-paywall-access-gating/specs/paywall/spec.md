# Paywall (capability spec)

## Purpose

Defines commercial access gating: subscription status derivation, the 402 Payment Required gate, lazy-guard recovery for missed provider webhooks, and non-blocking status exposure. Implemented in `go-b2b-starter/internal/modules/paywall`.

## ADDED Requirements

### Requirement: Subscription status resolves to active or inactive

The system SHALL expose a provider-agnostic `SubscriptionStatus` for an organization. The statuses `active` and `trialing` SHALL evaluate as active (`IsActive=true`); the statuses `past_due`, `canceled`, `unpaid`, and `none` SHALL evaluate as inactive (`IsActive=false`).

#### Scenario: Active subscription grants access

- **WHEN** an organization has a subscription with status `active`
- **THEN** `SubscriptionStatus.IsActive` SHALL be true

#### Scenario: Trialing subscription grants access

- **WHEN** an organization has a subscription with status `trialing`
- **THEN** `SubscriptionStatus.IsActive` SHALL be true

#### Scenario: Past-due subscription denies access

- **WHEN** an organization has a subscription with status `past_due`
- **THEN** `SubscriptionStatus.IsActive` SHALL be false

#### Scenario: Canceled subscription denies access

- **WHEN** an organization has a subscription with status `canceled`
- **THEN** `SubscriptionStatus.IsActive` SHALL be false

#### Scenario: Unpaid subscription denies access

- **WHEN** an organization has a subscription with status `unpaid`
- **THEN** `SubscriptionStatus.IsActive` SHALL be false

#### Scenario: No subscription denies access

- **WHEN** an organization has no subscription
- **THEN** the status SHALL be `none` and `IsActive` SHALL be false

### Requirement: Blocking middleware returns 402 on inactive subscription

The system SHALL provide `RequireActiveSubscription` middleware that requires an organization context, returns HTTP 402 Payment Required when the organization has no active subscription, and stores the status in the Gin context when access is granted. It SHALL run after the auth and organization-context middleware, and SHALL skip OPTIONS preflight requests.

#### Scenario: Missing organization context returns 500

- **WHEN** a request reaches `RequireActiveSubscription` without an organization ID in context
- **THEN** the middleware SHALL respond with HTTP 500 and error `configuration_error`
- **AND** SHALL abort the request

#### Scenario: No subscription returns 402 subscription_required

- **WHEN** an organization with no subscription accesses a paywalled route
- **THEN** the middleware SHALL respond with HTTP 402 and error `subscription_required`
- **AND** the response SHALL carry status `none`
- **AND** SHALL abort the request

#### Scenario: Active subscription proceeds

- **WHEN** an organization with an active subscription accesses a paywalled route
- **THEN** the middleware SHALL store the `SubscriptionStatus` in the Gin context
- **AND** SHALL pass the request to the handler

#### Scenario: OPTIONS preflight is skipped

- **WHEN** a request uses the OPTIONS method
- **THEN** the middleware SHALL pass without evaluating subscription status

### Requirement: Lazy guard refreshes status on boundary

When the locally stored subscription status is inactive but not `none`, the system SHALL call `RefreshSubscriptionStatus` to sync with the payment provider before denying access. If the provider reports the subscription as active, access SHALL be granted and the status SHALL be treated as active.

#### Scenario: Missed webhook heals via provider refresh

- **WHEN** local DB says the subscription is inactive (`past_due`, `canceled`, `unpaid`, or `trialing` boundary state) but not `none`, and the provider reports it active
- **THEN** the middleware SHALL grant access
- **AND** SHALL treat the refreshed status as active

#### Scenario: Provider confirms inactive

- **WHEN** local DB says the subscription is inactive but not `none`, and the provider also reports it inactive (or refresh fails)
- **THEN** the middleware SHALL respond with HTTP 402
- **AND** SHALL abort the request

### Requirement: 402 responses carry status-specific error codes

The system SHALL return HTTP 402 with a status-specific error code and message when subscription is inactive. The response body SHALL include `error`, `message`, an optional `upgrade_url` (defaulting to `/billing`), and the subscription `status`.

#### Scenario: Past-due returns payment_failed

- **WHEN** an organization with a `past_due` subscription accesses a paywalled route
- **THEN** the middleware SHALL respond with HTTP 402 and error `payment_failed`
- **AND** the message SHALL indicate the payment failed and the method needs updating

#### Scenario: Canceled returns subscription_canceled

- **WHEN** an organization with a `canceled` subscription accesses a paywalled route
- **THEN** the middleware SHALL respond with HTTP 402 and error `subscription_canceled`
- **AND** the message SHALL instruct resubscribing to continue

#### Scenario: Unpaid returns payment_required

- **WHEN** an organization with an `unpaid` subscription accesses a paywalled route
- **THEN** the middleware SHALL respond with HTTP 402 and error `payment_required`
- **AND** the message SHALL instruct updating the payment method

#### Scenario: Generic inactive returns subscription_inactive

- **WHEN** an organization has an inactive subscription with no specific reason
- **THEN** the middleware SHALL respond with HTTP 402 and error `subscription_inactive`
- **AND** SHALL use the status reason as the message when one is present

### Requirement: Optional middleware never blocks

The system SHALL provide `OptionalSubscriptionStatus` middleware that stores the subscription status in the Gin context when available and always continues to the next handler without blocking.

#### Scenario: Status available and stored

- **WHEN** an organization has a subscription and `OptionalSubscriptionStatus` runs
- **THEN** the middleware SHALL store the status in the Gin context
- **AND** SHALL pass the request to the handler

#### Scenario: No status, request proceeds

- **WHEN** an organization has no subscription (or the status lookup fails)
- **THEN** the middleware SHALL pass the request to the handler without storing a status

### Requirement: Status provider reads DB locally and refreshes from provider

The system SHALL expose a `SubscriptionStatusProvider` with `GetSubscriptionStatus` reading from the local database only (no external API calls during request handling) and `RefreshSubscriptionStatus` forcing a sync from the payment provider. The blocking middleware SHALL be applied to protected routes; billing portal, settings, profile, and webhook routes SHALL NOT be gated.

#### Scenario: Fast path reads local DB only

- **WHEN** `GetSubscriptionStatus` is called for an organization
- **THEN** the status SHALL be read from the local database
- **AND** no external provider API SHALL be called

#### Scenario: Refresh path syncs with provider

- **WHEN** `RefreshSubscriptionStatus` is called for an organization
- **THEN** the system SHALL sync the subscription status with the payment provider API

#### Scenario: Billing and settings routes are not gated

- **WHEN** a request targets the billing portal, settings, profile, or a webhook
- **THEN** the paywall blocking middleware SHALL NOT block the request

### Requirement: Named middlewares are registered for route use

The system SHALL register the paywall middlewares with the server under the names `paywall` (blocking) and `paywall_optional` (non-blocking), with legacy aliases `subscription` and `subscription_optional` resolving to the same behavior.

#### Scenario: Blocking middleware registered

- **WHEN** the server registers middleware named `paywall` or `subscription`
- **THEN** it SHALL resolve to `RequireActiveSubscription`

#### Scenario: Optional middleware registered

- **WHEN** the server registers middleware named `paywall_optional` or `subscription_optional`
- **THEN** it SHALL resolve to `OptionalSubscriptionStatus`
