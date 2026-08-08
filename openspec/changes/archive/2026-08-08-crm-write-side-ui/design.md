## Context

The CRM frontend (`next_b2b_starter`) is read-only today: `contact-table.tsx` and `company-table.tsx` render data with a client-side search box and lead-status filter, but no create/edit/delete flows. The backend is complete: `CreateContacto`/`UpdateContacto`/`DeleteContacto` and `CreateEmpresa`/`UpdateEmpresa`/`DeleteEmpresa` handlers exist with `ShouldBindJSON` request structs and permission-gated routes (`contact:view/manage/delete`). The frontend dependency stack is ready: react-hook-form, zod v4, sonner (Toaster already mounted in `app/layout.tsx`), and the full shadcn/ui kit (dialog, form, select, input, table).

Three blockers to the write-side UI:

1. **Build breaker**: `queryKeys.crm` is referenced by `use-crm-queries.ts`, `use-conversations-query.ts`, `use-messages-query.ts`, and `use-crm-mutations.ts`, but `query-keys.ts` defines no `crm` namespace (32 keys exist for other resources). `pnpm build` fails.
2. **No error contract**: create/update handlers return generic `500 "Error al crear empresa"` for every failure. The living specs (`company-management`, `contact-management`) already mandate conflict messages — "Ya existe una empresa con este nombre." / "Ya existe un contacto con este correo electrónico." — but nothing maps the PostgreSQL unique-violation (pgx 23505) to a distinguishable response. `companies.spec.ts` asserts `text=ya existe,text=duplicado` is visible, which cannot pass today.
3. **Missing mutation hooks**: `useCreateCompany` exists, but `useUpdateCompany` / `useDeleteCompany` do not; no CRM mutation shows an error toast.

Coordination: `add-crm-e2e-tests` is in flight (35/44 tasks) and owns the e2e specs; `crm-integrity-phase-a` (5/31) is hardening the same tables at the DB layer (delete semantics preserved, so no conflict). Environment: Next.js 16 + React 19, TanStack Query v5, strict TypeScript, Tailwind + shadcn/ui, Colombian Spanish UI.

## Goals / Non-Goals

**Goals:**
- Make `pnpm build` green via the missing `queryKeys.crm` namespace (gate zero).
- Deliver contact + company CRUD (create/edit/delete) in the CRM SPA, in Colombian Spanish, matching the e2e contract.
- Spanish validation and mutation-error toasts via sonner.
- Surface duplicate email/name as HTTP 409 with the exact spec'd Spanish messages (implements existing requirements).
- W1 e2e specs (`contacts.spec.ts`, `companies.spec.ts`) pass.

**Non-Goals:**
- Detail pages, deal CRUD, drag-and-drop, pipeline editor, activity filters/task fields, tag picker, server-side search wiring, pagination UI, inbox changes, sidebar nav, language unification, any DB migration or SQLC change, any Stytch contract change.

## Decisions

### D1: `queryKeys.crm` namespace — exact key inventory from hook call sites

The 13 keys used today (verified by reading every hook file) and their factory signatures:

| Key | Signature |
|---|---|
| `crm.contacts` | `(params?: { source?; lead_status?; limit?; offset? })` |
| `crm.contact` | `(id: number)` |
| `crm.companies` | `(params?: { limit?; offset? })` |
| `crm.company` | `(id: number)` |
| `crm.deals` | `(params?: { pipeline_id?; stage_id?; estado? })` |
| `crm.deal` | `(id: number)` |
| `crm.pipelines` | `()` |
| `crm.activities` | `(params?: { tipo? })` |
| `crm.contactActivities` | `(contactId: number)` |
| `crm.dealActivities` | `(dealId: number)` |
| `crm.tags` | `()` |
| `crm.conversations` | `(params?: { status?; limit?; offset? })` |
| `crm.messages` | `(conversationId: number)` |

Follow the existing pattern in `query-keys.ts` (`as const` factories). This lands as its own commit — the gate-zero verification is `pnpm build` and `pnpm lint` passing.

