## 1. Metadata persistence [BE-INFRA]

- [x] 1.1 `handleSubscriptionUpsert` writes the parsed `ProductMetadata` into the upserted `Subscription.Metadata` (`process_webhook_event_service.go:294-304`; the quota path already extracts it at `:150-186`)
- [x] 1.2 `SyncModulesFromMetadata` call passes the actual keys derived from the persisted metadata; empty key set → no-op sync (`process_webhook_event_service.go:318-321`)
- [x] 1.3 `module_service.go:160-171`: `SyncModulesFromMetadata` treats absent/empty key sets as "no change" (defense in depth)

## 2. Tests [BE-DOMAIN]

- [x] 2.1 Unit test: Polar/MP subscription webhook upsert preserves product metadata on the row
- [x] 2.2 Unit test: webhook without metadata does not disable org modules (module state unchanged)
- [x] 2.3 Unit test: entitlement read reflects persisted metadata after a webhook (no verify/refresh needed)

## 3. Verification [OPS-GOV]

- [x] 3.1 Run `go build ./...`, `go vet ./internal/modules/...`, `go test ./...` — all pass
- [x] 3.2 Optional data fix: for orgs whose modules were wiped by pre-change webhooks, trigger verify-payment/refresh or re-apply module presets (manual ops step; document in archive notes)
- [x] 3.3 Record verification results and archive decision in `tasks.md`

---

## Gate results (2026-08-11)

### go build ./...
- **Exit 0** — full `go build ./...` passes (2026-08-11; re-run after a concurrent agent landed `internal/modules/organizations` wiring).

### go vet ./internal/modules/...
- **Exit 0** — full `go vet ./internal/modules/...` passes.

### go test ./...
- **Exit 0, all pass** — full `go test ./...` green.

### Targeted tests (this change)
- `TestHandleSubscriptionUpsert_PersistsProductMetadata` — PASS (2.1: Polar webhook upsert persists `product_metadata` on the row; module sync receives `["tickets"]`; quota still seeded with `invoice_count=25`)
- `TestHandleSubscriptionUpsert_NoMetadataDoesNotDisableModules` — PASS (2.2: Polar webhook without metadata → row metadata nil, module sync invoked with empty key set)
- `TestHandleSubscriptionUpsert_MPWebhookWithoutMetadataDoesNotDisableModules` — PASS (2.2 MP path: `subscription_authorized` without product metadata → no metadata persisted, empty module sync)
- `TestEntitlementReflectsPersistedMetadataAfterWebhook` — PASS (2.3: entitlement provider reads `crm_features`/`ai_features`/`module_tickets` back from the persisted row — no verify/refresh)
- `TestSyncModulesFromMetadata_EmptyKeysAreNoOp` — PASS (1.3: `SyncModulesFromMetadata(ctx, org, nil)` and `[]string{}` leave org module state untouched, `Delete` never called)

### Changed files
- `internal/modules/billing/app/services/process_webhook_event_service.go` — persist parsed `ProductMetadata` (nested under `product_metadata`, mirroring the Polar adapter shape) on the upserted `Subscription.Metadata`; module sync keys derive from the persisted metadata.
- `internal/modules/billing/app/services/process_webhook_event_service_test.go` — fakes (`fakeUpsertRepo`, `fakeSyncModuleSvc`, catalog/org-mod/ai/store fakes) + tests 2.1/2.2/2.3.
- `internal/modules/registry/app/services/module_service.go` — `SyncModulesFromMetadata` treats empty/absent key sets as no-op (defense in depth).
- `internal/modules/registry/app/services/module_service_test.go` — delete-recorder on `fakeOrgModRepo` + `TestSyncModulesFromMetadata_EmptyKeysAreNoOp`.

### Notes / deviations
- One transient gate block during verification: `internal/modules/organizations` was mid-edit by a concurrent agent (auth-session hardening) and briefly broke `go build ./...` / one `TestRevokeMemberSessions*` test. The peer landed their fix; final gate run is green. No organizations files were touched by this change.
- `make sqlc` not needed: no schema/query changes; no sqlc regeneration required.

## Archive notes

- **Manual ops step (3.2, not executed):** orgs whose modules were wiped by pre-change webhooks (empty `subscription_billing.subscriptions.metadata` after a `subscription.created`/`subscription.updated` with no later verify/refresh) need a one-time data fix: trigger the verify-payment/refresh path (Polar `GetSubscription` + upsert rewrites metadata) or re-apply the playbook module presets for those orgs. This change only prevents new wipes; it does not backfill.
- **Rollback:** revert this change; no migration (behavior-only). Rollback should be paired with a verify-payment refresh for affected orgs since it restores the destructive wipe.

**Archive deferred:** centralized verification phase per repo practice.
