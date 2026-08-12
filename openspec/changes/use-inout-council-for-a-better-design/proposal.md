# Use Council Input/Output for a Better Design — Bounded Design-Revision Loop

## Why

Council review is a single pass: a `STATUS: REJECTED` verdict halts `scripts/pipeline.sh` (exit 2) and nothing consumes the verdict's findings — design quality depends on a human noticing the halt, manually revising `design.md`/`tasks.md`/delta specs, and re-running the pipeline. That loop is real and costly: `new-client-billing-lifecycle` was REJECTED twice on 2026-08-12 (initial + re-review), each requiring a manual revision cycle before the design passed. The council's output (`VERDICT.md`, with numbered required design changes) is prose that no stage reads, so findings are re-discovered instead of folded in. This change closes the loop: council output becomes the input to a bounded, automated design-revision stage, so a rejected design returns to the council improved within one pipeline run — and still halts, deterministically, if the cap is exhausted.

## What Changes

- **New `revise` pipeline stage** (`.pi/prompts/revise.md` prompt template, placed between council and sdet): invoked only when `VERDICT.md` carries `STATUS: REJECTED`. The revise stage SHALL read the verdict's numbered required design changes as its authoritative input, revise the change's planning artifacts (`proposal.md`, `design.md`, `tasks.md`, delta specs under `specs/`) to address them, and verify coverage. It SHALL apply the existing `opsx-update` reconciliation rules to artifact paths provided by the pipeline (thin wrapper — never re-implements OpenSpec update behavior) and SHALL NEVER edit application source code. It runs with read/write-only tools: `pipeline.sh` writes the artifact paths to a transient `.revise-context.json` (from `openspec status --change <name> --json`) and runs `openspec validate <change>` after each pass.
- **VERDICT.md required-changes contract**: the council prompt template SHALL require every verdict to include a numbered "Required design changes" list — the revise stage SHALL check off each numbered item against the revised artifacts before the design is re-submitted for review — and SHALL direct a re-review to read the prior `VERDICT.md` and `revision.md` when both exist (the "input/output" contract).
- **Bounded council-revision loop in `scripts/pipeline.sh`**: on `REJECTED`, run `revise` → `council` (re-review) up to a cap (`MAX_COUNCIL_REVISIONS`, default 2). If the verdict is still `REJECTED` after the cap, the pipeline SHALL halt with exit 2 (final halt semantics and the `^STATUS: (APPROVED|REJECTED)$` marker contract are preserved). New flags: `--max-revisions <N>`, `--skip-revise` (restore the current immediate-halt behavior), and `--reset-revisions` (delete the revision counter for a fresh cycle after a manual fix); an approved re-review auto-clears the counter. Revision progress is tracked in the change directory (`revision.json`) so the cap is enforced across pipeline re-runs and the loop is idempotent.
- **Revision records**: each revise pass SHALL record the covered required-change numbers and the revised artifacts in a machine-parseable `revision.md` sidecar, and SHALL log the pass under `logs/pipeline/<change>-revise-<n>-<ts>.log`.
- **Traceability**: the iso stage SHALL record council-revision-loop outcomes in `docs/compliance/ISO_TRACEABILITY_MATRIX.md`; `AGENTS.md`'s Agent Pipeline section SHALL document the `revise` stage and the loop.
- **Behavior change to the council gate** (not a marker-contract change): `REJECTED` no longer halts on the first pass — it triggers the bounded loop. Exit-code contract preserved: `APPROVED` → proceed, `REJECTED` after cap → exit 2, absent/ambiguous marker → exit 3.

## Capabilities

### New Capabilities
- *(none — the revision loop is an extension of the existing pipeline capability, expressed as added/modified requirements in one delta spec)*

### Modified Capabilities
- `native-agent-pipeline`: the "Council verdict marker contract" requirement changes — a `REJECTED` verdict triggers a bounded revise→re-review loop instead of an immediate halt (halt still occurs after the cap); the "pipeline.sh orchestrates stages deterministically" requirement changes — stage order gains the conditional revise loop and the new flags; a new "Council-driven design revision loop" requirement is added (revise stage, coverage verification, revision tracking).

## Impact

- `scripts/pipeline.sh` — council gate: loop orchestration, `--max-revisions`/`--skip-revise`, `revision.json` read/write, per-iteration logs.
- `.pi/prompts/revise.md` — new stage prompt template (thin wrapper over the `opsx-update` workflow, non-interactive).
- `.pi/prompts/council.md` — verdict contract: numbered required-changes list pinned for machine consumption + conditional re-review read clause (review personas and marker format unchanged).
- `.pi/prompts/sdet.md` / `opsx-apply.md` — unaffected (apply still consumes the final approved design + tasks).
- `AGENTS.md` — Agent Pipeline section documents the revise stage and bounded loop.
- `docs/compliance/ISO_TRACEABILITY_MATRIX.md` — iso stage records revision-loop outcomes.
- No application source code, no database migrations, no auth/billing/webhook behavior, no Stytch contract changes.
