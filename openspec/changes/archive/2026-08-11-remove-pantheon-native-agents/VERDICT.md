STATUS: APPROVED

# Council Verdict — remove-pantheon-native-agents

Reviewed: `design.md` + `proposal.md` (+ delta specs, `pipeline.sh`, `.pi/prompts/*.md`, `.pi/settings.json`, `AGENTS.md`, root docs, pi docs) on 2026-08-11. Premise validation performed with read-only tooling; every disk-verifiable claim in the proposal was confirmed.

## Summary

The design is sound: it collapses a demonstrably flaky external orchestration layer (omp/Pantheon) into a deterministic native Pi pipeline while keeping OpenSpec as the single lifecycle, and its removal scope is tightly bounded (tooling/docs only — no Go/Next code, no DB, no Stytch state, no credential storage). The `VERDICT.md` marker contract, kebab-case change sanitization, advisory `routing.json` gating, `--approve` trust rationale, and `--tools read,write` council restriction are all correctly designed and match pi's documented behavior. No design-blocking defects found.

## Per-persona findings

### Staff Security Engineer

- **LOW — Extension versions are unpinned despite the "reviewed pinned versions" claim.** `.pi/settings.json` declares bare package names (`npm:@bopstack/pi-codegraph`, `npm:pi-playwright`, `npm:pi-web-search`) with no version specifiers, yet `AGENTS.md` asserts "the current three are reviewed pinned versions." Under `--approve`, every headless stage executes project-local third-party code with full system access, so supply-chain drift is the residual risk. **Recommended:** pin versions in `.pi/settings.json` (e.g., `npm:@bopstack/pi-codegraph@0.1.1`) or soften the "pinned" wording in `AGENTS.md`. Residual risk accepted by design (documented trade-off, project-local and removable via `pi remove -l`).
- **LOW — Council's "cannot mutate the repo beyond VERDICT.md" is prompt-level, not tool-enforced.** `--tools read,write` removes bash/git but the `write` tool accepts arbitrary paths; the prohibition in `council.md` is the only constraint. For a trusted internal pipeline this is acceptable; do not rely on it against an adversarial or compromised template/extension.
- **OK — No secrets, no auth/persistence changes, no Stytch tenant policy impact.** Rollback is a plain `git revert`; the Non-Goals section correctly rejects local credential storage. omp artifacts (`omp-install.sh`, `.oh-my-opencode-pi-*.json`, `.oh-my-opencode-pi-debug/`) confirmed absent on disk.

### Staff DBA

- **LOW — No data-plane impact; advisory gating convention is coherent.** No migrations, no tables, no query changes. The `routing.json` sidecar fields the pipeline reads (`requires_council`, `complexity`, `requires_playwright`, `requires_iso`) are a strict subset of what the architect wrapper emits; extra fields (e.g., `requires_migration`) are ignored harmlessly. Nothing in the pipeline touches PostgreSQL, SQLC, or Stytch.
- **OK — Expand-contract and idempotency constraints preserved** in `sdet.md` delegation to the unmodified apply workflow.

### SRE

- **LOW — `run_stage` failure reporting prints the wrong exit code.** In `pipeline.sh`, the failure message `echo "... FAILED (exit $?)"` runs inside `if ! "${cmd[@]}" ...; then`, where `$?` is the negation's status (0), so it will read `FAILED (exit 0)` and the real code is lost (the pipeline still exits 1, and per-stage logs capture the actual output). **Recommended:** capture `rc=$?` from the direct command before negating.
- **LOW — Task 3.2's fixture tests are referenced but absent.** `pipeline.sh`'s comment says "Fixtures in scripts/tests/fixtures/" but `scripts/tests/` does not exist, so the claimed prose-mentions-"rejected" fixture test is not verifiable on disk. The parsing logic itself (first `^STATUS:` line, exact match, INCONCLUSIVE fallback) is correct. **Recommended:** add the fixture files (or a tiny test harness) and re-check task 3.2, or remove the comment reference.
- **LOW — governance-workflow delta wording vs. enforcement point.** The delta says "the apply workflow SHALL block marking the change complete until `VERDICT.md` has `STATUS: APPROVED`", but enforcement lives in the `sdet.md` wrapper stage, not in the (intentionally unmodified) `opsx-apply` prompt/skill. A direct `/opsx-apply` invocation bypasses the gate. Since gating is advisory-by-default per D4, this is acceptable — but align the delta wording ("the pipeline's sdet stage SHALL...") or document the bypass so archive/ISO reviewers don't over-trust the wording.
- **OK — Determinism claim is orchestration-determinism** (fixed stage set, fixed `pi -p --prompt-template` invocation, artifacts-on-disk state, deterministic gate parsing), not identical LLM outputs; the design's phrasing is fair. `bash -n`, dry-run, per-stage logs, and jq as a documented prerequisite are all present.
- **NOTE (claimed, not independently re-run here):** verification commands `bash -n scripts/pipeline.sh`, `scripts/pipeline.sh <change> --dry-run` exit 0, and `openspec validate` are recorded as passing in `tasks.md` but were not executable in this read/write-only review; re-run them at the verification gate.

## Findings that must be closed before archive

1. **Record this council verdict in `tasks.md`.** This change's own `routing.json` sets `requires_council: true` and its `governance-workflow` delta requires recording the verdict in `tasks.md` when council is required; no verdict record currently exists there. Add an entry (approved) before archiving so the change is consistent with the governance it introduces.
2. **Reconcile the "pinned" claim with `.pi/settings.json`** (pin versions or correct the wording) — supply-chain hygiene, low effort.
3. **Fix or remove the `scripts/tests/fixtures/` reference** and confirm task 3.2's verification criterion.
4. **Fix the `(exit $?)` reporting bug** in `pipeline.sh` `run_stage` (capture the real exit code).

## Residual risks (accepted by design, documented in `design.md`)

- `--approve` grants project-local third-party extension code full system access in headless mode; mitigated by reviewed packages, project-local install, and the council's restricted tool set.
- Direct `opsx-*` usage bypasses pipeline gating; OpenSpec lifecycle gates remain authoritative either way.
- Removal of omp breaks anyone still relying on it; rollback is a Git revert.

## Required changes (rejection criteria — none triggered)

No design changes are required. The four archive-blocking items above are process/consistency cleanups, not design defects; they can be closed during the verification gate.
