## Why

Entitlements today are static boolean feature flags derived from subscription metadata (verified: `parseCRMFeatures` in `internal/modules/billing/infra/features/billing_provider.go` reads flags like `crm_contacts_manage` from `subscription_billing.subscriptions.metadata`). The existing LLM stack — `internal/platform/llm/infra/openai_client.go` — already surfaces `TokensUsed` on every completion/embedding response, but no local ledger records that consumption and no tier-based gate maps it to a metered allowance. A pure boolean gate leaves unmapped API cost overruns once AI features (RAG chat, smart replies, summarization) scale. This change introduces an AI Credit / Token Consumption Ledger per tenant, integrated into the existing `Entitlement` tenant context, with best-effort meter events to the billing provider.

## What Changes

- **New ledger tables**: migration adds `subscription_billing.ai_usage` (per-org, per-period running totals of input/output/embedding tokens and credits used) and `subscription_billing.ai_usage_events` (append-only audit ledger: org, feature, model, tokens, credits, request_id, timestamp). `subscription_billing.quota_tracking` gains `ai_credits_max` (period allowance, synced from provider metadata / meter grants)
- **Token → credit conversion**: static per-model conversion (credits per 1K tokens, input/output/embedding differentiated) in billing infra, with a documented fallback default for unknown models
- **AiUsageLedger service** in the billing module: `RecordUsage`, `GetPeriodUsage`, `CheckCreditsRemaining` (read-only guard). Recording is idempotent per request_id
- **Metered LLM client decorator**: new `TokenLedger` interface in `internal/platform/llm/domain`; `internal/platform/llm/infra` gains a decorator wrapping the OpenAI client that records tokens from `CompletionResponse.TokensUsed` / `EmbeddingResponse.TokensUsed` into the ledger. Billing infra provides the `TokenLedger` implementation (billing → platform llm domain, allowed direction)
- **Tenant context integration**: `Entitlement.Usage` (returned by `GetEntitlement` in `billing_provider.go`) gains `ai_tokens_input`, `ai_tokens_output`, `ai_tokens_embedding`, `ai_credits_used`, `ai_credits_remaining`; `Entitlement.Quotas` gains `ai_credits` from new metadata key `ai_credits_max` (mirrors existing `max_contactos` parsing). New feature flag `ai_assistant` gates the cognitive routes
- **Enforcement**: cognitive `/example_cognitive/chat` route gains a credit guard (HTTP 402 `{"error": "ai_credits_exhausted"}`) when the org's period credits are exhausted; recording still happens after a successful response (no pre-decrement race)
- **Billing provider metering**: after recording, best-effort background ingestion of `ai.tokens.consumed` meter events via the existing `BillingProvider.IngestMeterEvent` (mirrors `consume_invoice_quota_service.go`). `handleMeterGrantEvent` extended to accept meter slug `ai.tokens` and refresh `ai_credits_max`. MercadoPago adapter remains a documented no-op
- **Usage endpoint**: `GET /api/crm/usage/ai` returning current-period usage (tokens, credits used/remaining, period bounds)
- **Frontend (light)**: subscription/paywall UI shows AI credits remaining for the current period

## Capabilities

### New Capabilities

- `ai-usage-metering`: per-tenant AI credit/token ledger — schema, `AiUsageLedger` service, metered LLM instrumentation, credit math, enforcement guard, provider meter ingestion, and the usage endpoint

### Modified Capabilities

- `feature-gating`: `Entitlement.Usage`/`Entitlement.Quotas` gain AI usage and credit fields sourced from the ledger; new `ai_assistant` feature flag gates AI routes (flags continue to derive from subscription metadata; no `plans.go` file is introduced — see Assumptions)

## Impact

- **Go backend**: new migration `000016`; new SQLC queries in `query/subscription_billing.sql` + regenerated code; new files `internal/modules/billing/domain/usage_ledger.go`, `internal/modules/billing/app/services/ai_usage_ledger_service.go`, `internal/modules/billing/infra/repositories/ai_usage_repository.go`, credit-conversion helper in `internal/modules/billing/infra/features/` or `infra/credits/`, `internal/platform/llm/infra/metered_llm_client.go`, new `TokenLedger` interface in `internal/platform/llm/domain/service.go`, credit guard middleware + usage handler in `internal/modules/cognitive/`, DI wiring in `init_mods.go` / `cognitive/module.go` / `billing/cmd/init.go`. `billing_provider.go` (`getUsage`, `parseQuotas`) extended. Existing Polar adapters, paywall middleware, invoice quota semantics untouched
- **Database**: new `ai_usage`, `ai_usage_events` tables; `ai_credits_max` column on `quota_tracking`. All scoped to `subscription_billing` schema (same Stytch-org FK pattern, no credentials)
- **Frontend**: usage display in `subscription-tab.tsx` (or equivalent paywall component), client fetch from the new endpoint
- **Dependencies**: none new (SQLC generated code, Go stdlib)
- **Config**: no new required env vars (credit conversion defaults are code constants; Polar product metadata key `ai_credits_max` and meter slug `ai.tokens.consumed` configured in the Polar dashboard — ops step)
- **Auth**: no changes. Routes reuse existing `auth` + `org_context` + `subscription` middleware and `RequirePermissionFunc`; the credit guard runs after `org_context` (reads `organization_id` from Gin context, set at `internal/modules/auth/middleware.go:261`)
- **Ops**: Polar dashboard — add `ai_credits_max` to product metadata, create meter `ai.tokens.consumed` with meter grant events; re-verify after deploy
- **Rollback**: Git — revert the change (migration, routes, DI). DB — run `000016.down.sql` dropping the new tables/column. Polar — remove/disable the meter and metadata key. Stytch tenant policy state is unaffected (no auth changes); no local credentials are introduced anywhere
- **Non-Goals**: not building a custom recurring billing engine (Polar/MP preapproval remain the billing source); no speech/transcription AI metering in this change (ledger schema is generic enough to extend later); MercadoPago meter ingestion remains a no-op (no MP usage-billing support); not replacing the static feature-flag derivation model — metering is additive on top of it; rejects any local credential storage (Stytch remains the sole identity authority)

## Assumptions

- **`ai_credits_max` metadata key**: Polar product metadata currently uses keys like `max_contactos`/`max_negocios` (verified in `parseQuotas`). The new `ai_credits_max` key and its value per plan must be configured in the Polar dashboard — not verifiable from this repo
- **Polar meter creation**: the meter `ai.tokens.consumed` and its grant events must exist in the Polar dashboard before `IngestMeterEvent` succeeds; until then ingestion fails safely (logged, best-effort, same as invoice meter behavior)
- **`plans.go` drift**: the `feature-gating` spec references `internal/platform/features/plans.go` as the plan→feature mapping source of truth, but that file does not exist in the repo (verified); flags derive from subscription metadata instead. This change does not create `plans.go`; it only adds AI fields to the existing metadata-driven derivation
- **`GET /api/crm/features` drift**: the `feature-gating` spec claims this endpoint exists, but no backend route was located. This change is unaffected and does not add it (separate concern)
- **Credit conversion rates**: the per-model credits-per-1K-token map is a static code default; exact economic rates are a product decision to be tuned later
- **MercadoPago meter support**: `MPAdapter.IngestMeterEvent` is a verified no-op (logs only); AI metering for MP organizations therefore stays local-only until MP adds usage-billing support
