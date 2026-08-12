## 1. Status derivation [BE-DOMAIN]

- [x] 1.1 `parseStatusFromReason` maps unknown statuses to a distinct inactive state preserving the raw status in the reason (never `none`) (`infra/adapters/status_provider.go:86-100`) — VERIFIED in tree (2026-08-11): default branch returns `paywall.StatusUnknown`; `none` reserved for no-subscription.
- [x] 1.2 Paywall lazy-guard condition runs `RefreshSubscriptionStatus` for the unknown state before denying (`paywall/middleware.go:114`) — VERIFIED: comment + `!status.IsActive && status.Status != StatusNone` gate includes unknown; refresh-to-active heals.
- [x] 1.3 MP `pending` SHALL resolve through the unknown/inactive path, not `none` (assert in `MapMPStatus` tests) — VERIFIED: `mp_webhook_parser_test.go:182-187` and `status_provider_test.go:42-46` assert `pending → StatusUnknown`, `!= none`.
- [x] 1.4 Unit tests: unknown status triggers refresh; refresh-to-active heals; 402 never reports `none` for unknown statuses — VERIFIED: `middleware_test.go` `TestUnknownStatusTriggersRefreshBeforeDenying` + `TestUnknownStatusRefreshToActiveHeals`; pass.

## 2. Quota row integrity [DB-SQLC]

- [x] 2.1 `GetQuotaStatus` uses LEFT JOIN with zeroed quota defaults so a present subscription always reads its real status (`query/subscription_billing.sql`) — VERIFIED: LEFT JOIN + `COALESCE(q.invoice_count, 0)` at `subscription_billing.sql:116-127`.
- [x] 2.2 Subscription upsert path seeds a missing quota row (single shared code path) — VERIFIED: `UpsertQuota` SQL `-1 → 0` default seeds on insert; webhook paths (`process_webhook_event_service.go:282-283,364`) call it for fresh subscriptions.
- [x] 2.3 `DecrementInvoiceCount` gains `WHERE invoice_count > 0` and returns a quota-exhausted error (`query/subscription_billing.sql`) — VERIFIED: bounded UPDATE at `subscription_billing.sql:93-102`; `subscription_repository.go:109-114` maps `ErrNoRows` → exhaustion error.
- [x] 2.4 Run `make sqlc`; confirm `gen/subscription_billing.sql.go` reflects the LEFT JOIN, seeded defaults, and bounded decrement — VERIFIED: gen file matches (LEFT JOIN at gen:162, bounded decrement at gen:17-19, `COALESCE(NULLIF(...,-1),0)` at gen:579-589); no regeneration needed.

## 3. Non-destructive quota upsert [BE-INFRA]

- [x] 3.1 `UpsertQuota` preserves stored `invoice_count`/`ai_credits_max` when the incoming record carries no new values (`subscription_repository.go`) — VERIFIED: `-1` sentinel preserved atomically in SQL (`COALESCE(NULLIF($2::int,-1), quota_tracking.invoice_count)`); `handleCustomerUpdated` passes `-1` on metadata-only updates.
- [x] 3.2 Unit tests: metadata-only update preserves count; count-carrying update replaces it; meter-grant webhook does not inflate consumed counts — DONE (2026-08-11): added `TestHandleCustomerUpdated_MetadataOnlyPreservesCount` (asserts -1 sentinel passed + stored count preserved), `TestHandleCustomerUpdated_CountCarryingUpdateReplacesCount`, `TestHandleMeterGrantEvent_AiTokensDoesNotInflateConsumedCounts`; extended `fakeUpsertRepo` to emulate the SQL `COALESCE(NULLIF(-1), stored)` contract (stored row + copy-returning read). Gate: `go test ./internal/modules/billing/app/services/` PASS (9/9 incl. pre-existing).

## 4. Agent gate [BE-DOMAIN]

- [x] 4.1 `agent_service.go` `analyze()` requires active/trialing subscription (via billing status) before the metered LLM call; subscriptionless orgs are refused (`agent_service.go:177-178`) — VERIFIED: billing status gate at `agent_service.go:179-189` returns `subscription_required` before the metered call.
- [x] 4.2 Unit tests: no-subscription org refused with no ledger rows; active org analysis proceeds — VERIFIED: `TestNoSubscriptionOrgRefusedWithNoMeteredLLMCall` (0 LLM calls, escalation created, flow awaits human) + `TestActiveOrgAnalysisProceeds`; pass.

## 5. Verification [OPS-GOV]

- [x] 5.1 Run `go build ./...`, `go vet ./internal/modules/...`, `go test ./...` — all pass — DONE (2026-08-11): full baseline sweep green (see Phase 0 record); `go test ./...` exit 0 across all packages.
- [x] 5.2 Record verification results and archive decision in `tasks.md` — verification recorded in this reconciliation; archive decision recorded below (archiving now, 2026-08-11).

## Phase 0 baseline checkpoint (2026-08-11, repo-wide active-changes run)

- [x] Repo-wide baseline recorded BEFORE further implementation work on this change (working tree: ~330 modified files across both apps from sibling in-flight changes):
  - `go build ./...` PASS (exit 0) · `go vet ./...` PASS · `go test ./...` PASS (all packages, exit 0) — go-b2b-starter
  - `npx tsc --noEmit` PASS · `pnpm lint` PASS (0 errors / 4 pre-existing warnings) · `pnpm build` PASS — next_b2b_starter
  - Context: this baseline anchors later verification gates — failures introduced by this change are distinguishable from pre-existing tree state. (Note: the fix code for this change is already in the tree — `StatusUnknown` distinct state, bounded `invoice_count > 0` decrement, lazy-guard includes unknown — and passes the baseline; checkbox reconciliation is part of this change's later phases.)

## Phase 1 reconciliation (2026-08-11, repo-wide active-changes run)

- [x] Code-point adjudication completed: 13 of 14 tasks verified complete in tree (see per-task VERIFIED notes above); gates `go build ./...`, `go vet ./...`, `go test ./...` all PASS on the current tree.
- [x] Single remaining gap closed (2026-08-11): task 3.2 unit tests written and passing (metadata-only preserves count, count-carrying replaces, ai.tokens grant never touches quota) — see 3.2 note. Full gate re-run: `go test ./internal/modules/billing/... ./internal/modules/paywall/... ./internal/modules/agent/...` PASS (exit 0), `go build ./...` PASS, `go vet ./...` PASS.
- [x] **Archive:** all tasks complete and locally verified; delta specs synced to living specs (ai-usage-metering + new billing-quota-integrity) on 2026-08-11; change archived 2026-08-11.