**Alternatives considered:** deleting the `.crm` references (rejected — every CRM hook depends on them; also the inbox relies on `crm.conversations`/`crm.messages`).

### D2: Shared foundations before any entity UI

- **`components/crm/confirm-dialog.tsx`** — shadcn `Dialog` wrapper: title/message in Spanish, destructive confirm button, cancel. Used by contact and company delete (e2e expects buttons matching `/confirmar|sí|eliminar/i`).
- **`lib/crm/errors.ts`** — pure function mapping a fetch/API error to a Spanish message: 409 → use the message from the response body (`{ error: "Ya existe una empresa con este nombre." }` — verify body shape against `response.Error`); other 4xx → "Solicitud inválida"; network → "Error de conexión". Unit-testable without a browser.
- **`lib/crm/validation.ts`** — zod v4 schemas: `contactSchema` (phone: required + Colombian E.164 pattern `^\+573\d{9}$`; email: optional + format; display_name optional; lead_status enum `['nuevo','contactado','calificado','descalificado','cliente']`) and `companySchema` (name: required; nit/sector/ciudad optional). Messages in Spanish, e.g. "El teléfono es requerido" / "Teléfono inválido" (e2e matches `/requerido|obligatorio|inválido/`).
- **Mutation error handling** — a shared `onError` handler (wraps `lib/crm/errors.ts` + `toast.error`) added to every CRM mutation in `use-crm-mutations.ts`, replacing the current invalidation-only handlers.

**Alternatives considered:** per-entity inline error handling (rejected — duplicated logic across 8+ mutations); a global Axios/fetch interceptor (rejected — the API client throws its own `ApiError`; keep mapping in one helper).

### D3: Contact/company interaction model

- Toolbar row above each table: "Nuevo contacto" / "Nueva empresa" button (primary), existing search input and lead-status filter unchanged.
- Row actions: "Editar" (opens dialog pre-filled) and "Eliminar" (`aria-label="Eliminar"`) → `ConfirmDialog` → `DELETE` + success toast + query invalidation.
- Create/edit dialog: single `contact-dialog.tsx` / `company-dialog.tsx` component supporting both modes (RHF + zod resolver), inputs named per the e2e page objects: `phone`, `display_name`, `email`, `lead_status`; `name`, `nit`, `sector`, `ciudad`. Guardar button submits.
- **Known e2e ambiguity (contacts.spec update step)**: the spec fills `input[name="display_name"]` and clicks Guardar without ever clicking "Editar". A dialog-based design cannot satisfy that literally. **Decision:** implement dialog-based editing and adjust `contacts.spec.ts` to click "Editar" first; coordinate with `add-crm-e2e-tests` (this is a clarification of intent, not a scope change — the crm-frontend spec says edits happen via an "Editar" action).
- Search stays client-side for W1 (e2e only asserts row counts after filling the search box). The server-side `SearchContactos`/`SearchEmpresas` endpoints remain unused until the detail/UX phase.

**Alternatives considered:** always-visible inline edit inputs to satisfy the loose assertion (rejected — bad UX and ambiguous selectors); omitting edit flow from W1 (rejected — the e2e CRUD test requires it).

### D4: Backend unique-violation → 409 mapping (implements existing spec requirements)

In `contact_service.go` and `company_service.go`, `Create` and `Update`: after the repository error, detect a unique violation via `errors.As(&pgconn.PgError{})` with `SQLState == "23505"` and return a domain-level conflict error (new typed error or sentinel per service). Handlers map it to `http.StatusConflict` with the spec'd Spanish message:

- contacts: "Ya existe un contacto con este correo electrónico." (unique index `idx_contacts_org_email`; also covers duplicate phone via `idx_contacts_org_phone` — message stays email-flavored per spec, log the constraint name)
- companies: "Ya existe una empresa con este nombre." (unique `(organization_id, name)`)

