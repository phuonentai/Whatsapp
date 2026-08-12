## 1. Schema & queries [DB-SQLC]

- [ ] 1.1 Verify next free migration number after `add-document-branding` lands (or reserve accordingly). Verify: `ls go-b2b-starter/internal/db/postgres/sqlc/migrations/ | tail -5`
- [ ] 1.2 Add migration `0000XX_create_quotes` (`crm.quotes` + `crm.quote_items` per design D1: UNIQUE(org,deal,version), totals NUMERIC(14,2), branding_snapshot_key TEXT, payload JSONB, items cascade with position/sku_ref nullable/quantity/unit_price/discount_percent/iva_percent/line_total) + down migration. Verify: `make sqlc` regenerates; up+down present
- [ ] 1.3 Add SQLC queries: quote CRUD by org/deal/version, active-quote lookup (highest version or canonical aprobada), items by quote, counter/next-number read with lock, status update, expire-by-valid_until list. Verify: `make sqlc` EXIT 0; queries in generated code

## 2. Domain model & state machine [BE-DOMAIN]

- [ ] 2.1 Define `domain.Quote`, `QuoteItem`, `QuoteStatus` with guarded transition table (borrador→enviada, enviada→aprobada|rechazada, enviada→vencida, revise→borrador v+1; reject unknown) with NO transport imports. Verify: `go build ./...`; grep confirms domain purity
- [ ] 2.2 Define `QuoteRepository` interface seam. Verify: `go build ./...`
- [ ] 2.3 Implement server-side total computation (line totals, subtotal, IVA, total) as pure functions with tests for Colombian IVA math incl. discounts. Verify: unit tests assert totals incl. discount+IVA edge cases

## 3. Quote service [BE-DOMAIN]

- [ ] 3.1 Implement `QuoteService` (create with items, list by deal, get, update items, transitions, revise, expiry job) enforcing state machine + org scoping + permissions + deal activity recording. Verify: unit tests per transition incl. unknown-transition rejection; activity recorded
- [ ] 3.2 Implement per-org consecutive numbering (`COT-0001`, prefix from branding config) transactionally with lock. Verify: concurrent test asserts no duplicate numbers
- [ ] 3.3 Implement deal integration: `monto` sync on aprobada + facturado guard (advisory default / hard via org flag) in deal service or listener. Verify: tests assert sync and both guard modes; existing invoicing tests still green

## 4. API + RBAC [BE-INFRA]

- [ ] 4.1 Add routes `/api/quotes` (org:manage write / org:view read), `/api/deals/:id/quotes`, `POST /api/quotes/:id/transition`, `POST /api/quotes/:id/revise` with Spanish errors + 403 gating. Verify: handler tests assert permission + validation + transition errors
- [ ] 4.2 Wire service + repository in DI. Verify: `go build ./...` EXIT 0; boot defers to live env

## 5. Frontend deal page section [FE-NEXT]

- [ ] 5.1 Add "Cotizaciones" section on deal detail: list versions, create/edit items, totals display, transition buttons (enviar/aprobar/rechazar), revise action. Verify: `npx tsc --noEmit` EXIT 0; component tests cover list/create/transition
- [ ] 5.2 Show active quote reference + synced monto on deal. Verify: component tests

## 6. Launch gate [OPS-GOV]

- [ ] 6.1 Run full gate: `make sqlc`, `go build ./...`, `go vet ./...`, `make test`. Verify: all EXIT 0, results recorded here
- [ ] 6.2 Frontend regression: `pnpm lint` at baseline, `npx tsc --noEmit`, `pnpm build`. Verify: recorded here
