## ADDED Requirements

### Requirement: Subscription webhook processing preserves module state
The system SHALL NOT mutate per-org module state when a subscription webhook carries no product metadata or the metadata key set is unchanged: `SyncModulesFromMetadata` SHALL run only on an actual key-set change, and an absent metadata key set SHALL be treated as "no change" rather than "disable all modules".

#### Scenario: Webhook without metadata keeps modules
- **WHEN** a subscription webhook upserts a subscription without product metadata
- **THEN** the organization's enabled modules SHALL be unchanged
- **AND** `SyncModulesFromMetadata` SHALL NOT run with an empty key list

#### Scenario: Metadata persists on the subscription row
- **WHEN** a subscription webhook carries product metadata
- **THEN** the upserted subscription row SHALL store that metadata
- **AND** entitlements derived from subscription metadata SHALL reflect it without requiring a verify/refresh

#### Scenario: Real key-set change reconciles modules
- **WHEN** a subscription event carries a product metadata key set that differs from the stored set
- **THEN** `SyncModulesFromMetadata` SHALL reconcile modules against the new key set
