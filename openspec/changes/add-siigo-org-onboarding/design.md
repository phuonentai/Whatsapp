## Context

`add-siigo-invoicing` delivered the rail: `InvoiceRouter` (default Siigo, fail-closed), Siigo adapter with in-memory OAuth token cache (TTL 300s, env-only credentials), `DealStageListener` triggering `CreateForDeal` on stage `facturado`, webhook + polling status sync, and the `invoicing.invoices` table (migration `000021`). Repo migration head is `000027` (`add_webhook_logs_delivery_key`); next free number is `000028`.

Gap: credentials come from env only → exactly one Siigo account platform-wide. No per-org connection, no onboarding, no gating — a non-connected org reaching `facturado` would attempt a provider call with the platform's single credential set and fail.

Existing precedent: `organizations.billing_provider` column (`gen/models.go:534`) shows the per-org provider pattern. `FeatureService` + `features.Require` middleware exist but feature flags are plan-derived; invoicing is not flag-gated and should not be (it is an integration, not a plan feature). The Stytch boundary is untouched: org identity stays Stytch-derived, no local identity material.

## Goals / Non-Goals

**Goals:**
- Per-org Siigo connection with encrypted credential storage (AES-256-GCM envelope, key in env)
- Explicit `siigo | none` provider states; `none` is a deliberate no-op, unknown still fails closed
- Onboarding state machine with guarded transitions; every state visible to clients and operators
- Deal-stage gating: invoicing fires only when state is `live`
- Self-serve connect (NIT verified) + admin-scoped assisted provisioning
- Kill-switch pause/resume without code changes

**Non-Goals:**
- No numbering/continuity work (change `add-siigo-onboarding-data`)
- No customer/product import (change `add-siigo-onboarding-data`)
- No wizard UI (change `add-siigo-onboarding-wizard`)
- No new provider adapter (Alegra remains a slot)
- No Stytch changes; no plaintext secrets anywhere
- No changes to the Siigo OAuth token cache or webhook handler

## Decisions

### 1. Encrypted per-org credential storage (governance deviation, explicit)

**Chosen:** `invoicing.org_connections` stores `client_id_enc` / `client_secret_enc` as AES-256-GCM ciphertext (nonce included), envelope key from `SIIGO_MASTER_KEY` (base64, env only). New `infra/secrets` package owns encrypt/decrypt; the adapter resolves credentials through it; nothing logs or returns secrets.

**Alternatives considered:**
- *Env-only per org (operator restarts with org-specific vars)* — rejected: N clients = N env pairs, no self-serve, unworkable at scale.
- *External vault (Vault/KMS call per connect/invoice)* — rejected for MVP: adds infrastructure + network dependency on the hot path; the envelope pattern is KMS-migratable later (swap master key source without schema change).
- *Plaintext column* — rejected: violates Non-Goals and governance guardrails.

**Governance note:** `add-siigo-invoicing` Non-Goal "no local credential storage" targeted user credentials (Stytch domain). Siigo client credentials are third-party integration secrets; the Stytch identity boundary is untouched (no passwords/MFA/session material stored). The exception is documented in the proposal's Non-Goals and this decision; encryption-at-rest + env key + no-log invariant satisfies the intent of the rule.

### 2. Provider `none` as explicit state, not fail-closed error

**Chosen:** `InvoiceRouter.Resolve` returns a `NoopProvider` for provider `none` (no call, informative result — no error), keeps fail-closed for unknown values. `org_connections.provider` is the single source; `organizations.billing_provider` is untouched (billing domain).

**Alternatives considered:**
- *Keep fail-closed for everything* — rejected: a `facturado` deal in a non-connected org would error the whole flow; silent-ish failures at sale close are the worst outcome.
- *New `InvoicingProvider` defaulting to no-op* — rejected: implicit no-op hides misconfiguration; explicit `none` state makes the contract testable and visible.

### 3. State machine enforced in service, single write path

**Chosen:** `ConnectionService` owns all transitions via a `nextState(current, action)` table (pure function, unit-tested). Status stored on `org_connections.status`. No free-form updates; repository exposes only state-guarded transitions.

```
none ──────────► awaiting_setup ──(admin provisions)──► connected
  │                                                     │
  └───────────(self connect)───────────────────────────► connected
                                                          │
connected ──► numeracion_ok ──► sandbox_ok ──► live
                                                    │
invoicing_disabled (terminal)                        └──(pause)──► paused ──(resume)──► live
```

**Alternatives considered:**
- *JSONB status blob on organizations* — rejected: `org_connections` keeps connection data cohesive and keeps `organizations` untouched (mirrors the `invoicing.invoices` scoping pattern).
- *No guard, trust callers* — rejected: gating is a legal-adjacent invariant; single guarded path is testable.

