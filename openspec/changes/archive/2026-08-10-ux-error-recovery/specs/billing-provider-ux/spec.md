# Delta Spec: billing-provider-ux — ux-error-recovery

## ADDED Requirements

### Requirement: Plan-switch blocking uses custom dialog, not window.alert

When an active subscription prevents switching plans, the billing UI SHALL render the blocking notice in the custom dialog component (or an inline banner) and SHALL NOT use `window.alert`.

#### Scenario: Active subscription blocks plan switch with dialog

- **WHEN** a user with an active subscription attempts to switch plans
- **THEN** the blocking notice SHALL render in the custom dialog component
- **AND** no `window.alert` SHALL be invoked
