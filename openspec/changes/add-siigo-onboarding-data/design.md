## Context

`add-siigo-org-onboarding` delivers per-org connections (`invoicing.org_connections`), the `siigo | none` provider states, the guarded state machine (`connected → numeracion_ok → sandbox_ok → live`), and deal-stage gating. This change fills the two data gaps between `connected` and `live`: numeration continuity (legal requirement for clients already issuing in Siigo) and customer master-data import. It also adds the sandbox test-invoice proof required before `live`.

Repo facts: migration head is `000028` after change 1 → next free is `000029`. Deals carry `monto` only (no line items) → single "Venta WhatsApp" service line, no product import. CSV import already exists in `data-transfer` as the manual fallback. Siigo API numeration/customer-list specifics are unverified (spike in `add-siigo-invoicing` tasks 1.1/1.2 was deferred as external); the adapter is a thin plain-HTTP seam so contract surprises stay contained.

## Goals / Non-Goals

**Goals:**
- Numbering continuity: confirm resolución/prefix/next number before any production invoice; auto vs manual numbering mode
- Customer import: Siigo customers → empresas/contactos, NIT dedupe, preview-then-confirm, idempotent
- Delta sync on demand + nightly
- Sandbox test invoice proving rail before go-live
- Every run/confirmation recorded for audit

**Non-Goals:**
- No product catalog import (single service line MVP)
- No historical invoice import / ledger reconciliation
- No CSV import path duplication
- No numbering re-assignment of existing Siigo invoices
- No manual resolución override
- No wizard UI (change `add-siigo-onboarding-wizard`)

## Decisions

### 1. Spike first, external-deferrable

**Chosen:** Task 1 is an OPS-GOV spike: verify Siigo numeration endpoints (resolución/prefix/next number), customer list pagination, and whether invoice creation auto-assigns numbers. If no network access, record **Deferred (external)** (matching `add-siigo-invoicing` precedent) and implement the `manual` mode with the `auto` detection behind a config flag, verified at deployment.

**Alternatives considered:** *Skip spike, assume auto-assign* — rejected: wrong assumption = double numbering (platform + Siigo assign) or invalid invoices.

### 2. Numeration snapshot table + confirmation endpoint

**Chosen:** `invoicing.org_numerations` (org, provider resolución id, prefijo, next_number, mode `auto|manual`, confirmed_at, UNIQUE org). `GET /api/v1/org/siigo/numeration` reads live from provider; `POST /api/v1/org/siigo/confirm-numeration` stores snapshot + advances `connected→numeracion_ok`. Human confirmation is the legal checkpoint — never auto-confirm.

**Alternatives considered:** *Auto-confirm first read* — rejected: legality of consecutive numbering demands a human yes; also catches wrong-company cases early.

### 3. Manual numbering: single-flight next-number fetch, one conflict retry

**Chosen:** In `manual` mode the adapter fetches next number per org with the existing token single-flight pattern (per-org mutex + in-memory next-number reservation), passes it in the payload, and on a provider "number conflict" error fetches once more and retries exactly once. Invoice creation remains idempotent by the `(organization_id, deal_id)` unique constraint.

**Alternatives considered:** *Pre-reserve number blocks per org* — rejected: complexity + gaps in the DIAN sequence are worse than a retry. *No retry* — rejected: concurrent sales would fail sporadically.

### 4. Import: preview → confirm, NIT-first dedupe

**Chosen:** `ImportService` pulls Siigo customers (paginated), normalizes NIT (strip non-digits, upper), groups: nuevos / existentes (NIT match on `crm.empresas.nit`) / duplicados within the pull / sin NIT. Preview returns counts only; confirm upserts empresas (name unique rule: merge into existing company when NIT matches, else create; name collision without NIT match → flagged in preview as `sin NIT` or skipped with reason) + linked contact (phone/email from Siigo), all in one transaction, records `invoicing.import_runs`.

