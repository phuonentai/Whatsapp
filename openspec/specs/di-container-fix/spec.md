## Purpose

Defines the DI container hardening requirements for the backend: duplicate-provider detection, error propagation from dependency registration, and full legacy store registration before the paywall module initializes.

## Requirements

### Requirement: Duplicate Provide detection

The DI container MUST tolerate only one registration of each interface type. Duplicate `container.Provide()` calls for the same type SHALL NOT silently block subsequent registrations.

#### Scenario: Duplicate provide returns error early

- **WHEN** `container.Provide()` is called twice for `crmDomain.PipelineRepository`
- **THEN** the second call SHALL return an error that propagates up the call chain

### Requirement: ProvideDependencies error propagation

`db.Init()` MUST panic with a descriptive message when `ProvideDependencies()` returns an error.

#### Scenario: Init panics on ProvideDependencies failure

- **WHEN** `ProvideDependencies()` returns a non-nil error
- **THEN** `Init()` SHALL panic with a string prefixed by `"database init failed: "` followed by the error message

#### Scenario: Init succeeds on clean registration

- **WHEN** all `container.Provide()` calls succeed
- **THEN** `Init()` SHALL complete without error or panic

### Requirement: All legacy stores registered

The full set of domain stores (OrganizationStore, AccountStore, SubscriptionStore, DocumentStore, FileStore, FilePermissionStore, TemplateStore, IntegrationStore, ContentStore) MUST be registered in the DI container before the paywall module initializes.

#### Scenario: Paywall module resolves OrganizationStore

- **WHEN** the paywall module's middleware dependencies are resolved via `container.Invoke()`
- **THEN** `adapters.OrganizationStore` SHALL be resolvable from the container
