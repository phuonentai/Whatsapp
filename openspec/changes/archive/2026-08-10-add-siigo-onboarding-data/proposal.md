## Why

Clients targeted by onboarding already issue DIAN electronic invoices in Siigo. Two things must happen before their first auto-issued invoice: (1) **numbering continuity** — Colombian invoices must be consecutive per DIAN resolución; if the platform issues a number the client's Siigo sequence has already passed, the invoice is legally invalid; (2) **master data** — the CRM must import the client's Siigo customers so deals reference real companies, instead of re-digitando. A sandbox test-invoice round trip proves the whole rail works before go-live.

## What Changes

- **Numeration verification**: the system reads the active DIAN resolución (prefijo, rango, próximo número) from the Siigo connection, presents it for human confirmation, and stores a numbered snapshot. Confirmation advances the connection from `connected` to `numeracion_ok` (state machine extended by `add-siigo-org-onboarding`).
- **Numbering continuity modes**: `auto` — Siigo assigns the next number at invoice creation (snapshot is read-only confirmation); `manual` — the adapter fetches the next available number per org (single-flight) and includes it in the invoice payload, retrying once on number conflict.
- **Customer import**: paginated pull of Siigo customers → normalized NIT → upsert into `crm.empresas` + linked `crm.contactos`, dedupe by `(organization_id, nit)`. Preview endpoint (counts: nuevos, existentes, duplicados, sin NIT) → confirm endpoint commits the batch.
- **Delta sync**: on-demand + nightly job re-pulling customers, idempotent upserts by NIT.
- **Sandbox test invoice**: `POST /api/v1/org/siigo/test-invoice` creates a sandbox invoice, waits for status sync (webhook/poll), and on `valid` advances the connection to `sandbox_ok`. Gated on `SIIGO_SANDBOX=true`.
- **Import/numeration audit**: every import run and numeration confirmation is recorded with timestamps and counts.

## Capabilities

### New Capabilities
- (none — all behaviour is requirements of the existing `invoicing` capability)

### Modified Capabilities
- `invoicing`: numbering continuity becomes a spec-level requirement (resolución/prefix/next-number snapshot + confirmation + auto/manual modes); customer import from the provider with NIT dedupe; delta sync; sandbox test-invoice round trip.

## Impact

- **Go backend**: new migration `000029` (`invoicing.org_numerations`, `invoicing.import_runs` + SQLC queries); `SiigoAdapter` extended with numeration and customer-list operations; new `NumerationService` and `ImportService` in the invoicing module; endpoints `GET /api/v1/org/siigo/numeration`, `POST /api/v1/org/siigo/confirm-numeration`, `GET /api/v1/org/siigo/import/preview`, `POST /api/v1/org/siigo/import/confirm`, `POST /api/v1/org/siigo/sync`, `POST /api/v1/org/siigo/test-invoice`; nightly delta job; state-machine transitions `connected→numeracion_ok→sandbox_ok` consumed from the `add-siigo-org-onboarding` service.
- **Database**: two new tables in `invoicing` schema; existing `crm.empresas`/`crm.contactos` populated via existing repositories (unique `(organization_id, name)` respected; NIT dedupe precedes name matching).
- **Auth boundary / Stytch**: unchanged — all endpoints org-scoped via existing auth; no Stytch contract or RBAC changes; no new identity data.
- **Dependencies**: no new vendor SDKs — plain HTTP calls through the existing adapter; Siigo API specifics for numeration endpoints remain unverified (spike task 1, external deferral allowed).
- **Frontend**: none (wizard consumes these endpoints in `add-siigo-onboarding-wizard`).
- **Rollback strategy**: Git — revert commits; DB — down migration drops the two new tables; Siigo side — test invoices live only in sandbox, no production numbering consumed before `live`; Stytch tenant policy — unaffected.

## Non-Goals

- No product/service catalog import (deals carry a single "Venta WhatsApp" service line for MVP).
- No historical invoice import or ledger reconciliation.
- No CSV import (exists in `data-transfer` as the manual fallback; not duplicated here).
- No changes to the WhatsApp send path or invoice notification.
- No numbering *re-assignment* — the platform never renumbers existing Siigo invoices.
- No manual override of the confirmed resolución in this change (deferred; may follow if sales needs demand it).
