# Council Market-Read Gate — Design

## Context

The council stage (`.pi/prompts/council.md`) reviews `design.md` + `proposal.md` in three personas — Staff Security Engineer, Staff DBA, Staff SRE — and writes `VERDICT.md` with a `^STATUS: (APPROVED|REJECTED)$` first-`STATUS:`-line marker that `parse_verdict` (scripts/pipeline.sh) consumes; `council_required()` gates the stage via `WITH_COUNCIL`, `routing.json .requires_council`, and `.complexity == "high"`. The review is engineering-correctness-only: nothing checks unit economics, pricing coherence, WhatsApp/Meta channel dependency, or Colombia-specific compliance. `GAP-ANALYSIS-2026.md` already ranks the market-facing gaps (trust, observability, onboarding, marketing) as highest severity, and ROADMAP Q2 defers evals/cost control for the AI stack — the "AI-first 2026" differentiator is the least-reviewed surface. This change adds a market-read lens inside the existing council stage, coupled to the existing verdict mechanism, so it gates designs without new machinery.

**Constraints (governance):** the change touches no auth, billing, webhooks, data migration, Stytch contracts, or tenant-scoped data; it is OPS-GOV tooling. The council marker contract (`^STATUS: (APPROVED|REJECTED)$` first `STATUS:` line), `parse_verdict`, exit codes 1/2/3, and `scripts/tests/fixtures/` are invariant. `routing.json` remains advisory per the living `native-agent-pipeline` spec ("SHALL NOT replace or override OpenSpec artifacts or the lifecycle gates"). Sibling change `use-inout-council-for-a-better-design` also edits `council.md`/`pipeline.sh` (revision loop); this change stays disjoint (personas/deliverable/routing field/traceability).

## Goals / Non-Goals

**Goals:**

- Council reviews market viability, not just correctness, for designs that touch money, AI cost, the WhatsApp/Meta channel, compliance, or the acquisition funnel — with repo-grounded evidence and external claims flagged as assumptions (premise-validation gate).
- A machine-parseable `MARKET:` line in `VERDICT.md` (always present: `PASS | CONDITIONAL | FAIL | N/A`) that gates through the existing STATUS verdict — no new marker contract, no new stage, no exit-code change.
- Advisory `routing.json` field `requires_market_read` so a medium-complexity pricing/billing change still triggers the council.
- Design-side contract: market-in-scope changes SHALL carry `## Market & Unit Economics` and `## Market Risk` sections; their absence is a reviewable defect.
- Traceability: iso records the `MARKET` line and top market risks; `AGENTS.md` documents the five-persona council.

**Non-Goals:**

- Changing the STATUS marker contract, `parse_verdict`, exit codes, or verdict fixtures.
- A separate market pipeline stage, a hard `MARKET`-independent gate, or new OpenSpec schema artifacts/lifecycle gates.
- Council doing copywriting, GTM execution, or real customer research — it reviews the design's market logic and risks; unknowns are flagged for validation, never asserted.
- Re-implementing the sibling change's revision loop or any opsx-workflow logic.

## Decisions

### D1 — Five-persona council (council.md)

`.pi/prompts/council.md` gains two personas, reviewed after SRE in fixed order: **Staff Product/GTM** (the SaaS-expert lens) and **Colombia IT & Market** (the local-expert lens). The two balance each other — the SaaS expert keeps the product commercially sane, the Colombian expert keeps it locally true. Both produce severity-tagged findings in the existing format and are checklist-driven:

- **Staff Product/GTM** — the SaaS-expert lens: unit economics (USD token cost per AI action vs price point in COP, plan/fee margins at Polar + MercadoPago/PSE/Nequi), pricing coherence (plan tiers vs feature gating vs credit guard/`ai-usage-metering`), activation/churn (onboarding, empty states, funnel), competitive substitution (Meta native WhatsApp AI, local WhatsApp CRM incumbents).
- **Colombia IT & Market** — the local-expert lens: Ley 1581/Habeas Data (consent, export/forget, transfer), Ley 1480 consumer law, the payment and invoicing ecosystem (PSE/Nequi; DIAN electronic invoicing — tools like SIIGO are examples, not the point), data retention; WhatsApp Business Platform policy drift (conversation pricing, template approval, AI-agent terms, native AI messaging) as an existential channel risk surface.

Existing personas, the review method, the absolute prohibitions, and the `--tools read,write` posture are unchanged.

### D2 — Market-in-scope definition (deterministic surface)

A change is **market-in-scope** when it touches any of: (a) billing/pricing/paywall/quotas/credits; (b) AI usage metering, model routing, LLM cost, agent behavior; (c) WhatsApp/Meta channel (ingress/outbound/templates/campaigns); (d) compliance (Ley 1581, the local invoicing ecosystem — DIAN, tools like SIIGO — consumer law, retention); (e) marketing site / signup funnel / onboarding / activation. The council detects scope from `design.md`/`proposal.md`; authors SHOULD also set `requires_market_read: true` in `routing.json` for these classes (D4). In-scope changes SHALL carry `## Market & Unit Economics` and `## Market Risk` sections in `design.md`; their absence is a REJECT-level design defect (unmitigated), because the council cannot verify what the design does not state.

### D3 — `MARKET:` line and STATUS coupling (marker contract preserved)

