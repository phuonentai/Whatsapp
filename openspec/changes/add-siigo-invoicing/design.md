## Context

The MVP closes "run the whole sale inside WhatsApp" for Colombian SMEs. Verified current state:

- Webhook ingress precedent exists: `POST /api/v1/webhooks/whatsapp`, `/polar`, `/mercadopago` (billing/routes.go:51-54) with per-provider signature verification (billing/handler.go:386-406, HMAC/Svix).
- Deal stage changes already emit `events.DealStageChanged` (crm/app/services/deal_service.go:95) and are consumed by `DealStageListener.HandleStageChanged` (crm/app/services/deal_stage_listener.go), subscribed in crm/cmd/init.go:56 — the trigger point for invoicing.
- `ProviderRouter` (billing/infra/routing/provider_router.go:15) is the proven per-provider seam: domain `BillingProvider` interface, per-provider adapters, per-org resolver.
- Companies/contacts already store `nit`, `sector`, `tipo_empresa` (company-management, crm-core-data) — the exact fields Siigo's `customers` resource needs. No data-model gap.
- MercadoPago adapter + payment links exist; the "pay by PSE/Nequi" rail is live.
- No invoicing/siigo code exists anywhere in the backend — greenfield module.
- Siigo API specifics are UNVERIFIED (no network access in this environment): assumed REST + OAuth2 `client_credentials`, sandbox availability, webhook vs polling notification model, DIAN CUFE handling on Siigo side. These live in the proposal's Assumptions until a spike verifies them.

The local DB is the system of record for invoice state; Siigo is the DIAN-certified invoicing rail.

## Goals / Non-Goals

**Goals:**
- Close the invoicing hole: deal reaches stage `facturado` → Siigo invoice created automatically, idempotently (one per deal)
- Provide a per-provider invoicing seam (`InvoicingProvider` + `InvoiceRouter`) mirroring the billing `ProviderRouter` so a second provider (Alegra) can slot in without rework
- Sync invoice/DIAN status back into the local DB via signed webhook with polling fallback
- Notify the customer inside WhatsApp: invoice link + MercadoPago payment link via the existing send path
- Keep Siigo credentials out of PostgreSQL (env/config only, in-memory token cache)

**Non-Goals:**
- No Alegra adapter implementation (interface/router slot only)
- No generic integration framework or marketplace
- No inventory, payroll, Puntored, or shipping integrations
- No FE work beyond optionally surfacing the invoice link on the deal page
- No changes to WhatsApp compliance/consent flow other than reusing the existing send path
- No Stytch/auth changes; no local credential storage

## Decisions

### 1. Per-provider webhook endpoint, matching repo precedent

**Chosen:** `POST /api/v1/webhooks/siigo`, signature-verified in the handler before any DB mutation, dispatching to the invoicing service.

**Alternatives considered:**
- *Single dispatch endpoint* — rejected; contradicts repo precedent (whatsapp/polar/mercadopago are per-provider).
- *Polling only* — rejected as primary; webhook gives near-real-time status, polling retained only as fallback (D3).

### 2. `InvoicingProvider` domain interface + `InvoiceRouter` (mirror of billing ProviderRouter)

**Chosen:** Domain interface with the operations the CRM actually needs:

```
type InvoicingProvider interface {
    CreateInvoice(ctx, orgID int32, req *domain.InvoiceRequest) (*domain.Invoice, error)
    GetInvoiceStatus(ctx, orgID int32, externalID string) (*domain.Invoice, error)
    UpsertCustomer(ctx, orgID int32, companyID int32) (*domain.CustomerRef, error)
}
```

`InvoiceRouter` resolves the provider per org (Siigo today; `billing_provider`-style resolver pattern, or a dedicated `invoicing_provider` column) and delegates. Domain models MUST NOT import Siigo SDKs or HTTP transport; the adapter in `infra/siigo` implements the domain interface (governance rule).

**Alternatives considered:**
- *Concrete Siigo service only* — rejected; bakes in single-vendor coupling when the billing router already established the multi-provider pattern, and the proposal explicitly reserves the Alegra slot.

### 3. Trigger = extend `DealStageListener`, idempotent via unique constraint

**Chosen:** `DealStageListener.HandleStageChanged` (or a new listener on the same event) checks `NewStageName == "facturado"` and calls `InvoicingService.CreateForDeal`. Idempotency enforced by `UNIQUE(organization_id, deal_id)` on `invoicing.invoices` inside the same transaction; a re-trigger finds the existing invoice and is a no-op returning it.

