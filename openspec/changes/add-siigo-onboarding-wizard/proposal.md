## Why

The backend now supports full Siigo onboarding (connection, numbering, import, sandbox test — changes `add-siigo-org-onboarding` and `add-siigo-onboarding-data`), but clients and operators have no interface to it. Onboarding is a user experience: three client types branch differently (has Siigo / no-Siigo manual / no-Siigo assisted), status must never be silent, and a kill-switch must be one toggle away. Without the wizard, the whole onboarding product is unreachable.

## What Changes

- **Settings section "Integración Siigo"** in the existing settings dashboard (`app/dashboard/settings/components/`), mirroring the `whatsapp-config-section.tsx` pattern: live status banner driven by the connection state machine, kill-switch toggle (pause/resume), `factura_lista` template approval status.
- **5-step wizard**: 1) Conectar Siigo (credentials form + NIT match result) → 2) Numeración (resolución/prefijo/próximo número read + human confirmation) → 3) Importar clientes (preview counts + dedupe review + confirm) → 4) Prueba en sandbox (test invoice + CUFE/status) → 5) Activar. Steps unlock only when the backend state allows; progress persists server-side, resume supported.
- **Branching paths**: no-Siigo clients see either "Facturación desactivada — activa con Siigo" (manual path) or "Tu equipo está configurando tu facturación" (assisted/`awaiting_setup`), never an unexplained block.
- **Admin view**: per-organization onboarding table (status, numeración snapshot, last import run, errors) and an assisted-provisioning form (admin RBAC) that writes credentials for `awaiting_setup` orgs — the no-Siigo upsell motion.
- **API client + data layer**: TanStack Query hooks over the change 1/2 endpoints, react-hook-form for credential/numeración/import forms, existing shadcn/ui components.

## Capabilities

### New Capabilities
- (none — the wizard is UI over existing capability APIs)

### Modified Capabilities
- `settings-ui`: settings dashboard gains the Siigo integration section, wizard steps, branching states, and kill-switch toggle as spec-level requirements.
- `admin-panel-navigation`: the admin surface gains the onboarding overview + assisted provisioning view with sidebar navigation, following the existing permission-filtered navigation pattern.

## Impact

- **Frontend**: new `siigo-integration-section.tsx` (+ wizard sub-components) in `app/dashboard/settings/components/`, wired in `settings-content.tsx`; new `lib/api/` client functions + TanStack Query hooks for the status/connect/numeration/import/sandbox/kill-switch endpoints; admin onboarding view + nav entry; component tests following `modules-section.test.tsx` precedent. No new dependencies (shadcn/ui, TanStack Query, react-hook-form already in use).
- **Backend**: none — consumes endpoints from changes 1 and 2 only. If an endpoint shape is missing for a UI need (e.g., test-invoice status polling helper), it is added via those changes' tasks, not ad hoc.
- **Auth/RBAC**: settings sections render under existing workspace-admin scoping; admin view under admin role (existing RBAC mapping). No Stytch changes.
- **Rollback strategy**: Git — revert commits; feature is UI-only, backend state untouched by reverting the wizard; a broken section can be hidden by reverting the `settings-content.tsx` wiring while keeping hooks.

## Non-Goals

- No invoice detail page or invoice list UI (invoice link stays in WhatsApp; optional later).
- No product catalog UI (single service-line MVP; no import).
- No changes to WhatsApp config UI beyond reading template approval status for display.
- No new backend endpoints invented for the wizard without going through changes 1/2.
- No multi-language UI (Spanish only, matching the CRM's existing Spanish-first conventions).
