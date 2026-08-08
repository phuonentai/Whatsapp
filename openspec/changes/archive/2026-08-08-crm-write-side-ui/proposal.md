## Why

The CRM frontend is read-only: the contact and company tables render data but offer no create, edit, or delete flows, even though the backend fully supports CRUD (services, permission-gated routes) and the e2e suite (`contacts.spec.ts`, `companies.spec.ts`) defines the interactive contract. The build is also broken today: `queryKeys.crm` is referenced by every CRM query/mutation hook but does not exist in `query-keys.ts`, so `pnpm build` fails. Finally, the living specs (`contact-management`, `company-management`) already require conflict errors ("Ya existe un contacto con este correo electrónico.", "Ya existe una empresa con este nombre.") on duplicate email/name, but the backend surfaces generic 500s — blocking both the spec-required Spanish toasts and the e2e duplicate-name test.

## What Changes

- Add the `queryKeys.crm` namespace (13 keys matching existing hook signatures) so the build goes green and cache invalidation works (gate-zero commit).
- Build shared write-side foundations: reusable `ConfirmDialog` for delete confirmations, Spanish mutation-error mapping for sonner toasts (Toaster already mounted), and zod schemas with Spanish validation messages.
- Contactos write UI: "Nuevo contacto" toolbar button, create/edit dialog (phone, display_name, email, lead_status), row Editar/Eliminar actions, delete confirmation, Spanish validation.
- Empresas write UI: "Nueva empresa" toolbar button, create/edit dialog (name, nit, sector, ciudad), row Editar/Eliminar actions, delete confirmation, duplicate-name error surfaced.
- Backend enabler: map unique-violation (pgx 23505) in contact/company create/update services to HTTP 409 with the exact Spanish messages already spec'd (implements existing requirements, not new behavior).
- Add missing `useUpdateCompany` / `useDeleteCompany` mutation hooks and wire error toasts into all CRM mutations.
- W1 e2e specs (`contacts.spec.ts`, `companies.spec.ts`) pass; the ambiguous update step in `contacts.spec.ts` is adjusted (explicit "Editar" click) and coordinated with the in-flight `add-crm-e2e-tests` change.

## Capabilities

### New Capabilities

- none

### Modified Capabilities

- `crm-frontend`: new requirements for contact and company create/edit/delete UI in Colombian Spanish, Spanish validation messages, and Spanish mutation toasts.
- `contact-management`: no new requirements — the 409 conflict-error requirement already exists in the living spec; this change implements it.
- `company-management`: no new requirements — the 409 conflict-error requirement already exists in the living spec; this change implements it.

## Impact

- **FE**: `lib/hooks/queries/query-keys.ts` (+`crm` namespace), `lib/hooks/mutations/use-crm-mutations.ts` (+`useUpdateCompany`, `useDeleteCompany`, shared error-handling), new `components/crm/confirm-dialog.tsx`, `components/crm/contact-dialog.tsx`, `components/crm/company-dialog.tsx`, new `lib/crm/validation.ts` (zod schemas + Spanish messages) and `lib/crm/errors.ts` (Spanish error mapping); `contact-table.tsx` / `company-table.tsx` gain toolbar buttons and row actions.
- **BE**: `internal/modules/crm/app/services/contact_service.go`, `company_service.go` — unique-violation → 409 mapping (~20 lines, no migration, no SQLC changes).
- **Tests**: W1 e2e specs pass; `pnpm build` / `pnpm lint` green; `make test` unaffected. `add-crm-e2e-tests` (in flight, 35/44) is the coordination point for the spec adjustment.
- **Auth boundary**: no change. This change does not touch the Stytch B2B runtime SSOT — no credentials, sessions, or identity data are added locally; `stytch_member_id`/`stytch_organization_id` linkage and all Stytch B2B API contracts (JWKS verification, webhook signatures, circuit breaker) are unaffected. CRUD routes remain gated server-side by `contact:view/manage/delete`.
- **Rollback**: Git state — revert the FE commits (pure additive code) and the BE error-mapping commit (self-contained, reversible). Stytch tenant policy state requires no rollback because this change never mutates Stytch state.

## Non-Goals

- Detail pages (`?view=<entity>&id=<id>`), deal CRUD, drag-and-drop, pipeline editor, activity type filters / task fields, tag picker — deferred to follow-on phases.
- Server-side search wiring, pagination UI, inbox changes, sidebar navigation, Spanish/English language unification.
- Any DB migration, SQLC query change, or contact identity redesign.
- Local storage of credentials, MFA tokens, or session tokens (forbidden by constitution; unchanged).