`VERDICT.md` SHALL always contain a `MARKET: PASS | CONDITIONAL | FAIL | N/A` line and, for in-scope changes, a `## Market Read` prose section. `MARKET:` is not `STATUS:`-prefixed, so `parse_verdict`, fixtures, and exit codes are untouched (verified against `parse_verdict`'s first-`STATUS:`-line logic). Coupling for in-scope changes:

- `MARKET: FAIL` without an explicitly accepted residual recorded in `design.md` `## Market Risk` (named risk, owner, trigger) → the council SHALL NOT write `STATUS: APPROVED`; it SHALL write `REJECTED`.
- `MARKET: CONDITIONAL` → `STATUS: APPROVED` only when each condition is either fixed by design revision or recorded as an accepted residual; otherwise `REJECTED`.
- `MARKET: PASS` / `N/A` → no constraint on the STATUS verdict.
- Residual rubric (calibrate on first real verdicts): negative projected unit margin at current LLM prices → FAIL; missing evidence for a cost/market claim → CONDITIONAL; bounded risk with mitigation → PASS.

### D4 — Advisory `routing.json` extension (one-line gate)

`routing.json` MAY declare `requires_market_read: true`; `council_required()` gains `[[ "$(routing_get '.requires_market_read' 'false')" == "true" ]] && return 0` so a medium/low-complexity market-touching change still runs the five-persona council. Header comment (line ~28) documents the field. `--with-council` already forces the full council, so no new CLI flag. Advisory semantics unchanged.

### D5 — Enforcement mechanics (review-time, no new machinery)

Market-read is review-time behavior inside the existing council stage. The design.md section requirement (D2) is a reviewer-side contract pinned in `council.md` — the same enforcement pattern as the council's existing deliverable contract (precedent: the sibling change pins the numbered-required-changes contract the same way). `sdet`/`opsx-apply` is unaffected; code landing still consumes the approved design + tasks. No `openspec` CLI change, no `apply.requires` change.

### D6 — Evidence rule (premise validation)

Both new personas SHALL ground findings in repo evidence (`design.md`, `proposal.md`, `openspec/specs/`, `ROADMAP.md`, `GAP-ANALYSIS-2026.md`, code where verifiable) and SHALL mark external facts (Meta pricing/terms, regulator posture, competitor pricing) as **assumptions to validate** — never asserted facts. A design that asserts an unverified external fact as a premise SHALL be flagged (mirrors the repo's premise-validation gate; unverifiable claims belong in the proposal's Assumptions section).

### D7 — Traceability

`.pi/prompts/iso.md` gains a line item: record the `MARKET` line (or `N/A` when absent) and the top 1–3 market risks in `docs/compliance/ISO_TRACEABILITY_MATRIX.md`. `AGENTS.md` "Agent Pipeline" documents the five-persona council, the in-scope surface, the `MARKET:` line, and the routing field.

**Alternatives considered:** (1) separate market stage/hard gate — rejected: duplicates the verdict mechanism, violates advisory routing semantics, adds machinery; (2) new OpenSpec artifact (e.g., `market.md`) — rejected: schema is fixed (non-goal), review-time behavior is the established pattern; (3) `MARKET: FAIL` hard-block independent of STATUS — rejected: two competing verdicts confuse the gate; coupling through STATUS keeps a single gate; (4) routing-only change without personas — rejected: nothing would actually review the market surface.

## Risks / Trade-offs

- **[Risk] Council review time grows** → Mitigation: checklist-driven personas, severity-tagged findings, market sections only for in-scope changes.
- **[Risk] Council invents market facts (competitor pricing, regulator posture)** → Mitigation: D6 evidence rule — repo-grounded findings; external claims flagged as assumptions; asserted external premises are flagged as defects.
- **[Risk] Collision with sibling `use-inout-council-for-a-better-design` (both edit council.md/pipeline.sh)** → Mitigation: disjoint edit regions (personas/deliverable/routing field vs revision loop); both preserve the same marker contract; delta specs target different requirements and merge cleanly at archive; rebase note recorded in tasks.
- **[Risk] Accepted residuals rubber-stamped** → Mitigation: residuals must be recorded in `design.md` `## Market Risk` with named risk, owner, and trigger; the (sibling) re-review loop surfaces them to the council.
- **[Risk] Scope creep into copy/GTM review** → Mitigation: non-goal — the personas review the design's market logic and risk, never copy or execution.
- **[Trade-off] Five-persona reviews are heavier** → in-scope changes are the minority; out-of-scope verdicts carry `MARKET: N/A` and no market sections.

## Migration Plan

1. Edit `.pi/prompts/council.md`: add the two personas (D1), the in-scope definition + design.md section requirement (D2), the `MARKET:` line + `## Market Read` deliverable + coupling rubric (D3), and the evidence rule (D6).
2. Edit `scripts/pipeline.sh`: one-line `council_required()` addition + header comment (D4).
3. Edit `.pi/prompts/iso.md` and `AGENTS.md` (D7).
4. Write the `native-agent-pipeline` delta spec (MODIFIED ×3, ADDED ×1).
5. Verify: `bash -n scripts/pipeline.sh`; `scripts/pipeline.sh <change> --dry-run` on a fixture change with `routing.json` `requires_market_read: true` + `complexity: medium` shows the council stage; grep fixture shows `^STATUS:` remains the first STATUS line with a `MARKET:` line present; `openspec validate add-council-market-read-gate` passes.
6. Rollback: revert the commit. `routing.json` `requires_market_read` is advisory and ignored by the pre-change pipeline; no schema, DB, auth, or Stytch impact to unwind.

## Open Questions

- Whether `MARKET: CONDITIONAL` should force a design revision rather than allow approved-with-conditions — initially allowed with recorded residuals; revisit after the first real conditional verdict.
- Whether in-scope detection should rely on author-set `requires_market_read` or council self-detection — both supported; guidance: billing/AI/channel/compliance changes SHOULD set the field, council self-detects regardless.
- Rubric calibration for PASS vs CONDITIONAL vs FAIL — a first-draft rubric ships (D3); calibrate on real verdicts once the lens runs in anger.
