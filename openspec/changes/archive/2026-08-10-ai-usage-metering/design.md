## Context

AI features exist today (RAG chat under `/example_cognitive/chat`, document embeddings in the cognitive module) but consumption is invisible: the OpenAI client (`internal/platform/llm/infra/openai_client.go`) returns `TokensUsed` on every completion/embedding response, and the RAG service surfaces `AssistantResponse.TokensUsed`, yet nothing persists it. Billing state lives in `subscription_billing` (migrations 000004/000005): `subscriptions` (plan + `metadata` JSONB with keys like `max_contactos`), `quota_tracking` (invoice-count quota, synced via Polar meter-grant webhooks). `FeatureProvider.GetEntitlement` (`internal/modules/billing/infra/features/billing_provider.go`) already returns `Entitlement.Usage`/`Quotas`/`Modules` per tenant. A best-effort meter-ingestion pattern exists in `consume_invoice_quota_service.go` (`IngestMeterEvent` → Polar meter `invoice.processed`; MercadoPago adapter is a verified no-op).

This design adds a per-tenant AI credit/token ledger on top of that state, without touching invoice quota semantics, the paywall, or the feature-flag derivation model.

## Goals / Non-Goals

**Goals:**
- Persist every LLM completion/embedding token usage per organization, per billing period, with an append-only audit trail
- Expose AI usage and remaining credits inside `Entitlement` so it flows through the existing tenant context and feature middleware
- Enforce a credit guard on AI routes (402 when exhausted) before the LLM is invoked
- Feed consumption back to the billing provider via the existing `IngestMeterEvent` seam (Polar real, MercadoPago no-op)
- Reuse the existing meter-grant webhook path to refresh the per-period AI allowance

**Non-Goals:**
- No new billing engine or payment flows (provider preapproval/checkout untouched)
- No changes to invoice quota (`invoice_count`) semantics or the paywall middleware
- No speech/transcription AI metering (schema is generic enough to extend later)
- No MercadoPago usage-billing support (adapter stays a logged no-op)
- No local credential storage — Stytch remains sole identity authority; ledger rows reference `organizations.id` only
- No changes to the feature-flag derivation model (metadata-driven flags stay as-is)

## Decisions

### D1: Ledger schema — two tables + one column

Migration `000016`:

- `subscription_billing.ai_usage` — one row per `(organization_id, period_start)`, with `tokens_input`, `tokens_output`, `tokens_embedding`, `credits_used`, `period_end`, `updated_at`. UNIQUE on `(organization_id, period_start)`.
- `subscription_billing.ai_usage_events` — append-only: `organization_id`, `feature`, `model`, `tokens_input`, `tokens_output`, `tokens_embedding`, `credits_consumed`, `request_id`, `created_at`. UNIQUE on `(organization_id, request_id)` to make recording idempotent. Index `(organization_id, created_at)`.
- `subscription_billing.quota_tracking.ai_credits_max INT NOT NULL DEFAULT 0` — the per-period allowance, refreshed via the existing subscription/metadata sync and meter-grant webhook paths (same lifecycle as `invoice_count_max`).

Rationale: mirroring the existing `quota_tracking`/`subscriptions` lifecycle keeps sync and webhook handling uniform. `ai_usage` holds mutable period totals (fast entitlement reads); `ai_usage_events` holds the immutable audit trail.

### D2: Credit conversion — static per-model table with fallback

`internal/modules/billing/infra/credits/rates.go`: `creditsPer1K(model) (input, output float64)` and embedding rate constant; unknown models fall back to a documented default (e.g., input=0.001/output=0.002 credits per 1K tokens) and log a warning for rate-table maintenance. Credits = round((input/1000)*rateIn + (output/1000)*rateOut) (+ embedding similarly). Rates are code constants for now — product tunable later.

### D3: TokenLedger interface in platform llm domain, implemented by billing infra

`internal/platform/llm/domain/service.go` gains:

```go
type TokenLedger interface {
    RecordUsage(ctx context.Context, event UsageEvent) error
}
type UsageEvent struct {
    OrganizationID  int32
    Feature         string // "rag_chat", "embedding", "completion"
    Model           string
    TokensInput     int
    TokensOutput    int
    TokensEmbedding int
    RequestID       string
}
```

Billing infra implements it (`internal/modules/billing/infra/ledger/token_ledger.go`): converts tokens → credits (D2), upserts `ai_usage`, inserts the event row, then fires the best-effort meter goroutine (D6). Dependency direction stays `billing → platform/llm/domain` (an interface, no transport) — no domain import of external packages.

### D4: Metered LLM client decorator

