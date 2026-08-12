## ADDED Requirements

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
