## Context

The OpenSpec workflow is layered: `AGENTS.md` (repo guidance) → `.opencode/commands/opsx-*.md` (project slash-command front-ends) → `.opencode/skills/openspec-*/SKILL.md` (agent instruction files, `generatedBy: 1.6.0` from the openspec CLI). The CLI's `openspec update` can regenerate the skill files and overwrite local edits.

Three observed governance failures (this change's motivating evidence):

1. **No archive enforcement.** `add-whatsapp-inbox` is marked "✓ Complete" (50/50 tasks) yet was never archived — its delta specs (`whatsapp-inbox`, `crm-conversation-api`, `whatsapp-outbound-send`, `whatsapp-config-api`) were never folded into `openspec/specs/`. The apply skill only "suggests" archiving (`state: all_done → suggest archive`), and the archive skill explicitly refuses to block on warnings.
2. **No premise validation.** `add-crm-e2e-tests` committed a proposal asserting "a full React UI exists" when the shipped UI was read-only tables. The propose workflow has no step verifying factual claims against the codebase.
3. **No verification gate.** The frontend shipped 39 TypeScript errors with the build never run as part of apply; the apply workflow marks tasks complete based on agent self-report, not on the verification commands tasks.md defines.

## Goals / Non-Goals

**Goals:**
- Add a mandatory verification gate to the apply workflow: run the verification commands recorded in tasks.md before reporting completion; failures keep the change in-progress.
- Add archive enforcement: after a green gate, require an archive decision (invoke `/opsx-archive` by default, or record an explicit deferred entry). No silent "complete but unarchived" state.
- Add premise validation to the propose workflow: verify factual claims against the codebase before writing delta specs; unverifiable premises become explicit Assumptions.
- Harden archive guardrails: block when incomplete tasks are verification tasks; keep confirm-and-proceed for non-verification gaps.
- Document the three gates in AGENTS.md and extend the config.yaml tasks tag rule with `[OPS-GOV]`.
- Keep all edits textual and reversible.

**Non-Goals:**
- Repairing the observed drift itself (39 TS errors — fixed by the archived `fix-frontend-build`; `add-whatsapp-inbox` folding; e2e spec rewrite) — tracked separately.
- Changing the `openspec` CLI binary or its behavior.
- Any application code in `go-b2b-starter/` or `next_b2b_starter/`.

## Decisions

### D1: Verification gate in `opsx-apply` / `openspec-apply-change`

Before reporting "Implementation Complete", the agent MUST collect every verification command named in tasks.md (per-task verification criteria per config rules) and run them. All must pass (or record an explicit accepted-exception with the user) before the change is marked complete. Failures are recorded in tasks.md and the change remains in-progress.

Implementation: in the "Handle states" section, the `all_done` branch gains a verification sub-step; the "Implementation Complete" section gains a required "Verification" line listing commands run and results. Both files (`opsx-apply.md` command, `openspec-apply-change/SKILL.md` skill) carry the same wording so the contract is consistent across the two entry points.

### D2: Archive enforcement

The `all_done` + green-verification state transitions to an archive decision: invoke `/opsx-archive` by default, or record `**Archive deferred:** <reason>` in tasks.md. The apply skill's final output changes from "suggest archive" to "require archive decision". This closes the `add-whatsapp-inbox` failure class at the workflow level.

### D3: Premise validation in `opsx-propose` / `openspec-propose`

Before writing delta specs, the propose workflow MUST verify each factual premise in the proposal against the codebase (routes, components, endpoints, build status via grep/glob/tsc). Verified premises proceed; unverifiable ones MUST be demoted to an "## Assumptions" section in the proposal with the evidence gap noted. The proposal template gains an optional Assumptions section. This is the direct antidote to the "full React UI exists" failure.

### D4: Archive-side guardrail tightening

`opsx-archive.md` / `openspec-archive-change/SKILL.md`: when incomplete tasks are **verification tasks** (criteria not met), the archive MUST block with an explanation of which task blocks. Non-verification gaps (e.g., optional documentation) keep the current confirm-and-proceed behavior. This is a targeted change to the existing "Don't block archive on warnings" guardrail — it becomes "block on verification gaps, inform-and-confirm otherwise".

### D5: Durable rules (survive `openspec update`)

`.opencode/skills/*.md` carry `generatedBy: "1.6.0"` and may be overwritten by `openspec update`. Therefore:
- The authoritative contract lives in `AGENTS.md` (Mandatory Workflow section) and `openspec/config.yaml` (tasks rule gains `[OPS-GOV]`).
- Skill/command files carry the same gates as operational instructions; AGENTS.md documents the regeneration risk so future `openspec update` runs re-apply the gates.

## Risks / Trade-offs

- **Agent friction**: mandatory verification adds a step to every apply. Mitigation: gates are explicit and record results once per change, not per task; failures are logged with the failing command output.
- **Skill regeneration clobbering**: `openspec update` may overwrite skill edits. Mitigation: AGENTS.md documents the risk (D5); the gates exist in both the command files (project-owned) and the skill files.
- **False blocks on archive**: an overly strict archive could block legitimate infra-only changes. Mitigation: `--skip-specs` remains available; blocking applies only to verification-task gaps, not to missing specs.
- **Premise-validation cost**: grepping the codebase on every propose adds a few seconds. Mitigation: validation is scoped to claims made in the proposal, not a full code audit.