`internal/platform/llm/infra/metered_llm_client.go` wraps the OpenAI client and implements `domain.LLMClient`:

- `Complete`/`CompleteStream`: on success, extract `resp.TokensUsed` and record; on error, return without recording; on ledger failure, log and still return the response (fail-open, spec'd).
- `GenerateEmbedding`: needs token usage — the OpenAI client currently returns only `[]float64`. Extend `EmbeddingResponse` usage surfacing: change `GenerateEmbedding` to return tokens (e.g., add a sibling method or change the signature to return `([]float64, int, error)`); the decorator records the embedding tokens.
- `organization_id` is read from `context.Context` via a small helper (`usage.ContextWithOrgID`/`OrgIDFromContext`) in the platform llm domain, populated from Gin context (`c.Get("organization_id")`, set at `internal/modules/auth/middleware.go:261`) by the cognitive handler or a tiny middleware.

### D5: Entitlement integration

`billing_provider.go`:
- `getUsage` gains AI fields: read the org's `ai_usage` row for the current period (SQLC query `GetAiUsageByOrgAndPeriod`) → `ai_tokens_input`, `ai_tokens_output`, `ai_tokens_embedding`, `ai_credits_used`, `ai_credits_remaining` (max−used, floor 0).
- `parseQuotas` gains `ai_credits` from metadata key `ai_credits_max` (same pattern as `max_contactos`).
- Feature derivation (via `parseCRMFeatures`) gains `ai_assistant` from metadata key `ai_assistant`.

### D6: Meter ingestion — mirror the invoice pattern

After a successful record, a background goroutine (10s timeout, best-effort, logged) calls `billingProvider.IngestMeterEvent(ctx, externalID, "ai.tokens.consumed", credits)` exactly like `ingestMeterEventToPolar` in `consume_invoice_quota_service.go`. `handleMeterGrantEvent` in `process_webhook_event_service.go` is extended: when `MeterSlug == "ai.tokens"`, refresh `ai_credits_max` (upsert path identical to the existing invoice path). Both remain idempotent via the same transaction-isolated upsert pattern.

### D7: Enforcement — 402 guard on AI routes

New middleware `ai_credits_guard` (in the cognitive module, after `org_context`): reads the org's current-period credits via the ledger's read-only `CheckCreditsRemaining`; if `ai_credits_max > 0 && remaining <= 0`, abort with HTTP 402 `{"error": "ai_credits_exhausted"}`. Route order for `/example_cognitive/chat`: `auth` → `org_context` → `subscription` → `features.Require("ai_assistant")` → credit guard → `RequirePermissionFunc("resource","create")` → handler. No pre-decrement — recording happens post-response (fail-open by design, D4).

### D8: Usage endpoint

`GET /api/crm/usage/ai` (auth + org_context + subscription middleware): returns period usage/credits via `GetPeriodUsage` (zeroed row when none). Handler lives in the billing module (owns the ledger) or cognitive module; registered on the same router group as the features endpoint pattern.

### D9: Frontend consumption (light)

`subscription-tab.tsx` (or the paywall component) fetches `/api/crm/usage/ai` and renders "AI credits: X remaining" with period dates. Minimal; server-action pattern reuse.

## Risks / Trade-offs

- **Fail-open recording**: ledger DB errors don't block AI responses → possible undercount. Accepted: AI is a premium feature, availability beats strict metering; alerts on ledger failures surface problems. Ticket-metric reconciliation (Polar ledger) is a later hardening.
- **Period rollover**: `ai_usage` rows are per `period_start`; the guard uses the current period. If `quota_tracking.period_end` is stale (webhook missed), rollover misalignment is possible — mitigated by the existing fallback sync in `VerifyAndConsumeQuota` style flows and by keying on period bounds from the quota row.
- **Race on guard vs record**: no pre-decrement means bursty over-consumption beyond the allowance is possible within a period. Mitigated by provider-level meters (Polar is authoritative for billing) and acceptable for v1; strict atomic enforcement is a listed future option.
- **Embedding token surfacing**: requires changing `GenerateEmbedding`'s return signature — touches the platform llm client and its callers (`text_vectorizer`, `embedding_service`, `document_listener`); contained but must be done in one task with build verification.
- **Dependency direction**: billing infra implements a platform llm domain interface — acceptable (billing already depends on platform packages), but the `TokenLedger` interface must stay transport-free.
- **Rollback**: Git revert + `000016.down.sql` (drops tables/column); Polar ops: remove `ai.tokens.consumed` meter and `ai_credits_max` metadata key. Stytch policy untouched.
