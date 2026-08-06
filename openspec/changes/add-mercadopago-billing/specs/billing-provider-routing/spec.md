## ADDED Requirements

### Requirement: Provider router implements BillingProvider
The system SHALL provide a `ProviderRouter` struct that implements `domain.BillingProvider` and delegates to the correct adapter (Polar or MercadoPago) based on the organization's configured billing provider.

#### Scenario: Route to Polar adapter
- **WHEN** an organization has `billing_provider` set to `"polar"` (or unset/default)
- **THEN** all `BillingProvider` method calls SHALL delegate to the `PolarAdapter`

#### Scenario: Route to MercadoPago adapter
- **WHEN** an organization has `billing_provider` set to `"mercadopago"`
- **THEN** all `BillingProvider` method calls SHALL delegate to the `MPAdapter`

#### Scenario: Unknown provider
- **WHEN** an organization has an unrecognized `billing_provider` value
- **THEN** the router SHALL return an error indicating an unsupported billing provider

### Requirement: Organization provider preference storage
The system SHALL store each organization's billing provider preference in the database and expose it for lookup during request processing.

#### Scenario: Default provider for existing orgs
- **WHEN** an organization does not have a `billing_provider` value set
- **THEN** the system SHALL treat it as `"polar"`

#### Scenario: Set provider on first MercadoPago checkout
- **WHEN** an organization completes their first MercadoPago checkout successfully
- **THEN** the system SHALL set the organization's `billing_provider` to `"mercadopago"`

#### Scenario: Provider lookup during request
- **WHEN** a billing operation is invoked for an organization
- **THEN** the `ProviderRouter` SHALL look up the organization's `billing_provider` preference
- **AND** route to the corresponding adapter

### Requirement: No breaking changes to existing Polar integration
The system SHALL preserve all existing Polar.sh functionality unchanged. The `PolarAdapter` and all Polar-specific code SHALL continue to work without modification.

#### Scenario: Polar subscriptions continue working
- **WHEN** an organization uses Polar.sh as their billing provider
- **THEN** all existing Polar checkout, webhook, and subscription flows SHALL function identically to before this change

#### Scenario: Polar webhooks unaffected
- **WHEN** a Polar.sh webhook is received
- **THEN** the webhook SHALL be processed by the existing Polar webhook handler without routing through the provider router
