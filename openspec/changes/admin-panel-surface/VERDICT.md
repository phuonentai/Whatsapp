# Council Verdict — admin-panel-surface (re-review, revision 2)

STATUS: APPROVED
MARKET: PASS

## Market Read

Re-review of `design.md` rev. 2 (`revision.md` passes 1–2). Both round-2 required changes are resolved:

- **Required change #1 (Siigo write/read-only contradiction) — RESOLVED by product decision:** two panels with a strict boundary. Tenant panel (`/dashboard/settings`) keeps Siigo (`siigo-admin-view` org-scoped, role-`admin` gate, unchanged — consistent with the living `admin-panel-navigation` "Admin onboarding overview" requirement and with `settings-redesign`, which restyles it in settings); platform panel (`/admin`) exposes NO Siigo data (no section, no connection state, no credentials, no invoicing). Enforced at query level: no `/api/v1/platform/*` route reads Siigo tables/fields, with regression tests asserting absence of Siigo fields in platform responses (tasks 2.2/3.5, delta requirement "Siigo fuera de alcance de la plataforma"). The contradiction, the secrets-handling concern, and the `settings-redesign` conflict are all gone. ✓
- **Required change #2 (access-log operational state) — closed except one recorded decision:** 90-day retention cleanup job + `(created_at)` index tasked (1.3); write-failure policy (fail-open vs fail-closed) remains as an explicit open question with a decision point at task 2.8 — a recorded residual, not a blocker. ✓

Market read: unit economics "no delta" remains correct (read-only aggregation of the existing `ai_usage` ledger; no new LLM calls, no pricing/billing surface; model-rate table is reference-only). The compliance surface **shrank** with the Siigo decision: platform operators now touch only usage/ops state, never customer invoicing data or credentials. Residuals R1 (owner: security, trigger: access incident, mitigation: 403/503 gate + bounded scope + `platform_access_log`), R2 (owner: product, trigger: operator requests), R3 (owner: product/backend, trigger: model/price change) are recorded with owners and triggers. **PASS** — no market conditions remain open.

## Findings by persona

### 1. Staff Security Engineer

- ✓ Siigo contradiction resolved (no cross-org credential writes; platform never touches Siigo tables). Tenant isolation is well-specified: platform principal without tenant (Decisions 2/7), `org_id` validated against `organizations` (400/404), member-scoped endpoints never serve cross-org data, 403/503 fallback contract per `stytch-authorization`, regression tests in both directions (2.7). Stytch grounding correct: sessions always org-scoped → dedicated `platform-ops` org; `authorization_check` correctly rejected for cross-org reads (org-coupled).
- **[LOW] S1 — JWT roles-claim revocation latency.** A banned/role-removed operator's JWT retains `platform_admin` until re-auth (5-min policy cache doesn't change member-role resolution). Consistent with the existing member surface, but the rollback docs should include the Stytch ban + session revocation procedure. **Residual to record in revision.md.**

### 2. Staff DBA

- ✓ No ALTERs/locks: `platform_access_log` is additive with a retention index; aggregation over `ai_usage`/`ai_usage_events` validated in spike 1.1 (period-first index coverage, pagination mandatory, expand-contract path if an index is needed); transaction boundaries N/A (read-only).
- **[LOW] D1 — Audit-view "eventos operativos" source undefined.** The cross-org audit lists `ai_usage_events` + "actividad operativa (estado de conexiones/suscripción)", but no event source for connection/subscription changes is named. v1 SHALL scope the audit view to `ai_usage_events` + `platform_access_log` events; a connection/subscription-change event source is a follow-up decision. **Residual to record.**

### 3. SRE

- ✓ Rollback clean (git revert + Stytch policy adjustment; audit-only DB state); breaker/fallback states defined for the RBAC policy (403/503) and external subscription state ("—" degradation); idempotency/distributed locks N/A (read-only).
- **[LOW] SRE1 — `platform_access_log` write-failure policy.** Open question at task 2.8 (fail-open with structured error log + alerting on sustained failure vs fail-closed). Acceptable as a recorded residual with a defined decision point. **Residual to record.**

### 4. Staff Product/GTM

- ✓ No unit-economics delta (verified); pricing tiers, credit guard, and plan coherence untouched. The two-panel split matches the audience reality: tenants manage their own domain (incl. Siigo), operators get operational oversight only.
- **[LOW] P1 — Passive value.** Uso IA view is monitoring-only until alerting lands (design records "alertas de uso" as follow-up); product assumption to validate. **Residual to record.**

### 5. Colombia IT & Market

- ✓ Siigo fully out of platform scope removes the Ley 1581/credential-handling concern for invoicing data; cross-org surface bounded to operational/usage state (purpose limitation); 90-day access-log retention; no new retention windows; WhatsApp channel display-only (no policy-drift surface); no DIAN/invoicing contract change. Living-spec and ROADMAP/GAP-ANALYSIS coherence verified (no contradictions).

## Residual risks to record in revision.md (approved — no required design changes)

1. S1 — JWT roles-claim revocation latency: Stytch ban + session revocation procedure in rollback docs.
2. D1 — v1 audit-view scope: `ai_usage_events` + `platform_access_log` events only; connection/subscription-change event source deferred (follow-up).
3. SRE1 — `platform_access_log` write-failure policy: decision at task 2.8 (fail-open vs fail-closed) with observability/alerting.
4. P1 — Passive Uso IA value until alerting lands (product assumption).
5. R2, R3 — unchanged accepted residuals (owners/triggers in design.md `## Market Risk`).

Design is coherent, grounded, and within the bounded revision budget. Ready to proceed past the council stage.
