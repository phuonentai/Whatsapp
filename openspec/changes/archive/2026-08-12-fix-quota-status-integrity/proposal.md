## Why

The local subscription/quota state machine has four integrity holes plus one monetization leak, all of which strand paying or borderline orgs in wrong states:

1. **Unknown statuses collapse to `none` and skip the lazy guard** — `parseStatusFromReason` defaults to `StatusNone` (`infra/adapters/status_provider.go:86-100`) and the paywall lazy-guards only `Status != none` (`paywall/middleware.go:114`). Polar `revoked`/`incomplete`, MP raw `paused`/`in_process`, and MP `pending` (a trial/billing state the status mapper emits for pending preapprovals) all land as 402 with status `none` and no provider refresh is ever attempted — no recovery path, and a `pending` MP subscription (its trial window) reads as "no subscription".
2. **`GetQuotaStatus` INNER JOIN masks a present subscription** when the quota row is missing (`query/subscription_billing.sql`; the subscription and quota upserts are separate non-transactional writes, so a partial write leaves a valid subscription with no quota row) — status reads `none`, stuck 402, lazy guard skipped.
3. **`UpsertQuota` full-overwrites `invoice_count`** — a `customer.updated` or meter webhook clobbers any concurrently decremented count, inflating quotas.
4. **`DecrementInvoiceCount` has no lower bound** — `invoice_count = invoice_count - 1` with no `WHERE invoice_count > 0`, so concurrent consumption can drive the count negative.
5. **Inbound WhatsApp/agent path is not subscription-gated** — `agent_service.go:177-178` `analyze()` gates only on the credit check; with no quota row `CreditsMax=0` the gate passes, so a lapsed or never-subscribed org keeps receiving messages and accrues **unbilled metered AI consumption** (the inbound webhook path is not paywalled, confirmed by `cmd/seed-e2e/main.go:250-253`).

## What Changes

- `infra/adapters/status_provider.go` — unknown/unmapped statuses SHALL map to a distinct inactive status (not `none`), so the paywall lazy-guard refresh runs and the 402 message carries the real status; MP `pending` SHALL NOT read as `none`.
- `paywall/middleware.go` — lazy-guard condition SHALL include the distinct unknown-status state (refresh attempted before denying).
- `query/subscription_billing.sql` — `GetQuotaStatus` SHALL NOT mask a present subscription when the quota row is missing: LEFT JOIN with quota defaults (or error propagation that triggers reconciliation), and missing quota rows SHALL be seeded on upsert; regenerate SQLC (`make sqlc`).
- `subscription_repository.go` (UpsertQuota) — quota upsert SHALL preserve the current `invoice_count` when the incoming record carries no new count (metadata-only updates must not clobber consumed counts).
- `query/subscription_billing.sql` — `DecrementInvoiceCount` SHALL add `WHERE invoice_count > 0` and surface an exhaustion error; regenerate SQLC.
- `agent/app/services/agent_service.go` — `analyze()` SHALL require an active or trialing subscription before invoking the metered LLM call; organizations without one SHALL be refused (no unbilled AI consumption on the inbound webhook path).

## Capabilities

### New Capabilities

- `billing-quota-integrity`: status/derivation robustness (unknown statuses stay distinct and refreshable), quota-row reconciliation, non-destructive quota upserts, and bounded decrements.

### Modified Capabilities

- `ai-usage-metering`: metered LLM invocations SHALL require an active subscription on non-paywalled inbound paths; subscriptionless orgs SHALL NOT accrue billed credits.

## Non-Goals

- Changing the stored status vocabulary or paywall middleware semantics beyond the lazy-guard condition.
- Dunning/retry behavior (covered by `new-client-billing-lifecycle`).
- Trial seeding (covered by `new-client-billing-lifecycle`).
- No local credential storage; no Stytch API contract changes.

## Rollback

- **Git**: revert the change; SQLC regeneration is included in the revert (no schema change — query-only + generated code).
- **Backend**: behavior-only; reverting restores the old masking/overwrite behavior, so rollback should be followed by a verify-payment refresh for affected orgs.
- **Stytch**: untouched — no tenant policy changes.
