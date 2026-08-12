## MODIFIED Requirements

### Requirement: 402 responses carry status-specific error codes
The system SHALL return HTTP 402 with a status-specific error code and message when subscription is inactive. The response body SHALL include `error`, `message`, an optional `upgrade_url` (defaulting to `/dashboard/settings?view=subscription`), and the subscription `status`.

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

#### Scenario: No subscription returns subscription_required
- **WHEN** an organization has no subscription row at all
- **THEN** the middleware SHALL respond with HTTP 402 and error `subscription_required`
- **AND** the response SHALL carry status `none`

#### Scenario: Status lookup failure returns 500
- **WHEN** the billing status service cannot read the quota/subscription state due to a database error
- **THEN** the middleware SHALL respond with HTTP 500
- **AND** SHALL NOT respond with a 402 subscription error
