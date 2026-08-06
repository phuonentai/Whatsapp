## Context

The DI container (uber-go/dig) fails on startup with `missing type: adapters.OrganizationStore` because the `registerDomainStores()` function in `internal/db/inject.go` exits early at line 211 — a duplicate `container.Provide` for `crmDomain.PipelineRepository` triggers an error that returns before the "legacy" stores section (lines 242-294) ever runs. `db/cmd/init.go` silently swallows the error from `ProvideDependencies`, so the misconfiguration only surfaces as a runtime panic when `container.Invoke()` resolves the paywall module's dependencies.

Error chain:

```
registerDomainStores()
  └─ duplicate PipelineRepository provide → error → return early
      └─ db.Inject() returns error
          └─ db.ProvideDependencies() returns error
              └─ db.Init() ignores error ← [bug]
                  └─ legacy stores (OrganizationStore, etc.) never registered
                      └─ paywall module needs OrganizationStore → panic
```

## Goals / Non-Goals

**Goals:**
1. Remove the duplicate `crmDomain.PipelineRepository` Provide in `internal/db/inject.go`
2. Add error propagation in `internal/db/cmd/init.go` so DI misconfiguration panics at init time with a clear message

**Non-Goals:**
- Refactoring the overall DI registration pattern (legacy vs. CRM sections)
- Adding runtime DI health checks or test coverage

## Decisions

### Decision 1: Remove duplicate Provide, keep the first

The duplicate `PipelineRepository` block at lines 207-212 is identical to lines 200-205. Delete lines 207-212. The first registration succeeds and is sufficient.

**Alternative considered:** Wrapping with `dig.IsProvide()` check. Rejected — no such API exists in dig; removing the duplicate is simpler.

### Decision 2: Propagate ProvideDependencies error

Change `db/cmd/init.go` from:
```go
func Init(container *dig.Container) {
    ProvideDependencies(container)
}
```
to:
```go
func Init(container *dig.Container) {
    if err := ProvideDependencies(container); err != nil {
        panic(fmt.Sprintf("database init failed: %v", err))
    }
}
```

This turns silent failures into immediate startup panics with a clear diagnostic, rather than surfacing as a confusing type-missing error downstream.

**Alternative considered:** Returning the error up the call chain. Rejected — all existing module `Init()` signatures return nothing. Changing the pattern across all modules is out of scope.

## Risks / Trade-offs

| Risk | Mitigation |
|------|-----------|
| A future duplicate Provide silently breaks startup | The `panic` in `init.go` makes this impossible to miss |
| Other duplicate Provides exist in the file | Only the one duplicate was found; the change is scoped to removing it |
