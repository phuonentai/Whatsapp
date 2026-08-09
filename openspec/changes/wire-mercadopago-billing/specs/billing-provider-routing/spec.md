## MODIFIED Requirements

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

#### Scenario: Router is the DI binding
- **WHEN** the dependency container resolves `domain.BillingProvider`
- **THEN** it SHALL return the `ProviderRouter` wired with the `PolarAdapter`, the `MPAdapter`, and the provider resolver
- **AND** the `BillingService` SHALL receive the `ProviderRouter` as its `domain.BillingProvider`

### Requirement: Organization provider preference storage
The system SHALL store each organization's billing provider preference in the database and expose it for lookup during request processing. The provider resolver SHALL read `organizations.billing_provider` from the local PostgreSQL database via the SQLC queries `GetOrganizationBillingProvider` and `SetOrganizationBillingProvider`, treating a NULL value as `"polar"`.

#### Scenario: Default provider for existing orgs
- **WHEN** an organization does not have a `billing_provider` value set
- **THEN** the resolver SHALL return `"polar"` as the default provider

#### Scenario: Set provider on first MercadoPago checkout
- **WHEN** an organization completes their first MercadoPago checkout successfully
- **THEN** the system SHALL set the organization's `billing_provider` to `"mercadopago"`

#### Scenario: Provider lookup during request
- **WHEN** a billing operation is invoked for an organization
- **THEN** the `ProviderRouter` SHALL look up the organization's `billing_provider` preference via the resolver
- **AND** route to the corresponding adapter

#### Scenario: Resolver reflects stored preference
- **WHEN** an organization's `billing_provider` is set to `"mercadopago"` in the database
- **THEN** the resolver SHALL return `"mercadopago"` (never a hardcoded value)
