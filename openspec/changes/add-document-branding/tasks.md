## 1. Schema & queries [DB-SQLC]

- [ ] 1.1 Verify next free migration number (repo previously had `000020` duplicated and used `000021` for invoicing; confirm current tail before writing). Verify: `ls go-b2b-starter/internal/db/postgres/sqlc/migrations/ | tail -5` — pick next free number
- [ ] 1.2 Add migration `0000XX_create_org_branding` (`document_branding.org_branding`: org_id unique, logo_file_asset_id nullable, primary_color, accent_color, letterhead_text, terms_footer, validity_days INT default 15, default_iva_percent NUMERIC(5,2) default 19, quote_number_prefix default 'COT', config JSONB default '{}', created_at, updated_at) + down migration. Verify: `make sqlc` regenerates; up+down present
- [ ] 1.3 Add SQLC queries for branding get-by-org (with default fallback semantics at service layer) + upsert + touch updated_at. Verify: `make sqlc` EXIT 0; queries appear in generated code

## 2. Domain interface [BE-DOMAIN]

- [ ] 2.1 Define `domain.OrgBranding` value object + `DocumentBrandingProvider` interface (`GetBranding(ctx, orgID)`) with NO file-asset/storage/transport imports. Verify: `go build ./...`; grep confirms no infra imports in domain
- [ ] 2.2 Define `branding_updated` / `logo_updated` audit event payloads. Verify: `go build ./...`

## 3. Branding service + repository [BE-INFRA]

- [ ] 3.1 Implement `BrandingService` (get/update) with default fallback (no row → defaults + company name/NIT from existing company data) and optimistic concurrency via `updated_at`. Verify: unit tests assert default fallback + update bumps timestamp
- [ ] 3.2 Implement logo upload: content-type whitelist (PNG/JPEG, reject SVG), size ≤ 2MB, dimension sanity, store via `FileAssetStore` with branding category, reference asset on branding row, audit `logo_updated`. Verify: unit tests assert accept/reject paths and audit emission; no bytes in logs

## 4. API + audit [BE-INFRA]

- [ ] 4.1 Add routes `GET /api/branding` (org:view), `PUT /api/branding` + `POST /api/branding/logo` (org:manage, multipart) with Spanish 403/400 errors. Verify: handler tests assert permission gating + validation errors
- [ ] 4.2 Wire `DocumentBrandingProvider` + service in DI (`init_mods.go`, named bindings). Verify: `go build ./...` EXIT 0; server boot defers to live env per repo gate practice

## 5. Frontend settings UI [FE-NEXT]

- [ ] 5.1 Build "Marca / Documentos" settings section: logo upload with client-side validation (type/size), color pickers, letterhead/terms textareas, validity/IVA/prefix inputs, sample document-header preview. Verify: `pnpm lint` no new errors beyond baseline; `npx tsc --noEmit` EXIT 0
- [ ] 5.2 Wire section to branding API via existing client pattern. Verify: component tests cover render + save + preview update

## 6. Launch gate [OPS-GOV]

- [ ] 6.1 Run full gate: `make sqlc`, `go build ./...`, `go vet ./...`, `make test`. Verify: all EXIT 0, results recorded here
- [ ] 6.2 Frontend regression: `pnpm lint` at documented baseline, `npx tsc --noEmit`, `pnpm build`. Verify: recorded here (external exceptions noted per repo practice)
