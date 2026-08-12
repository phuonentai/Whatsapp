## Context

The repository runs a dual orchestration story today:

- **OpenSpec governance (native Pi)**: `.pi/prompts/opsx-{propose,apply,archive,explore,sync,update}.md` and `.pi/skills/openspec-*` implement the spec-driven lifecycle (premise validation → verification gate → archive decision) mandated by `AGENTS.md` and `openspec/config.yaml`. This is healthy and authoritative.
- **omp/Pantheon (external layer)**: `omp-install.sh`, `.oh-my-opencode-pi-workflow.json`, `.oh-my-opencode-pi-stats.json`, `.oh-my-opencode-pi-debug/` add background agents and a Pantheon delegate. Its own stats record 136 failed background tasks and a `pantheon` failure kind — the layer this repo relies on for orchestration is demonstrably flaky.

The goal is to collapse the second layer into the first: one deterministic, native Pi pipeline that keeps OpenSpec as the single lifecycle and adds the missing stages (adversarial council review, Playwright QA, ISO traceability) as plain prompt templates, chained headlessly via `pi -p`.

```
                    BEFORE                          AFTER
┌──────────────────────────────┐     ┌───────────────────────────────────────┐
│  omp/Pantheon (flaky: 136 ✗)│     │  scripts/pipeline.sh (deterministic)  │
│   background agents          │     │   pi -p --prompt-template <stage>.md  │
└──────────────┬───────────────┘     └───────────────┬───────────────────────┘
               │                                     │
┌──────────────▼───────────────┐     ┌───────────────▼───────────────────────┐
│  OpenSpec governance (kept)  │     │  OpenSpec governance (kept, extended) │
│  .pi/prompts/opsx-*          │◀────│  opsx-propose → council → opsx-apply  │
│  .pi/skills/openspec-*       │     │  → uiux → iso                        │
│  3 lifecycle gates           │     │  artifacts-on-disk = shared state     │
└──────────────────────────────┘     └───────────────────────────────────────┘
```

## Goals / Non-Goals

**Goals:**
- Remove omp/Pantheon artifacts from the repository and stop referencing them as the pipeline.
- Add `/council`, `/uiux`, `/iso` prompt templates (plus thin `/architect`, `/sdet` wrappers) under `.pi/prompts/`.
- Install the three verified extensions project-locally (`@bopstack/pi-codegraph`, `pi-playwright`, `pi-web-search`) into `.pi/settings.json`.
- Provide `scripts/pipeline.sh` that chains stages deterministically via `pi -p --prompt-template`, with a machine-parseable council gate and per-stage logs.
- Extend `governance-workflow` with an adversarial-review gate requirement; leave all other OpenSpec behavior intact.

**Non-Goals:**
- Not modifying the `opsx-*` prompts, `openspec-*` skills, or the three lifecycle gates.
- Not repairing omp/Pantheon.
- Not installing `git-guard` / `safe-guard` (do not exist on npm).
- Not touching auth, persistence, or any Go/Next application code.

## Decisions

### D1 — Orchestration: bash-chained `pi -p --prompt-template`, not omp, not a single session
Each stage is a stateless `pi -p` invocation; shared state is the change directory on disk (`proposal.md`, `design.md`, `tasks.md`, `VERDICT.md`, `qa/`). This is deterministic (identical inputs → same gate decisions), auditable (logs in `logs/pipeline/`), and uses only native pi primitives.
*Alternatives:* (a) keep and repair omp — rejected: 136 failed background tasks and the user's goal to strip the layer; (b) one long interactive session — rejected: no reproducibility, no headless CI use.

### D2 — Print-mode invocation: `--prompt-template`, not slash commands
pi resolves `/name` slash commands only in the TUI editor; print mode loads templates via `pi -p --prompt-template <path> "..."` (`docs/usage.md` §33, §226). Every stage invocation in `pipeline.sh` uses the explicit flag. Prompt files keep frontmatter `description:` so they also work as `/council` etc. interactively.
*Alternative:* rely on slash commands — rejected: breaks in `-p` mode, which is the whole point.

### D3 — Council gate: one machine-parseable marker line, not substring grep
`VERDICT.md` MUST contain exactly one line matching `^STATUS: (APPROVED|REJECTED)$`. The pipeline reads the first `^STATUS:` line: `APPROVED` → proceed; `REJECTED` → halt with exit 2; absent/ambiguous → halt as inconclusive. This fixes the fragile `grep -c "STATUS: REJECTED"` substring match (which trips on prose like "rejected items: ...").
*Alternative:* JSON verdict — rejected: heavier for LLM output, harder to read as a human artifact.

