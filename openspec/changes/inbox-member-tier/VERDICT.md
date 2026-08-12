STATUS: APPROVED
MARKET: PASS

# Council Verdict — inbox-member-tier

**Change:** Inbox v2 — tiers de acceso (miembro lee/responde) + pulido del AI rail
**Scope:** Market-in-scope (`requires_market_read: true`; WhatsApp/Meta channel + authorization contract + compliance surface)
**Review basis:** design.md, proposal.md, tasks.md, delta specs (stytch-authorization, whatsapp-inbox, inbox-ui, agent-governance), living specs (stytch-authorization, whatsapp-inbox, whatsapp-compliance, agent-governance, ai-usage-metering, paywall), code evidence (`app/dashboard/inbox/page.tsx` `org:manage` gate, `sidebar.tsx` gate, `rbac.go` roles/permissions, `crm/routes.go` export gates).
**Prior verdict / revision:** none — first review.

## Verdict

**APPROVED.** No REJECT-level defects. The permission-tier change is correctly designed as a Stytch runtime-SSOT contract (new `inbox:view`/`inbox:reply` permissions) with server-side enforcement, unchanged guardrail path for member sends, and dual rollback (Git + Stytch policy) — satisfying the governance rule for auth-flow changes. Design and delta specs are mutually coherent, both mandatory market sections are present, and all residuals carry named owners/triggers. Findings below are implementation obligations, not blockers.

## Per-Persona Findings

### 1. Staff Security Engineer

- **LOW — Enumerate all conversation/message read paths (residual).** `inbox:view` expands PII read surface (WhatsApp/Instagram conversations) to members. The design scopes to org tenancy and relies on server-side 403 — correct — but the tasks SHALL add a spike grepping every endpoint that can expose conversation/message content (search endpoints, contact-detail last-message, message attachments/media URLs, exports). Note: CSV export endpoints are gated on `contact:export`/`deal:export` (verified in `crm/routes.go`), so a member with only `inbox:view` cannot exfiltrate via CSV — preserve this boundary.
- **PASS — No local credential storage; auth remains Stytch B2B (runtime SSOT).** Matches the `stytch-authorization` living contract (policy resolved from Stytch, 5-min Redis cache, 503 on unavailable); the change does not weaken it.
- **PASS — Member sends are guardrail-governed:** same `send_message` snapshot (kill switch, discount cap, forbidden/escalation terms, consent, window, daily limit) with denials in audit; no role exemption — coherent with `agent-governance` delta spec and living spec.
- **INFO — 5-min policy cache staleness:** a member could transiently see admin controls right after a policy change; server-side 403 is the backstop (design correctly emphasizes it). Record rollout/rollback sequencing in the migration plan.

### 2. Staff DBA

- **PASS — No data model change, no migration.** Policy lives in Stytch; nothing to expand/contract in PostgreSQL. No index/N+1 concerns introduced.

### 3. SRE

- **PASS — Rollback is dual and documented** (Git revert + Stytch policy restore to `org:manage` gate), satisfying the Stytch-state rollback governance rule. No new outbound calls, no circuit-breaker delta.
- **PASS — Poll (5s) unchanged; live-region announces increments only** — no a11y spam risk.
- **INFO — Stytch policy propagation delay:** both deploy and rollback take effect within the 5-min RBAC cache TTL; call this out in the migration plan so a rollback is understood as near-term, not instant.

### 4. Staff Product/GTM

- **PASS — Unit economics coherent:** no new cost per message (same path/guardrails/metering); the tier widens *who* can respond, not what responding costs. No pricing/plan/credit delta (verified against `ai-usage-metering`, `paywall`).
- **PASS — 402-only-for-IA is a strong retention decision:** manual WhatsApp replies stay live at zero credits, protecting the core business workflow from credit exhaustion — good activation/churn surface hygiene.
- **PASS — Human-in-the-loop preserved:** approve = prefill, escalation amber human-only, sequences advance only on successful sends and never auto-send — coherent with the copilot philosophy and defensible against Meta AI-agent term drift (residual R2 with owner/trigger).
- **INFO — Competitive substitution is thinner here** (Meta native AI, local WhatsApp CRM incumbents addressed only via R2 drift framing); acceptable for a permission-tier change, but the market read should note that member-tier access is a parity feature, not a differentiator.

### 5. Colombia IT & Market

- **PASS — Ley 1581 coherence:** consent-withdrawn contacts render a structural context card (never pre-generated facts), consistent with `whatsapp-compliance` (withdrawn → no autonomous sends, drafts fall back to human review); PII masking before third-party AI calls unchanged.
- **INFO — Read-surface expansion is the compliance crux:** more members can now read conversation PII; R1 names security/architecture as owner with a clear trigger (access incident, audit, or Stytch role change) and audit records the sending actor — acceptable as an accepted residual.
- **PASS — WhatsApp Business Platform policy drift is explicitly a channel-existential risk** (R2: conversation pricing, AI-agent terms) with owner product/ops, trigger term change or autonomy-claiming copy, and mitigation (copy without autonomy claims; prefill keeps human review).

## Market Read

The change is in-scope because it modifies the WhatsApp/Meta channel operating surface (who may read/reply) and the authorization contract (new Stytch permissions — runtime SSOT), not pricing. Cost math is neutral: same send path, guardrails, and metering for member sends; no plan/price/credit delta (verified against `ai-usage-metering` and `paywall`). Market upside: more operators can respond → faster response times and higher operational capacity for the WhatsApp inbox at zero marginal cost; the 402-only-for-IA rule keeps the revenue-critical manual reply path alive during credit exhaustion. Channel risk is the dominant exposure — Meta conversation pricing and AI-agent terms are recorded as an accepted residual (R2, owner product/ops, trigger term change or copy drift), with human-in-the-loop prefill as the standing mitigation; RBAC-read-expansion compliance risk is accepted with owner and trigger (R1). No unverified external fact is asserted as a premise. Residuals are accepted and tracked: **MARKET: PASS**.

## Implementation Obligations (from findings above)

1. Add a task spike to enumerate every conversation/message read path (search, contact-detail preview, media URLs, exports) and confirm `inbox:view` scoping on each; keep CSV exports gated as today.
2. Document the 5-min Stytch RBAC policy-cache TTL in the migration/rollback plan (deploy and revert are near-term, not instant).
3. Extend `AllPermissions`/local permission constants with `inbox:view`/`inbox:reply` where the codebase mirrors Stytch permissions, keeping Stytch policy as the authoritative source.
4. Verify member-role assignment (`member` exists with assignable permissions) in spike 1.1 as designed, and record the policy change + rollback commands in tasks.
