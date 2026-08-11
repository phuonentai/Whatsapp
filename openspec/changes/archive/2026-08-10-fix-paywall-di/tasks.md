## 1. Remove duplicate PipelineRepository Provide

- [x] 1.1 [BE-INFRA] Delete the duplicate `crmDomain.PipelineRepository` Provide block (lines 207-212) in `internal/db/inject.go`

## 2. Add error propagation in db.Init

- [x] 2.1 [BE-INFRA] Update `internal/db/cmd/init.go` to check the error from `ProvideDependencies()` and panic with a descriptive message

## 3. Fix CRM Gin route conflict (discovered during implementation)

- [x] 3.1 [BE-INFRA] Rename tag-entity routes in `internal/modules/crm/routes.go` from `/:entityType/...` to `/entity/:entityType/...` to avoid Gin wildcard conflict with `/:id`
- [x] 3.2 [FE-NEXT] Update frontend `crm-repository.ts` tagEntity/untagEntity API paths to match new backend routes

## 4. Verify startup

- [x] 4.1 [BE-INFRA] Run `go build ./cmd/api/main.go` to verify compilation
- [x] 4.2 [BE-INFRA] Rebuild Docker images and restart containers
- [x] 4.3 [BE-INFRA] Verify `GET /health` returns 200