### D4 — Stage gating: optional `routing.json` sidecar, advisory by default
The change directory MAY contain `routing.json`:
```json
{ "requires_council": true, "requires_playwright": true, "requires_iso": true, "complexity": "high" }
```
Defaults when absent: council only when `complexity: high` (or explicit flag), playwright only on UI-touching changes (or explicit flag), iso always. `pipeline.sh` supports `--with-council`, `--with-uiux`, `--skip-iso` overrides. The sidecar is advisory metadata; it never replaces OpenSpec artifacts or gates.
*Alternative:* hard-code gates in the pipeline — rejected: every change differs (a pure backend change shouldn't force Playwright).

### D5 — Extensions: `pi install -l` for the three verified packages only
`@bopstack/pi-codegraph`, `pi-playwright`, `pi-web-search` exist on npm (verified 0.1.1 / 0.1.1 / 1.3.1). `pi install -l npm:<pkg>` writes `.pi/settings.json` (project-local). Tool-level guardrails stay in `AGENTS.md` policy + review gates until a real guard package exists.
*Alternative:* invent local stubs for `git-guard`/`safe-guard` — rejected: hallucinated dependencies are worse than none.

### D6 — `/architect` and `/sdet` are thin delegating wrappers
`architect.md` instructs the agent to run the `/opsx-propose` workflow; `sdet.md` delegates to `/opsx-apply` (which already owns the verification gate and task-by-task implementation). The pipeline therefore never bypasses OpenSpec: `opsx-apply`'s verification gate remains the completion authority.
*Alternative:* independent re-implementations of propose/apply — rejected: forks governance.

### D7 — omp removal is git-tracked and revertible; side-agent scripts gated on verification
Delete: `omp-install.sh`, `.oh-my-opencode-pi-workflow.json`, `.oh-my-opencode-pi-stats.json`, `.oh-my-opencode-pi-debug/`. Whether `.pi/side-agent-start.sh`, `.pi/side-agent-finish.sh`, `.pi/side-agents/` are omp machinery is verified during apply (grep for surviving references); if referenced by remaining tooling they are kept and documented, otherwise removed.
*Alternative:* delete side-agent files blindly — rejected: they may belong to a separate mechanism still in use.

## Risks / Trade-offs

- [Removing omp breaks a workflow someone relies on] → Git-tracked removal; rollback is `git revert` of this change; migration note in `tasks.md`/docs.
- [Extensions break `pi` startup or consume context] → `.pi/settings.json` is project-local and removable via `pi remove`; `codegraph` exists precisely to reduce context reads.
- [Headless `pi -p` loses "previous context"] → Each prompt template explicitly names its input files (e.g., council reads `design.md` + `proposal.md`); never rely on session memory.
- [Council gate adds latency/throughput friction] → Advisory by default (D4); verdict recorded in `tasks.md` per the governance-workflow delta so it is visible, not blocking unless required.
- [`pipeline.sh` quoting/env issues in CI] → `set -Eeuo pipefail`, `bash -n` check, per-stage log files, `--dry-run` mode that prints the exact commands.
- [npm `verify-deps-before-run` config warning on `pi install`] → Benign npmrc warning, unrelated to pi; ignore.

## Migration Plan

1. Install the three extensions (`pi install -l`) — additive, zero risk.
2. Add `.pi/prompts/{council,uiux,iso,architect,sdet}.md` — additive.
3. Add `scripts/pipeline.sh` + `routing.json` convention + `logs/pipeline/`.
4. Remove omp artifacts (D7), update `BOOTSTRAP_REPORT.md`, `DEVELOPMENT.md`, `README.md`, `AI-architecture.md` references, add the short agent-pipeline section to `AGENTS.md`.
5. Write the `governance-workflow` delta spec and the `native-agent-pipeline` spec.
6. Smoke-test end-to-end: run `pipeline.sh` (or `--dry-run`) against a sample change; verify extension tools appear in the pi manifest.

Rollback: `git revert` of this change restores all files; `pi remove -l` the three extensions if desired. Stytch tenant policy state is not affected (no auth/persistence changes).

## Open Questions

- Do `.pi/side-agent-*.sh` / `.pi/side-agents/` belong to omp? (verified during apply — D7)
- Which of the omp references in root docs are normative vs. historical? (update conservatively, keep history in git)
- Should `pipeline.sh` grow a CI mode (`--ci` with `--mode json` output for status checks) in a follow-up change? (out of scope here)
