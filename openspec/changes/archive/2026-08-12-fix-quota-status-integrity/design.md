## Context

The paywall's status derivation collapses anything it does not recognize to `none` (`infra/adapters/status_provider.go:86-100`), and the lazy guard only fires for `status != none` (`paywall/middleware.go:114`) — so unknown statuses (Polar `revoked`/`incomplete`, MP raw `paused`/`in_process`, MP `pending`) become permanent 402s with no refresh, no recovery, and a misleading status. Separately, the quota row is coupled to the subscription row by an INNER JOIN (`GetQuotaStatus`) while the two are written non-transactionally, `UpsertQuota` overwrites `invoice_count` wholesale, `DecrementInvoiceCount` can go negative, and the inbound agent path runs metered LLM calls for orgs with no subscription at all.

## Goals / Non-Goals

**Goals:**
- Unknown statuses stay distinct (inactive, not `none`) and trigger the lazy-guard refresh
- A present subscription is never masked by a missing quota row; missing quota rows are seeded
- Quota updates do not clobber consumed counts; decrements cannot go negative
- No metered AI consumption for organizations without an active subscription

**Non-Goals:**
- New status values or paywall semantics beyond the lazy-guard condition
- Dunning/retry, trial seeding, payment-method changes (other changes own those)

## Decisions

### 1. Unknown statuses are distinct, not `none`

`parseStatusFromReason` default maps to a dedicated unknown-inactive state (the raw status string is preserved in the reason). The paywall lazy-guard condition becomes `status != none && status != unknown` — wait, the guard must RUN for unknown statuses, so the condition is `status == unknown → refresh`. Concretely: unknown statuses trigger `RefreshSubscriptionStatus` before the 402, matching the spec's lazy-guard premise ("local state stale → ask the provider"). MP `pending` maps to the unknown/inactive path (it is a valid MP state, not "no subscription").

**Alternatives considered:**
- Map unknown → `past_due`-like: fabricates a payment problem — rejected; keep the state honest and let refresh decide.

### 2. Quota row reconciliation

`GetQuotaStatus` uses a LEFT JOIN with quota defaults (invoice_count 0, ai_credits_max 0) so a present subscription always reads as its real status; the subscription upsert path seeds a quota row when none exists (single code path, still two writes but the read no longer masks). Missing quota rows can no longer strand an org as `none`.

### 3. Non-destructive quota upsert

`UpsertQuota` distinguishes "incoming record carries a count" from "metadata/status-only update": the latter preserves the stored `invoice_count` (and `ai_credits_max` unless explicitly provided). Meter-grant and `customer.updated` webhooks therefore cannot inflate consumed counts.

### 4. Bounded decrement

`DecrementInvoiceCount` gains `WHERE invoice_count > 0` and returns a quota-exhausted error when the count is already zero, so concurrent consumption cannot drive the count negative.

### 5. Agent gate requires subscription

`agent_service.go` `analyze()` consults the billing status (active/trialing) before the metered LLM call on the inbound webhook path. Subscriptionless or lapsed orgs get a refusal (message not processed by the AI agent), not free unbilled inference. The AI-usage ledger stays append-only; the gate prevents the unbilled rows from existing.

## Risks / Trade-offs

- **[Risk] LEFT JOIN + seeding changes read behavior org-wide** → Mitigation: quota defaults (0) match today's empty-row semantics; only the masking is removed.
- **[Risk] Refusing inbound agent messages for lapsed orgs changes WhatsApp behavior** → Mitigation: the refusal is scoped to the metered AI analysis, not message delivery; delivery continues, analysis is skipped.
- **[Risk] SQLC regeneration churn** → Mitigation: query-only changes; `make sqlc` regenerates `subscription_billing.sql.go`; no migration.

## Migration Plan

1. Status provider: distinct unknown state + lazy-guard condition; unit tests.
2. SQL: LEFT JOIN `GetQuotaStatus`, seeded quota row on subscription upsert, bounded decrement; `make sqlc`.
3. `UpsertQuota` preservation semantics + tests.
4. Agent gate subscription check + tests.
5. `go build ./...`, `go vet ./internal/modules/...`, `go test ./...`.
6. Rollback: revert the change; no migration.