**Alternatives considered:**
- *Manual 1-click button on deal page* — rejected as sole path; violates the "no re-digitar" story, requires FE work. Kept as optional future manual "re-enviar factura" action.
- *Trigger on any deal in a won stage by configurable stage key* — deferred; the playbook already names the winning stage `facturado` (vertical-playbooks), so matching on the stage name string is sufficient for MVP and avoids a config surface.

### 4. Status sync: signed webhook primary + polling fallback

**Chosen:** Webhook updates status (valid/invalid, CUFE, PDF URL) with idempotent transaction-isolated updates; a periodic job polls `GetInvoiceStatus` for invoices stuck in a non-final state (safety net). Webhook handler validates signature before mutation (per whatsapp/polar precedent).

**Alternatives considered:**
- *Polling only* — rejected as primary (D1); latency and rate cost.
- *Webhook only* — risky; if Siigo's notification model is unreliable or absent (unverified), status never lands. Polling fallback is cheap and covers it.

### 5. OAuth2 token cache (mirror JWKS cache)

**Chosen:** `infra/siigo` holds an in-memory token cache keyed by org/provider-account with TTL (default 300s, matching the JWKS cache TTL convention), refreshing via OAuth2 `client_credentials` on expiry. Secrets live in env (viper), never in DB or logs.

**Alternatives considered:**
- *No cache, refresh every call* — rejected; every invoice call would burn an OAuth round-trip and risk rate limits.
- *Persist tokens in DB* — rejected by Non-Goals (no local credential storage).

### 6. Schema: `invoicing.invoices` (small, local system of record)

**Chosen:** Table scoped by `organization_id` with `UNIQUE(organization_id, deal_id)`:

```
invoicing.invoices (
  id BIGSERIAL PK,
  organization_id BIGINT NOT NULL,
  deal_id BIGINT NOT NULL,
  external_id VARCHAR NOT NULL,        -- Siigo invoice id
  cufe VARCHAR,
  status VARCHAR NOT NULL,             -- pending|valid|invalid|errored
  pdf_url TEXT,
  amount NUMERIC(14,2),
  currency VARCHAR(3) DEFAULT 'COP',
  created_at, updated_at,
  UNIQUE (organization_id, deal_id)
)
```

SQLC-generated queries only. No FK to Siigo; Siigo is a rail, local DB is the record.

### 7. WhatsApp notification via existing send path

**Chosen:** On invoice creation or status→valid, the invoicing module publishes a message through the existing WhatsApp send path (same mechanism the playbook guiones / deal service use) with invoice + MP payment link. Message template `factura_lista` (must be created/approved in Meta as part of deployment, matching the WhatsApp operational checklist).

**Alternatives considered:**
- *Send via a new dedicated client* — rejected; reuses existing send path and keeps the consent/send guardrails in one place.

## Risks / Trade-offs

- [Siigo API contract unverified (OAuth grant, endpoints, webhook model, sandbox access)] → Mitigation: an upfront spike task (verify + document the API before adapter coding); design keeps the provider seam thin so any contract surprises stay inside `infra/siigo`.
- [DIAN/CUFE semantics unknown] → Mitigation: store CUFE/status as opaque strings; relay Siigo's values without interpreting them; UI/link just points at Siigo's PDF.
- [Stage-name string matching is brittle if a tenant renames their `facturado` stage] → Mitigation: MVP accepts the brittleness (playbook seeds the name); documented in Open Questions; a future configurable stage-key is non-breaking.
- [Webhook reliability unknown] → Mitigation: polling fallback (D4) guarantees eventual consistency.
- [Token caching expiry races / expired token mid-request] → Mitigation: single-flight refresh and a one-retry-on-401 policy inside the adapter.
- [SIigo price tier / API cost] → Mitigation: sandbox spike confirms cost model before committing; if prohibitive, revisit MP facturación electrónica (flagged, unverified).

## Migration Plan

1. Add migration `00XXXX_create_invoices` + SQLC queries; `make sqlc` regenerates.
2. Ship domain interface + router + Siigo adapter behind env-gated config (sandbox default, no live calls until deployed).
3. Register webhook route; extend `DealStageListener`; wire DI in `init_mods.go`.
4. Deploy; verify sandbox invoice + webhook delivery against live Siigo sandbox credentials (external, deferred to deployment like the MP sandbox deferrals).
5. Rollback: revert commit(s); reverse migration (drop `invoicing.invoices`); no Stytch/tenant policy state to roll back.

## Open Questions

- Does Siigo notify via true webhook or only events/polling? (spike — determines whether D4 fallback is the primary path)
- Should the `facturado` stage key be configurable per org now or later?
- Does the MP payment link live in the invoice body or as a separate WhatsApp message?
- Is the WhatsApp `factura_lista` template approved at deployment time (Meta approval lead time)?
