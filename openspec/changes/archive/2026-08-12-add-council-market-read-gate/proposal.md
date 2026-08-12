# Add Council Market-Read Gate — Proposal

## Why

The council reviews every gated design through Security / DBA / SRE lenses only — it validates engineering correctness, never market viability. A 2026 AI-first SaaS for the Colombian market wins or loses on judgment the council currently lacks: a **SaaS expert** who understands product-market fit, unit economics (USD AI cost per action vs what a Colombian SME will actually pay), pricing and packaging, activation and churn — and a **Colombian IT/market expert** who knows the local reality: how businesses really run WhatsApp, the payment and invoicing ecosystem (PSE/Nequi, DIAN electronic invoicing), Ley 1581/Habeas Data, consumer law, and the risk of depending on Meta's channel. Tools like SIIGO are just tools — they come and go; what matters is the expertise that navigates the ecosystem. Neither lens alone is enough: the value is the two of them balancing each other out — the SaaS expert keeps the product commercially sane, the Colombian expert keeps it locally true. Today neither perspective exists in any review gate, even though `GAP-ANALYSIS-2026.md` ranks the market-facing gaps (trust, observability, onboarding, marketing) as the highest severity.

## What Changes

- **Expand the council to five personas** in `.pi/prompts/council.md`: the existing Staff Security Engineer, Staff DBA, and Staff SRE are joined by **Staff Product/GTM — the SaaS-expert lens** (unit economics, pricing coherence, activation, competitive substitution) and **Colombia IT & Market — the local-expert lens** (local regulation and consumer law, the payment and invoicing ecosystem — tools like SIIGO are examples, not the point — and WhatsApp/Meta channel dependency). The two new lenses balance each other: the SaaS expert keeps the product commercially sane, the Colombian expert keeps it locally true. Security/DBA/SRE methods and prohibitions are unchanged.
- **Add a machine-parseable Market Read deliverable** to `VERDICT.md` for market-in-scope changes: a `MARKET: PASS | CONDITIONAL | FAIL` line plus prose findings. The existing `^STATUS: (APPROVED|REJECTED)$` first-`STATUS:`-line marker contract is **NOT** changed (`parse_verdict`, exit codes, and fixtures untouched).
- **Couple the market verdict to the STATUS verdict**: for in-scope changes, a `MARKET: FAIL` that is not mitigated by an explicitly accepted residual in the design SHALL NOT accompany `STATUS: APPROVED` — the council SHALL issue `REJECTED` (or the design records the accepted risk). The market lens therefore gates through the existing verdict mechanism instead of a new pipeline gate.
- **Require market sections in `design.md`** for in-scope changes: a `## Market & Unit Economics` section (cost-per-AI-action vs price point, plan/fee margins) and a `## Market Risk` section (channel/regulatory dependencies, competitive substitution). Absence of these sections for an in-scope change is itself a design defect the council SHALL flag (REJECT-level when unmitigated).
- **Advisory routing extension**: `routing.json` MAY declare `requires_market_read: true`; when set, `council_required()` returns true regardless of `complexity`, so a medium-complexity pricing/billing change still gets the council's market lens. Advisory semantics are unchanged (it never overrides OpenSpec artifacts or lifecycle gates).
- **Traceability**: the iso stage records the `MARKET` line and key market risks in `docs/compliance/ISO_TRACEABILITY_MATRIX.md`; `AGENTS.md` documents the personas and the gate.

### Non-Goals

- **No change to the council marker contract**: `^STATUS: (APPROVED|REJECTED)$` as the first `STATUS:`-prefixed line, `parse_verdict`, exit codes 1/2/3, and existing fixtures are untouched.
- **No new OpenSpec schema artifact or lifecycle gate**: market-read is review-time behavior inside the existing council stage; no `openspec` CLI change, no `apply.requires` change.
- **No separate market pipeline stage**: the loop/revision machinery belongs to the sibling `use-inout-council-for-a-better-design` change; this change stays orthogonal (personas + deliverable + routing field + traceability).
- **No local credential storage, no auth-flow or data-persistence changes**: this change touches `.pi/prompts/`, `scripts/pipeline.sh` gating, `AGENTS.md`, and `docs/compliance/` only.

## Capabilities

### New Capabilities
- *(none — this extends the existing `native-agent-pipeline` capability; no new capability file)*

### Modified Capabilities
- `native-agent-pipeline`:
  - "Prompt-template agent stages exist" — the council stage's persona set grows from three (security/DBA/SRE) to five (adding Product/GTM and Colombia IT & Market), with a market-read review applied to market-in-scope changes.
  - "Council verdict marker contract" — `VERDICT.md` gains a required `MARKET: PASS|CONDITIONAL|FAIL` line and Market Read prose for in-scope changes; the STATUS marker contract and its parsing are unchanged; for in-scope changes a `MARKET: FAIL` without an accepted residual is incompatible with `STATUS: APPROVED`.
  - "Advisory routing.json sidecar" — the optional `requires_market_read` field is added; when true, the council stage runs regardless of `complexity`.

## Impact

- `.pi/prompts/council.md` — two new personas, Market Read deliverable section, `MARKET:` line, design.md market-section requirement.
- `scripts/pipeline.sh` — `council_required()` honors `requires_market_read`; help/usage text; no other pipeline behavior changes (loop machinery is the sibling change's scope).
- `.pi/prompts/iso.md` — records `MARKET` line + market risks in the traceability matrix.
- `AGENTS.md` — Agent Pipeline section documents the five-persona council and the market-read gate.
- `docs/compliance/ISO_TRACEABILITY_MATRIX.md` — market-read outcomes recorded by the iso stage.
- No application source code, no DB migrations, no auth/billing/webhook behavior, no Stytch contract changes.

### Assumptions (unverified external facts — validate before asserting)

- WhatsApp Business Platform conversation pricing, template categories, and Meta's native AI messaging features change over time and can shift channel economics; the specific figures are NOT asserted here or in the council — the personas treat channel terms as a risk surface to be validated, per the repo's premise-validation gate.
- Colombian regulatory posture (Ley 1581 reform trajectory, DIAN invoicing resolutions, Ley 1480) evolves; the personas flag regulatory dependencies as risks, not facts.