**Alternatives considered:** a global middleware translating 23505 (rejected — the requirement is per-entity, spec'd Spanish text differs, and a middleware could mask non-org-scoped constraint violations); frontend-only generic toast (rejected — cannot distinguish duplicate from other failures, and the e2e duplicate test would fail).

### D5: Button gating

CRUD buttons are gated by the existing entitlement payload (`/crm/entitlement` → `funcionalidades.crm_companies`, and `crm_contacts_manage`-equivalent for contacts), which the CRM page already fetches for tab gating. The client-side permission map (`lib/auth/permissions.ts`) has no contact/deal/pipeline permissions — adding them is deferred; the backend still enforces `contact:manage`/`contact:delete` server-side regardless of button visibility.

**Alternatives considered:** adding client permissions now (rejected — scope creep; entitlement + server enforcement suffice for W1).

### D6: JSON tags on request structs (discovered during apply — required for the write path)

The CRM request structs (`CreateContactRequest`, `UpdateContactRequest`, `CreateCompanyRequest`, `UpdateCompanyRequest`) had **no `json` tags**. Gin's `ShouldBindJSON` (encoding/json) matches keys case-insensitively against the field name, so the FE's snake_case payloads (`phone_number`, `display_name`, `lead_status`; `name` for companies) silently failed to bind: `phone_number` does not match `PhoneNumber`, and `name` does not match `Nombre`. Contacts were created with an empty phone and edits to `display_name`/`lead_status`/`tipo_empresa` were silently dropped. Added snake_case `json` tags matching the FE DTOs and spec field names — a minimal contract fix without which the write-side UI cannot function. The FE sends `Partial<ContactDto>`/`Partial<CompanyDto>` keys, so no FE payload change was needed. The 409 conflict mapping in D4 also depends on this: duplicate email on `Update` only reaches the DB unique index when `email` actually binds.

## Risks / Trade-offs

- **Response body shape for 409** — the toast mapping must parse the backend's error body; if `response.Error` uses a different shape than assumed, the duplicate e2e test fails. → Mitigation: verify the response shape first (task 0 of the FE error-mapping task reads the response helper).
- **23505 mapping breadth** — a single SQLState check could translate unrelated unique violations. → Mitigation: scope to the org-scoped unique indexes only; log `constraint_name` for traceability.
- **e2e spec churn** — `add-crm-e2e-tests` is in flight and may edit `contacts.spec.ts`/`companies.spec.ts` concurrently. → Mitigation: W1 pins its acceptance to the specs as of this change; the contacts update-step adjustment is coordinated, and verification runs the full W1 subset at the end.
- **zod v4 vs RHF resolver** — zod v4 is installed (`^4.1.5`); the RHF `zodResolver` package must be compatible. → Mitigation: confirm `@hookform/resolvers` supports zod 4 (or use manual resolver) in the validation task; test the dialog against a strict-mode build.
- **Gate-zero build failure cascade** — other latent type errors may surface once `queryKeys.crm` is fixed. → Mitigation: gate-zero task fixes only the namespace + any directly related type errors; unrelated pre-existing failures are logged and not silently patched.

## Migration Plan

1. Gate zero: `queryKeys.crm` → `pnpm build` green.
2. Foundations (ConfirmDialog, errors, validation) — additive, no behavior change.
3. Mutation hooks: add `useUpdateCompany`/`useDeleteCompany`, shared `onError`.
4. Backend 409 mapping (BE commit, independently revertible).
5. Contact dialog + row actions; company dialog + row actions (per entity, independently testable).
6. Adjust `contacts.spec.ts` update step (coordinated).
7. Verification: `pnpm build`, `pnpm lint`, W1 e2e subset, `make test`.

**Rollback (Git + Stytch):** each step is an independent revertible commit (FE additive; BE mapping self-contained, no migration). No Stytch tenant policy state is touched — this change never mutates Stytch state and stores no credentials/sessions (constitution-compliant).

## Open Questions

- None blocking. (Confirmed during planning: sonner Toaster mounted at `app/layout.tsx`; zod + RHF installed; backend handlers accept the required fields; delete semantics untouched by `crm-integrity-phase-a`.)