### 4. Gating in `DealStageListener` (not in service)

**Chosen:** `DealStageListener` (invoicing module, `app/services/deal_listener.go`) checks connection state before calling `InvoicingService.CreateForDeal`. Not live → record activity "Facturación no activa" (existing deal-activity path), skip provider, no WhatsApp invoice message; payment-only WhatsApp flow still proceeds via existing send path.

**Alternatives considered:**
- *Gate inside `CreateForDeal`* — rejected: service-level gating hides the decision from the listener contract and complicates idempotency tests; listener-level check keeps `CreateForDeal` provider-faithful.
- *Feature flag via `FeatureService`* — rejected: plan-derived flags are the wrong axis (invoicing is integration config, not tier entitlement); state machine already exists.

### 5. Connect validation: credentials + NIT, sandbox-aware

**Chosen:** `POST /api/v1/org/siigo/connect` runs: (1) OAuth token fetch against configured base URL (sandbox flag respected), (2) `companies` lookup via existing adapter surface, (3) NIT comparison vs org company NIT (normalize digits/format), (4) persist encrypted + `connected`. Any failure → error, no persistence, no state change.

**Alternatives considered:**
- *Skip validation, trust creds* — rejected: a typo'd credential set poisons the org state and confuses onboarding.
- *NIT match advisory only* — rejected: cross-company invoice creation is a legal risk; mismatch is hard failure.

### 6. Assisted provisioning: same service, admin-scoped route

**Chosen:** `POST /api/v1/admin/siigo/provision` (admin RBAC role check on existing mapping) writes credentials through `ConnectionService` for orgs in `awaiting_setup`. Same validation as self-serve (minus interactive UX). No second code path for storage.

**Alternatives considered:**
- *Ops edits DB directly* — rejected: bypasses encryption path and state guards.
- *Client-side only* — rejected: no-Siigo clients sold Siigo as a service need operator entry.

### 7. Pause/resume as state, not column flag

**Chosen:** `paused` is a first-class reversible state; pause from `live`, resume → `live`. Kill-switch endpoints `POST /api/v1/org/siigo/pause|resume`. Polling fallback and webhook continue to run (status sync must not stop during pause — invoices already issued still resolve).

**Alternatives considered:**
- *Boolean `paused` column* — rejected: dual-source truth with the state machine; state-only keeps guards uniform.

## Risks / Trade-offs

- [Master key compromise → credential exposure] → Mitigation: key in env, never in repo/CI logs; rotate by updating env + re-encrypt task; Siigo creds remain revocable at Siigo side (client can reset their API key).
- [Ciphertext at rest adds decryption on connect/invoice paths] → Mitigation: decryption is AES-GCM, microseconds; only token acquisition path decrypts (cached token covers steady state).
- [State machine complexity grows with numbering/import steps] → Mitigation: transitions added by `add-siigo-onboarding-data` follow the same guarded single-write pattern; machine is a pure table.
- [Assisted setup abused by non-admins] → Mitigation: admin RBAC check enforced at route + service level; audit-log event on provisioning (audit-log capability exists).
- [NIT normalization mismatch (punctuation/digits)] → Mitigation: normalize digits only (`\D` stripped), compare; documented; manual override deferred.

## Migration Plan

1. Migration `000028_create_org_connections` (+ down): `invoicing.org_connections` with `provider`, `status`, `client_id_enc`, `client_secret_enc`, `nit`, `siigo_company_name`, `last_error`, `paused_at`/timestamps; `UNIQUE(organization_id)`. `make sqlc` regenerates.
2. `infra/secrets` package (encrypt/decrypt/validate master key) + unit tests (round-trip, tamper, no-log invariant).
3. `ConnectionService` + state machine + `NoopProvider` + router extension + tests.
4. Connect/status/pause/resume routes + assisted provision route (admin) + DI wiring.
5. Gate `DealStageListener`; extend existing listener tests for non-live states.
6. Gate: `make sqlc`, `go build ./...`, `go vet ./...`, `go test ./...`.
7. Rollback: revert commit(s); down migration drops `org_connections`; no Stytch state to roll back; existing invoices/webhooks unaffected.

## Open Questions

- Where does the assisted-provisioning admin route live — new admin route group or existing admin surface? (resolved at implementation against repo admin precedent; no spec impact)
- Should `SIIGO_MASTER_KEY` rotation support multiple active keys (key-id prefix) now or later? (later; single key MVP)
- Does pause need a reason field surfaced in settings/admin views? (deferred to wizard change if needed)
