## Why

Every subscription webhook wipes product metadata and all org modules. `handleSubscriptionUpsert` (`process_webhook_event_service.go:294-304`) builds the `Subscription` without its `Metadata` — the parsed `ProductMetadata` at `:150-186` is used only for quota — then `:318-321` calls `SyncModulesFromMetadata` with `moduleKeysFromMetadata(nil)`, i.e. an empty key list. `module_service.go:160-171` deletes every module not in the list, so the very webhook that creates a new org's subscription also disables its modules, and the empty `subscription_billing.subscriptions.metadata` column yields empty entitlements (`billing/infra/features/billing_provider.go` → empty `crm_features`/`ai_assistant`) — a paying customer 403s on gated CRM/AI routes until a verify or refresh path rewrites the metadata. This is the shared Polar/MP webhook path, so it affects every provider and every org, including brand-new clients whose first `subscription.created` webhook arrives minutes after signup.

## What Changes

- `process_webhook_event_service.go` — `handleSubscriptionUpsert` SHALL persist the parsed `ProductMetadata` into the upserted `Subscription.Metadata` (the quota path already extracts it; the subscription row must keep it too).
- `SyncModulesFromMetadata` invocation — SHALL use the actual keys derived from the persisted metadata; when the webhook carries no metadata (or metadata is unchanged), the module sync SHALL be a no-op and SHALL NOT disable modules.
- `module_service.go` — `SyncModulesFromMetadata` SHALL treat an absent/empty key set as "no change" rather than "disable everything" (defense in depth; the billing path stops passing empty keys as well).
- Tests — add unit tests: subscription webhook upsert preserves metadata; webhook without metadata does not disable org modules; entitlement reads reflect persisted metadata after a webhook.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `module-registry`: per-org module state SHALL NOT be mutated by subscription webhook processing when the event carries no product metadata; module sync SHALL only apply when the metadata key set changes.
- `billing-provider-ux` (via shared path): the entitlement derived from subscription metadata SHALL remain populated after a subscription webhook, so paying orgs keep their `crm_features`/`ai_assistant` grants without requiring a verify/refresh.

## Non-Goals

- Changing what metadata Polar/MP webhooks may carry — this change only stops destructive handling of absent metadata.
- Altering paywall middleware, quota semantics, or the module registry validation path.
- No local credential storage; no Stytch API contract changes.

## Rollback

- **Git**: revert the change; no migration.
- **Backend**: the fix is behavior-only (metadata persistence + no-op module sync); reverting restores the destructive wipe, so rollback should be paired with a verify-payment refresh for affected orgs.
- **Stytch**: untouched — no tenant policy changes.
