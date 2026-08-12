## Purpose

Defines the integrity invariants of the subscription/quota status layer: unknown provider statuses stay distinct and refreshable, a present subscription is never masked by a missing quota row, quota updates preserve consumed counts, and invoice quota decrements are bounded.

## Requirements

### Requirement: Unknown subscription statuses remain distinct and refreshable

The system SHALL NOT collapse unrecognized subscription statuses to `none`: an unmapped status SHALL resolve to a distinct inactive state that preserves the raw status in the reason, and the paywall lazy guard SHALL attempt a provider refresh before denying.

#### Scenario: Unknown status triggers refresh

- **WHEN** a stored subscription has a status the system does not recognize (e.g. Polar `revoked`, MP `pending`, raw `paused`)
- **THEN** the paywall SHALL attempt `RefreshSubscriptionStatus` before denying
- **AND** the 402 SHALL NOT report status `none`

#### Scenario: Provider refresh heals an unknown status

- **WHEN** the provider refresh reports the subscription active
- **THEN** access SHALL be granted
- **AND** the stored status SHALL be updated to the mapped status

### Requirement: A present subscription is never masked by a missing quota row

The quota-status read SHALL return the subscription's real status even when its quota row is absent, and the subscription write path SHALL seed a missing quota row.

#### Scenario: Subscription row without quota row reads its status

- **WHEN** a subscription row exists but its quota row is missing
- **THEN** the status read SHALL return the subscription's actual status (not `none`)
- **AND** a quota row SHALL be seeded with zeroed counts

### Requirement: Quota updates preserve consumed counts

The quota upsert SHALL NOT overwrite `invoice_count` (or `ai_credits_max`) when the incoming update carries no new value, so webhook-driven metadata updates cannot inflate consumed quotas.

#### Scenario: Metadata-only update preserves count

- **WHEN** a webhook triggers a quota upsert without a new count
- **THEN** the stored `invoice_count` SHALL remain unchanged

### Requirement: Invoice quota decrements are bounded

`DecrementInvoiceCount` SHALL only decrement when the stored count is positive and SHALL surface a quota-exhausted error otherwise.

#### Scenario: Concurrent consumption cannot go negative

- **WHEN** two concurrent invoice consumptions race at `invoice_count = 1`
- **THEN** exactly one SHALL succeed
- **AND** the count SHALL NOT go below zero
