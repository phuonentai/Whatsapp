## ADDED Requirements

### Requirement: MercadoPago checkout uses the configured MP plan id
The MP checkout action SHALL prefer the public `NEXT_PUBLIC_MERCADOPAGO_CHECKOUT_PLAN_ID` over the Polar plan id passed from the plans modal.

#### Scenario: Checkout with env plan id
- **WHEN** a user triggers MercadoPago checkout and `NEXT_PUBLIC_MERCADOPAGO_CHECKOUT_PLAN_ID` is set
- **THEN** the backend SHALL receive that env plan id as `plan_id`

### Requirement: Checkout callback handles preapproval id
The dashboard callback SHALL handle MercadoPago returns carrying `preapproval_id` without a `payment_id` by routing to the subscription view without an error banner.

#### Scenario: Preapproval-only return
- **WHEN** the MP redirect includes `preapproval_id` but no `payment_id`
- **THEN** the user SHALL be redirected to `/dashboard/settings?view=subscription` without `payment_error`

### Requirement: Checkout result parameters are acknowledged once
The subscription tab SHALL clear `payment_verified`/`payment_error` from the URL after the banner renders, so refresh and back-navigation do not re-show a stale banner.

#### Scenario: Banner acknowledged without dismissal
- **WHEN** the subscription tab loads with `payment_verified=true` and the banner renders
- **THEN** the parameter SHALL be removed from the URL without requiring the dismiss click

### Requirement: Subscription state refreshes after checkout and on focus
The subscription query SHALL refetch on window focus and after a checkout callback so webhook/verify-driven status changes appear without a manual refresh.

#### Scenario: Status updates on window focus
- **WHEN** a user pays in another tab and returns to an open subscription tab
- **THEN** the tab SHALL refetch and reflect the active status

### Requirement: String-typed numeric product metadata is coerced
The plans modal and checkout SHALL coerce string-typed numeric product metadata (`included_seats`, `included_invoices`, `ai_credits_max`) so configured limits render instead of "—".

#### Scenario: String metadata renders as number
- **WHEN** a plan's product metadata stores `"500"` as a string
- **THEN** the plan card and comparison SHALL render 500
- **AND** the checkout SHALL forward the numeric value
