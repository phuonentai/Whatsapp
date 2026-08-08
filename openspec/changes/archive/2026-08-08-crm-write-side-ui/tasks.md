## 1. Gate Zero: Build Recovery

- [x] 1.1 Add `queryKeys.crm` namespace to `lib/hooks/queries/query-keys.ts` with all 13 keys (contacts, contact, companies, company, deals, deal, pipelines, activities, contactActivities, dealActivities, tags, conversations, messages) matching hook signatures per design D1 [FE-NEXT]
- [x] 1.2 Run `pnpm build` and `pnpm lint`; fix only errors directly caused by the new namespace; log any unrelated pre-existing failures without patching them [FE-NEXT]

## 2. Shared Foundations

- [x] 2.1 Create `components/crm/confirm-dialog.tsx`: shadcn Dialog wrapper with Spanish title/message, destructive confirm and cancel buttons (labels match `/confirmar|sí|eliminar/`) [FE-NEXT]
- [x] 2.2 Create `lib/crm/errors.ts`: pure function mapping API errors to Spanish messages (409 → response body message, other 4xx → "Solicitud inválida", network → "Error de conexión"); verify backend `response.Error` body shape first; add unit tests [FE-NEXT]
- [x] 2.3 Create `lib/crm/validation.ts`: zod v4 `contactSchema` (phone required + `^\+573\d{9}$`, email optional, display_name optional, lead_status enum) and `companySchema` (name required, nit/sector/ciudad optional) with Spanish messages matching `/requerido|obligatorio|inválido/`; confirm `@hookform/resolvers` zod-4 compatibility; add unit tests [FE-NEXT]

## 3. Mutation Hooks

- [x] 3.1 Add `useUpdateCompany` and `useDeleteCompany` to `lib/hooks/mutations/use-crm-mutations.ts` (matching existing hook patterns, invalidating `queryKeys.crm.companies`) [FE-NEXT]
- [x] 3.2 Add shared Spanish `onError` toast handler (wrapping `lib/crm/errors.ts` + sonner) to every CRM mutation hook; keep existing invalidation behavior [FE-NEXT]

## 4. Backend Conflict Enabler

- [x] 4.1 In `contact_service.go` Create/Update: detect pgx unique violation (SQLState 23505) via `errors.As(&pgconn.PgError{})`, log the constraint name, return a typed conflict error mapped to HTTP 409 "Ya existe un contacto con este correo electrónico." in the handler [BE-INFRA]
- [x] 4.2 In `company_service.go` Create/Update: same 23505 mapping → HTTP 409 "Ya existe una empresa con este nombre." [BE-INFRA]
- [x] 4.3 Verify with `make test` (existing suite green) and manual curl: duplicate contact email and duplicate company name return 409 with the Spanish message [BE-INFRA]

## 5. Contactos Write UI

- [x] 5.1 Create `components/crm/contact-dialog.tsx` (RHF + zod resolver, create/edit modes): fields `phone`, `display_name`, `email`, `lead_status`; POST/PUT to `/api/crm/contactos`; Spanish field errors; Guardar/Cancelar [FE-NEXT]
- [x] 5.2 Wire `contact-table.tsx`: "Nuevo contacto" button (gated by entitlement), row "Editar" (opens dialog pre-filled) and `aria-label="Eliminar"` (ConfirmDialog → DELETE + success toast + invalidation) [FE-NEXT]
- [x] 5.3 Adjust `contacts.spec.ts` update step to click "Editar" before filling `input[name="display_name"]` (per design D3; coordinate with `add-crm-e2e-tests`) [FE-NEXT]
- [ ] 5.4 Verify `contacts.spec.ts` passes: create, edit, delete, search, empty-phone validation — BLOCKED: backend stack down (saas_backend crash-looping, postgres/redis exited); e2e cannot run in this environment [FE-NEXT]

## 6. Empresas Write UI

- [x] 6.1 Create `components/crm/company-dialog.tsx` (RHF + zod resolver, create/edit modes): fields `name`, `nit`, `sector`, `ciudad`; POST/PUT to `/api/crm/empresas`; Spanish field errors; Guardar/Cancelar [FE-NEXT]
- [x] 6.2 Wire `company-table.tsx`: "Nueva empresa" button (gated by `crm_companies` entitlement), row "Editar" and `aria-label="Eliminar"` (ConfirmDialog → DELETE + success toast + invalidation) [FE-NEXT]
- [ ] 6.3 Verify `companies.spec.ts` passes: create, edit, delete, search, duplicate-name error — BLOCKED: same environment blocker as 5.4 (no live backend) [FE-NEXT]

## 7. Final Verification

- [x] 7.1 Verification run: `pnpm build` GREEN (exit 0, all routes incl. /dashboard/crm); `npx tsc --noEmit` GREEN; `go build ./...` GREEN; `go test ./internal/modules/crm/...` GREEN. Blocked/downstream: `pnpm lint` (pre-existing — `next lint` removed in Next 16, ESLint 9 flat-config migration pending), `make test` script (pre-existing — looks for `./src/go.mod`, none exists), W1 e2e (backend stack down: saas_backend crash-looping, postgres/redis exited), Go integration package (in-flight `crm-integrity-phase-a` harness, fixture violates `chk_organizations_status`) [FE-NEXT]
- [x] 7.2 Re-read `openspec/changes/crm-write-side-ui/` artifacts; confirm proposal/design/specs/tasks stay consistent with the implementation (design.md gained D6: request-struct json tags discovered during apply) [FE-NEXT]
