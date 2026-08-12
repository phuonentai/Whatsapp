## Context

Verified current state:

- Deals exist (`crm.deals`) with `monto` (single lump sum), `moneda` (COP), `metadata` JSONB, contact/company links, Spanish status lifecycle (`abierto/ganado/perdido/abandonado`), `DealStageChanged` events consumed by `DealStageListener` (which triggers Siigo invoicing on `facturado`).
- Pipeline stage "Cotización" exists in playbooks with a guion promising to send the quote ("Te preparamos la cotización y te la enviamos hoy") — but no quote entity or document exists.
- The invoicing connection state machine (`none → awaiting_setup → connected → numeracion_ok → sandbox_ok → live`, guarded transitions, terminal `invoicing_disabled`) is the repo's proven pattern for guarded state machines — quotes mirror it.
- Deal activity convention: activities of type `sistema` with Spanish messages ("Negocio ganado", "Facturación no activa") recorded on lifecycle events.
- `add-document-branding` defines the branding snapshot key convention (`org_id:updated_at`) that quotes reference.
- Procurement's SKU catalog exists (`procurement.*`) with products/SKUs — a nullable `sku_ref` on quote items reserves the linkage without coupling this change to procurement.

## Goals / Non-Goals

**Goals:**
- First-class, org-scoped, versioned quote entity (1:N per deal)
- Line items with Colombian tax math (subtotal, discount, IVA, total)
- Guarded state machine with revision loop
- Deal `monto` sync + aprobada-guard on `facturado` (advisory)
- Audit trail on the full quote lifecycle

**Non-Goals:**
- No PDF/rendering/delivery (that is `add-quote-delivery`)
- No branding storage (that is `add-document-branding`; quotes store only the snapshot key)
- No AI drafting/approval extraction (deferred; seams reserved)
- No SKU-catalog linkage implementation (nullable `sku_ref` reserves it)

## Decisions

### 1. Schema: `crm.quotes` + `crm.quote_items`

**Chosen:**

```
crm.quotes (
  id BIGSERIAL PK,
  organization_id BIGINT NOT NULL,
  deal_id BIGINT NOT NULL REFERENCES crm.deals(id),
  version INT NOT NULL,                 -- 1,2,3... per deal
  status TEXT NOT NULL,                 -- borrador|enviada|aprobada|rechazada|vencida
  number TEXT NOT NULL,                 -- COT-0001 (per-org sequence)
  valid_until TIMESTAMPTZ,
  currency TEXT NOT NULL DEFAULT 'COP',
  subtotal NUMERIC(14,2), iva_total NUMERIC(14,2), total NUMERIC(14,2),
  branding_snapshot_key TEXT,           -- "org_id:updated_at" per add-document-branding
  payload JSONB NOT NULL DEFAULT '{}',  -- future extension fields
  created_by BIGINT, created_at, updated_at,
  UNIQUE (organization_id, deal_id, version)
)

crm.quote_items (
  id BIGSERIAL PK,
  quote_id BIGINT NOT NULL REFERENCES crm.quotes(id) ON DELETE CASCADE,
  position INT NOT NULL,
  description TEXT NOT NULL,
  sku_ref BIGINT NULL,                  -- future procurement linkage
  quantity NUMERIC(12,2) NOT NULL,
  unit_price NUMERIC(14,2) NOT NULL,
  discount_percent NUMERIC(5,2) NOT NULL DEFAULT 0,
  iva_percent NUMERIC(5,2) NOT NULL DEFAULT 19,  -- default from org branding, snapshotted here
  line_total NUMERIC(14,2) NOT NULL
)
```

**Rationale:** `UNIQUE(organization_id, deal_id, version)` enforces versioning (like the invoicing `UNIQUE(organization_id, deal_id)` idempotency pattern, but versioned). Totals are denormalized on the quote for fast list views and to preserve history even if items change. IVA percent is snapshotted per line so a later tax-rule change never alters historical quotes.

### 2. State machine (guarded)

**Chosen:**

```
borrador ──enviar──▶ enviada ──aprobar──▶ aprobada   (terminal, feeds facturado)
   │                   │  │
   │                   │  └──rechazar──▶ rechazada ──nueva versión──▶ borrador(v+1)
   │                   └──vencer (job)──▶ vencida  ──nueva versión──▶ borrador(v+1)
   └──editar (within borrador/enviada)──┘
```

Transitions implemented as a guard table (map of current → allowed targets), rejecting unknown transitions with an error — mirroring the invoicing connection state machine. Only `enviada` may be approved/rejected; only `borrador`/`enviada` are editable; `aprobada` is terminal (a new offer = new version).

**Alternatives considered:**
- *Status as free string* — rejected; the repo's pattern is guarded enums.
- *Version as same-row overwrite* — rejected; loses history and audit value (Option B's whole point).

### 3. Deal integration

**Chosen:** When a quote reaches `aprobada`: deal `monto` syncs to quote total, deal activity recorded ("Cotización aprobada — total $X"). When a deal would move to `facturado`: the service checks for an `aprobada` quote — if none, it records an advisory activity ("Cotización no aprobada") and blocks the transition (configurable via org flag, default advisory-block).

**Rationale:** `facturado` currently auto-creates the Siigo invoice from deal `monto`. Syncing monto from the approved quote guarantees the invoice amount matches what the client approved — closing the consistency trap identified in exploration. The advisory guard prevents silent invoices on unapproved offers while not breaking existing automation out of the gate.

### 4. Numbering

**Chosen:** Per-org consecutive sequence `COT-0001` (prefix from branding config, default "COT"), generated transactionally (SELECT FOR UPDATE on a per-org counter row, or computed from `MAX(version)` scan under lock — spike decides; repo precedent: numeration continuity in invoicing reads provider-side next-number, but quotes are platform-owned so a local counter is appropriate).

### 5. API + RBAC

**Chosen:** `/api/quotes` CRUD (org:manage write, org:view read), `/api/deals/:id/quotes` listing, `POST /api/quotes/:id/transition` (enviar/aprobar/rechazar), `POST /api/quotes/:id/revise` (creates v+1 from current). Spanish error messages, 403 on permission violation (repo convention).

### 6. Extensibility seams

**Chosen:** `QuoteRepository` domain interface (infra implementation behind it) so `add-quote-delivery` adds delivery side-effects without touching the domain; `payload` JSONB documented for known future keys (payment_terms, delivery_terms, ai_draft, approved_by_extraction). The branding snapshot key is stored on the quote at creation.

## Risks / Trade-offs

- **Advisory vs hard block on `facturado`**: hard-block is safer commercially but risks regressing existing invoice automation for orgs that never use quotes. Mitigation: org flag, default advisory with activity trail.
- **Version explosion**: repeated revise loops can fragment deals; mitigated by listing only the active (highest-version) quote prominently and treating `aprobada` as the canonical one.
- **Tax math drift**: line totals + quote totals are denormalized — drift risk if edited in two places; mitigated by recomputing totals server-side on every mutation (totals are never client-supplied).
- **Counter sequence race**: mitigated by transactional counter; spike confirms approach against repo precedent (no existing local sequence util found; invoicing uses provider-side numeration).
