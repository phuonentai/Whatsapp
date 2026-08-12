## Context

`handleSubscriptionUpsert` (`process_webhook_event_service.go:294-304`) is the shared Polar/MP webhook persistence path: it upserts the subscription row and then reconciles org modules. It drops the parsed `ProductMetadata` from the row and passes `moduleKeysFromMetadata(nil)` — an empty list — to `SyncModulesFromMetadata`, which treats the empty list as "the org now has zero modules" and deletes every module not present (`module_service.go:160-171`). Consequences: `subscription_billing.subscriptions.metadata` is empty after every webhook, entitlements derived from it (`billing/infra/features/billing_provider.go` → `crm_features`/`ai_assistant`) are empty, and paying orgs 403 on gated routes until a verify/refresh path happens to rewrite the row. For a brand-new client, the first `subscription.created` webhook — the event that should unlock their plan — is the event that disables their modules.

## Goals / Non-Goals

**Goals:**
- Subscription webhooks persist product metadata on the subscription row
- Module sync becomes a no-op when the webhook carries no metadata (or the key set is unchanged)
- Paying orgs keep entitlements after a webhook without needing a verify/refresh

**Non-Goals:**
- Changing webhook payload handling, quota semantics, paywall middleware, or module validation
- Backfilling metadata for already-wiped orgs (that is a data-fix task, tracked in tasks.md as an optional step)

## Decisions

### 1. Metadata persists on the upserted row

`handleSubscriptionUpsert` already computes `ProductMetadata` from the event for the quota path; the same value SHALL be written to `Subscription.Metadata` so the row and the quota agree. The metadata column is `jsonb` and `Subscription.Metadata` already round-trips through the repository (`subscription_repository.go`), so no schema change is needed.

**Alternatives considered:**
- Re-derive entitlements from the quota row instead of subscription metadata: larger refactor of the entitlement provider — rejected for this change.

### 2. Empty keys = no change, not disable-all

`SyncModulesFromMetadata` gets the module key set derived from the persisted metadata. When the set is empty (no metadata on the event) or unchanged from the stored set, the sync is a no-op. The destructive branch (delete modules not in the list) only runs when the event actually carries a key set that differs.

**Alternatives considered:**
- Skip module sync entirely from the webhook path: modules would never reconcile when a plan change legitimately alters entitlements — rejected; the sync stays, keyed on real changes.

## Risks / Trade-offs

- **[Risk] Legitimate plan downgrades stop disabling modules** → Mitigation: the sync still runs on real key-set changes; only absent/unchanged metadata is a no-op, which matches the intent (metadata absent means the event doesn't express module changes).
- **[Risk] Orgs already wiped by past webhooks stay wiped** → Mitigation: optional data-fix task (re-run verify/refresh or re-apply playbook presets) listed in tasks.md; the change prevents new wipes.

## Migration Plan

1. Persist `ProductMetadata` in `handleSubscriptionUpsert`; pass actual keys to `SyncModulesFromMetadata`; make empty-key sync a no-op in `module_service.go`.
2. Unit tests: metadata preserved; no-metadata webhook does not disable modules; entitlement reads reflect persisted metadata.
3. Optional data fix: for orgs wiped before this change, trigger a verify-payment/refresh or re-apply module presets.
4. `make test` (backend), `go build ./...`.
5. Rollback: revert the change; no migration.
