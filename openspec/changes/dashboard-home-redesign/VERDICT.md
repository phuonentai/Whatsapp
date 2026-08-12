# Verdict: dashboard-home-redesign (re-review, Revision 1)

STATUS: APPROVED
MARKET: CONDITIONAL

## Market Read

The revision resolves the previous REJECT (missing market sections for an in-scope change). `design.md` now carries both mandated sections, and the five numbered required changes are verifiably addressed in the artifacts.

**Top market findings:**

1. **Unit economics delta is zero and now stated as such.** `## Market & Unit Economics` explicitly declares: no new AI actions, no LLM invocation, no model routing, no metering (`ai-usage-metering` untouched); no pricing/plan/credit change (`paywall`, `plan-pricing-ux`, `billing-quota-integrity` untouched); Polar/MercadoPago verification in `app/dashboard/page.tsx` preserved byte-for-byte (independently re-verified in code: `checkout_id`/`payment_id`/`preapproval_id` branches with their redirects). All widgets reuse existing queries via TanStack cache with no fan-out. This is the correct content for a frontend-only change — cost-per-AI-action vs COP price point is moot because no AI action is added, and the design says so explicitly rather than implying it.
2. **Channel risk is the dominant accepted residual.** R1 (WhatsApp Business Platform policy drift — AI-agent terms, conversation pricing, template approval) is recorded with owner (product/ops), trigger (Meta terms/pricing change, or banner copy asserting autonomous mode without model confirmation), and design mitigation (banner renders as static suggestion when `mode` is unavailable; copy lives in `lib/copy/ui.ts` with no autonomy claims). This is the right posture for the only existential risk surface in the change.
3. **Activation posture is preserved.** R2 records the checklist-fold regression risk with owner (product) and trigger (activation/completion metric drift); the design pins folding-default only when the checklist is complete and never hides an incomplete checklist at first use — consistent with the verified current behavior (`FirstRunChecklist` derives completion from existing queries and already returns `null` when complete).
4. **Competitive substitution is acknowledged, not dodged.** R3 (perceived-value gap vs the numbers-dense mockup — Meta native WhatsApp AI, local WhatsApp CRM incumbents) is accepted as a deliberate honest-data differentiator, mitigated by CTAs, with owner (product/GTM) and trigger.
5. **Invoicing-ecosystem expectation gap is bounded.** R4 (Facturas Siigo empty state vs user expectation of invoice data) is accepted with owner (product/backend) and trigger, scoped to the future backend list endpoint — no DIAN/data flow in this change.

**Conditions for the CONDITIONAL marker** (each is an accepted residual recorded in `design.md` `## Market Risk`, not an open design gap):
- C1 — R1 channel/agent-claims drift tracked by product/ops on the stated trigger.
- C2 — R2 activation regression tracked by product on the stated trigger.
- C3 — R3 perceived-value gap tracked by product/GTM on the stated trigger.
- C4 — R4 invoice-data expectation tracked by product/backend on the stated trigger.
- C5 — Apply-time verification of the two LOW findings below (completed-checklist shell rendering vs the living `ai-onboarding` "disappear" wording; conversations-widget gate note) folded into tasks 1.9/1.11/2.3, which already carry the verification commands.

## Persona Findings

### 1. Staff Security Engineer
- **INFO — No auth surface changed.** Stytch B2B contracts untouched; no credential storage; payment verification branches verified intact in `page.tsx`.
- **INFO (verified, non-issue) — Conversations-widget gate is not a widening.** The inbox route applies a client-side `ORG_MANAGE` redirect (`app/dashboard/inbox/page.tsx`), but the backend `ListConversaciones` handler authorizes by authentication + org scope only (no permission check in the handler; org-scoped SQLC query), and `dashboard-home.tsx` already calls `useConversationsQuery` today. The design's D8 statement (widget condition = org-scoped query access; ORG_MANAGE redirect stays on the inbox route) is accurate: the home panel displays data the member is already authorized to fetch and the home already fetches. Recommended residual note (not blocking): document the inbox-route redirect as UX-only so a future backend permission hardening doesn't silently contradict the home widget.
- **PASS — RBAC inheritance pinned.** Delta spec scenario "Widgets heredan el gate de su superficie fuente" + design D8 table + task 1.11; "ningún widget SHALL exponer datos a un miembro que no tenga el permiso de su módulo fuente; si no hay gate explícito → estado vacío honesto".
- **PASS — PII minimization pinned.** Snippet-only, no full bodies/thread/export on home; both delta specs + design D9 + tasks 1.2/2.3.

### 2. Staff DBA
- **N/A — No database impact.** No migrations, no SQLC, no schema, no new queries beyond existing query reuse; TanStack cache prevents refetch fan-out. Nothing to lock, index, or expand-contract.

### 3. SRE
- **INFO — No new failure modes.** Read-only display change; no endpoints, no idempotency keys, no distributed locks, no new external-provider calls; rollback is git revert with no DB/Stytch state. The only new persistence is a localStorage fold preference (client-only, self-cleaning).
- **LOW — Broadcast route checkpoint is correctly sequenced.** Task 1.6 carries a decision checkpoint (no `/dashboard/campaigns` or `view=campaigns` exists today; `use-campaign-queries.ts` exists) with the D2.2 omit fallback, so the apply stage cannot silently drop a designed action.

### 4. Staff Product/GTM
- **PASS — Unit economics section present and correct.** Zero-delta declaration is verifiable; activation metric correctly scoped as a follow-up rather than introduced ad hoc.
- **PASS — Residuals are governance-complete.** R1–R4 each carry named risk, owner, and trigger, satisfying the MARKET/STATUS coupling contract.
- **LOW — Completed-checklist shell vs living spec.** The living `ai-onboarding` spec says the checklist "SHALL disappear once all steps are complete"; design D5 introduces a minimal collapsed shell when complete. The delta spec's MODIFIED requirement ("disponibles... plegados en un patrón colapsable") supersedes on archive, but the exact completed-state rendering (shell vs nothing) should be pinned during apply to avoid a stale-spec contradiction at archive time. Covered by task 1.9 verification; no design change required.
- **LOW (advisory) — routing.json** carries `requires_council: true` (which per AGENTS.md forces the council regardless of complexity) but lacks the `requires_market_read: true` key for the market-in-scope class; advisory only, market read is present in `design.md`.

### 5. Colombia IT & Market
- **Ley 1581/Habeas Data — PASS.** Snippet-only minimization (same fields as the inbox list), no new transfer/export/retention surface; consistent with `data-transfer`/`data-backup-recovery`; verified against the conversations API surface (org-scoped list, messages remain inbox-only).
- **DIAN/invoicing — PASS.** Siigo widget moves no invoice data; empty state + CTA; R4 bounds the expectation gap until the future backend endpoint.
- **WhatsApp Business Platform — PASS with R1 residual.** Read-only display change; no template/conversation-pricing surface touched; R1 owns the drift surface (banner claims) with trigger and mitigation.
- **Ley 1480 consumer law — N/A** (B2B authenticated dashboard; no consumer claims).

## Conclusion

All five required design changes from the prior verdict are addressed and verifiable in the artifacts: market sections present (`## Market & Unit Economics`, `## Market Risk` with R1–R4 named/owner/trigger), per-widget RBAC inheritance pinned in the delta spec, PII snippet-only boundary pinned in both delta specs, and component attribution corrected (recomposition → `DashboardHome`; `page.tsx` → verification only). Residual risks are accepted and owned per the coupling contract. Approved with the five recorded conditions (C1–C5) tracked by their named owners.
