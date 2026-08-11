## Why

The invoicing rail exists (`add-siigo-invoicing`): a deal moved to `facturado` creates a Siigo invoice and notifies the customer on WhatsApp. But organizations cannot connect their own Siigo account: Siigo credentials are resolved from environment configuration only, which supports exactly one client account platform-wide. Onboarding a client into invoicing is impossible today, and a non-connected organization silently fails at the `facturado` stage. This change makes per-organization Siigo connection real: encrypted per-org credential storage, an explicit `siigo | none` provider state, an onboarding state machine, and deal-stage gating so invoicing never fires for orgs that are not `live`.

## What Changes

- **Per-org Siigo connection**: new `invoicing.org_connections` table storing the organization's connection state, Siigo company data, and encrypted API credentials (AES-256-GCM envelope, master key from env). One row per organization.
- **Provider `none` state**: `InvoiceRouter` gains an explicit `none` provider state — a non-connected org routes to a no-op, NOT a fail-closed error. Unknown providers still fail closed.
- **Onboarding state machine**: `none → awaiting_setup | connected → numeracion_ok → sandbox_ok → live`, plus terminal `invoicing_disabled` and reversible `paused`. Transitions guarded; every state visible to clients and operators.
- **Connect endpoint**: `POST /api/v1/org/siigo/connect` — validates credentials against Siigo live (sandbox-respecting), verifies the Siigo company NIT matches the org's NIT, stores encrypted credentials, advances state to `connected`.
- **Assisted setup path**: admin-scoped endpoint writes credentials on behalf of a client (org stuck at `awaiting_setup` until provisioned) — for no-Siigo clients sold Siigo as a service.
- **Deal-stage gating**: `DealStageListener` consults the connection state before invoice creation. State != `live` → no provider call, deal activity "Facturación no activa" recorded, WhatsApp sends payment-only flow.
- **Kill-switch**: `pause`/`resume` endpoints allowing immediate invoicing suspension without code changes.
- **Credential governance deviation (explicit)**: `add-siigo-invoicing` declared "no local credential storage". Per-org Siigo credentials are third-party integration secrets, not user passwords/MFA/session tokens — the Stytch boundary is untouched (no local identity, no session material). They SHALL be stored AES-256-GCM encrypted at rest (envelope key in env), never plaintext, never in logs, never returned by any API. This deviation is intentional and documented in the Non-Goals and design Decisions.

## Capabilities

### New Capabilities
- (none — connection behaviour is a requirement of the existing `invoicing` capability)

### Modified Capabilities
- `invoicing`: provider resolution gains an explicit `none` state; organizations SHALL hold an onboarding state that gates deal-stage invoice creation; Siigo credentials SHALL be stored encrypted per-org; connect/NIT-verification and kill-switch behaviours become spec-level requirements.

## Impact

- **Go backend**: new migration `000028` (`invoicing.org_connections` + SQLC queries); new `infra/secrets` envelope-encryption package; `InvoiceRouter` extended with `none` state; `DealStageListener`/`InvoicingService.CreateForDeal` gated on connection state; new `ConnectionService` + `POST /api/v1/org/siigo/{connect,pause,resume}` + `GET /api/v1/org/siigo/status`; admin-scoped assisted-provisioning endpoint; DI wiring in `cmd/provider.go` / `bootstrap/init_mods.go`. Siigo adapter and token cache unchanged.
- **Database**: new table `invoicing.org_connections`; no changes to existing tables; no Stytch identity data stored (org references remain `stytch_organization_id`-derived `organization_id` only).
- **Auth boundary / Stytch**: no change to Stytch contracts, RBAC roles, sessions, or JWKS verification. Connect/status endpoints use existing org-scoped auth; assisted-provisioning uses an admin role check on the existing RBAC mapping. Credential encryption is app-layer, outside the Stytch domain.
- **Frontend**: none required for this change (the wizard is a separate change). Status endpoints are consumed later.
- **Rollback strategy**: Git — revert the change commit(s). DB — down migration drops `invoicing.org_connections`; existing invoices and webhook behaviour unaffected. Stytch tenant policy state — unaffected (no RBAC/auth changes), no Stytch-side rollback. Credential exposure — on rollback, encrypted rows are dropped; the env master key can be rotated without touching Siigo (credentials are Siigo-side secrets, not revocable via our infra).

## Non-Goals

- **No plaintext credential storage**: credentials SHALL NOT be stored, logged, or transmitted in plaintext; encryption key lives in environment only; API responses SHALL NEVER return secrets.
- No numbering/continuity work (separate change `add-siigo-onboarding-data`).
- No customer/product import (separate change).
- No onboarding wizard UI (separate change).
- No new invoicing provider adapter (Alegra remains an interface slot only).
- No changes to Siigo OAuth token cache, webhook handler, or WhatsApp send path.
- No changes to the Stytch identity boundary; no new local user/identity tables.
