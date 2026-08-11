## Why

The Docker Compose-managed backend panics on startup due to a DI container failure: `missing type: adapters.OrganizationStore (did you mean postgres.Store?)`. This prevents the backend from serving requests and blocks signup from working end-to-end.

## What Changes

- Remove duplicate `crmDomain.PipelineRepository` registration in `internal/db/inject.go` — the identical block at lines 207-212 causes `registerDomainStores()` to return early, preventing all subsequent stores (OrganizationStore, AccountStore, SubscriptionStore, etc.) from being registered in the DI container
- Add error propagation in `internal/db/cmd/init.go` — the current silent swallowing of `ProvideDependencies` errors hides DI misconfiguration until runtime

## Capabilities

### New Capabilities

- `di-container-fix`: Ensure the uber-go/dig container is correctly wired so all domain stores are registered and startup panics are caught at init time rather than at first request

### Modified Capabilities

<!-- No existing capabilities change their requirements — this is an internal infrastructure fix. -->

## Impact

- `go-b2b-starter/internal/db/inject.go` — remove duplicate PipelineRepository provide block
- `go-b2b-starter/internal/db/cmd/init.go` — add error propagation on ProvideDependencies