**Alternatives considered:** *Stream straight to DB without preview* — rejected: onboarding UX needs the confirmation moment; also surfacing duplicates prevents company-name collisions. *Sync into a staging table* — rejected: over-engineering; preview counts computed from the pull + a single existence query set.

### 5. Delta sync: idempotent upserts, no preview

**Chosen:** Same upsert path without preview; on-demand endpoint + nightly job (cron pattern matching the existing poller). Runs recorded in `import_runs`. Delta never deletes (Siigo deletions are outside scope; CRM is the system of record).

**Alternatives considered:** *Two-way sync* — rejected: CRM is the record; Siigo pushes customers, never pulls changes from us.

### 6. Sandbox test invoice: real round trip, gated

**Chosen:** `POST /api/v1/org/siigo/test-invoice` → adapter `CreateInvoice` in sandbox (no deal), row inserted in `invoicing.invoices` with a `test` marker (nullable `deal_id`), status awaited via existing webhook/poll path; on `valid` → `numeracion_ok→sandbox_ok`. Rejected 400 unless `SIIGO_SANDBOX=true`. The existing schema requires `deal_id` NOT NULL → test rows need `deal_id NULL` — schema delta in this change's migration (ALTER or new nullable column).

**Alternatives considered:** *Skip sandbox step* — rejected: onboarding claims "prueba antes de activar"; the step is the user-visible proof.

### 7. Audit surface

**Chosen:** `import_runs` (org, kind `preview|confirm|delta`, counts, pulled_at, error) and `org_numerations.confirmed_at` give the admin view (change 3) its data. No new audit capability; existing audit-log path reused for admin actions.

## Risks / Trade-offs

- [Siigo numeration/customer endpoints unverified] → Mitigation: spike task first; thin adapter seam; external-deferral pattern; `manual` mode default until verified.
- [Manual numbering races between concurrent invoices] → Mitigation: per-org single-flight + one conflict retry; worst case is a retried request, never a duplicate number.
- [Preview counts drift between preview and confirm] → Mitigation: confirm recomputes and re-checks within the transaction; drift logged, not fatal.
- [NIT-less customers block import] → Mitigation: counted and skipped with reason; never blocks the batch; reported in preview.
- [Test invoice rows with NULL deal_id touch existing constraint] → Mitigation: migration `000029` alters `deal_id` to nullable with the UNIQUE moving to a partial unique index on `(organization_id, deal_id)` where deal_id IS NOT NULL; existing rows unaffected.
- [Delta job load on Siigo API] → Mitigation: nightly off-peak + on-demand only; pagination respected; rate-limit errors recorded, not retried hot.

## Migration Plan

1. Migration `000029_onboarding_data` (+ down): create `invoicing.org_numerations` and `invoicing.import_runs`; ALTER `invoicing.invoices` deal_id → nullable + partial unique index `(organization_id, deal_id)` WHERE deal_id IS NOT NULL. `make sqlc` regenerates.
2. Spike task (OPS-GOV): verify Siigo numeration + customer endpoints (deferrable external).
3. Adapter additions: `GetNumeration`, `ListCustomers` (paginated), `CreateInvoice` numbered-payload support.
4. `NumerationService` (read/confirm/snapshot) + state advance; `ImportService` (preview/confirm/delta) + repositories; tests.
5. Routes: numeration GET/confirm, import preview/confirm, sync, test-invoice; DI wiring.
6. Nightly delta job registration.
7. Gate: `make sqlc`, `go build ./...`, `go vet ./...`, `go test ./...`; FE regression; archive decision.
8. Rollback: revert commits; down migration drops new tables + restores deal_id NOT NULL (test rows removed first if any); no Stytch state.

## Open Questions

- Does Siigo require a `resolución` reference in the invoice payload, or is numbering purely sequence-based? (spike)
- Should `sin NIT` customers be importable as contacts-only? (default: skipped, recorded)
- Is a manual resolución override needed by sales? (deferred; non-breaking later)
