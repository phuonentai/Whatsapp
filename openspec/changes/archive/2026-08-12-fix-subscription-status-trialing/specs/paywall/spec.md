## ADDED Requirements

### Requirement: Trialing subscription processes invoices within quota
The system SHALL treat a `trialing` subscription as active for quota enforcement: `GetBillingStatus` SHALL report `HasActiveSubscription=true`, and the database `can_process_invoice` gate SHALL be true when a trialing subscription has remaining invoice quota.

#### Scenario: Trialing org with quota processes invoices
- **WHEN** an organization has a subscription with status `trialing` and `invoice_count > 0`
- **THEN** `GetBillingStatus` SHALL return `HasActiveSubscription=true`
- **AND** `CanProcessInvoices` SHALL be true

#### Scenario: Trialing org passes the paywall
- **WHEN** an organization with a `trialing` subscription accesses a paywalled route
- **THEN** `RequireActiveSubscription` SHALL store the status and pass the request without a 402

#### Scenario: Trialing org with exhausted quota is quota-blocked, not paywall-blocked
- **WHEN** an organization has a `trialing` subscription with `invoice_count = 0`
- **THEN** quota enforcement SHALL deny invoice processing with the quota error
- **AND** the paywall SHALL still evaluate the subscription as active
