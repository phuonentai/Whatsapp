## ADDED Requirements

### Requirement: Plan comparison across plans

The plans modal SHALL render a comparison of plans alongside the plan cards, rowing up included seats, included invoice quotas, and AI credits per plan so a user can compare quantities at a glance. Values SHALL be sourced from the fetched Polar products (`includedSeats`, `includedInvoices`, metadata `ai_credits_max`); missing values SHALL render as "—" rather than a fabricated number.

#### Scenario: Comparison renders from product data

- **WHEN** a user opens the plans modal with multiple plans
- **THEN** the comparison SHALL display rows for seats, invoices, and AI credits across all plans using product-derived values

#### Scenario: Missing metadata renders placeholder

- **WHEN** a plan lacks an AI-credit or quantity value
- **THEN** the comparison SHALL render "—" for that cell

### Requirement: Billing-interval toggle

The plans modal SHALL offer a monthly/annual billing-interval toggle when the fetched catalog contains both intervals for the plan family. The toggle SHALL filter displayed plans to the selected interval. When only one interval exists, the toggle SHALL NOT render.

#### Scenario: Toggle shown with both intervals

- **WHEN** the catalog contains both monthly and annual plans
- **THEN** the modal SHALL render a monthly/annual toggle and display plans matching the selected interval

#### Scenario: No toggle for single interval

- **WHEN** the catalog contains only one interval
- **THEN** the modal SHALL NOT render the interval toggle

### Requirement: AI credits are a visible plan line item

The plans modal SHALL display each plan's AI credits per period as a first-class line item on the plan card and in the comparison, using the same "credits per period" language as the subscription-tab AI meter.

#### Scenario: AI credit line rendered

- **WHEN** a plan card or the comparison renders
- **THEN** the AI-credits figure from `metadata.ai_credits_max` SHALL be shown per period

### Requirement: Post-checkout outcome feedback

The subscription tab SHALL acknowledge checkout results delivered via the `payment_verified` and `payment_error` query parameters with a success or error banner, and SHALL clear the parameter after acknowledgement so the banner is not repeated.

#### Scenario: Verified payment shows success

- **WHEN** the subscription tab loads with `payment_verified=true`
- **THEN** the tab SHALL render a success banner confirming the plan is active
- **AND** SHALL remove the parameter from the URL after acknowledgement

#### Scenario: Failed payment shows error with retry

- **WHEN** the subscription tab loads with `payment_error=true`
- **THEN** the tab SHALL render an error banner with a link to open the plans modal to retry
- **AND** SHALL remove the parameter from the URL after acknowledgement
