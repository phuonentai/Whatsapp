## Why

The repository's orchestration layer (omp/Pantheon) is installed and active — `.oh-my-opencode-pi-workflow.json`, `.oh-my-opencode-pi-stats.json`, `omp-install.sh`, and `.oh-my-opencode-pi-debug/` (8 `trace_pantheon_delegate_*` files) — but its own telemetry shows it is unreliable: `.oh-my-opencode-pi-stats.json` records **136 failed background tasks** and a `pantheon` failure kind. Meanwhile the OpenSpec governance workflow already exists as native Pi primitives (`.pi/prompts/opsx-*`, `.pi/skills/openspec-*`) and is the repository's constitution per `AGENTS.md` and `openspec/config.yaml`. We want to replace the flaky omp/Pantheon layer with a deterministic native Pi pipeline, add the missing agent stages (council, uiux, iso), install the three real extensions (verified on npm), and keep OpenSpec governance untouched.

## What Changes

- **Remove omp/Pantheon** — **BREAKING**: delete `.oh-my-opencode-pi-workflow.json`, `.oh-my-opencode-pi-stats.json`, `.oh-my-opencode-pi-debug/`, `omp-install.sh`. Remove references to this orchestration layer from `BOOTSTRAP_REPORT.md`, `DEVELOPMENT.md`, `README.md`, and `AI-architecture.md` where they assert it as the pipeline. Verify during apply whether `.pi/side-agent-start.sh`, `.pi/side-agent-finish.sh`, and `.pi/side-agents/` belong to omp machinery; remove only if unreferenced by surviving tooling, otherwise keep and document.
- **Add the missing agent stages as native prompt templates** in `.pi/prompts/`:
  - `/council` — adversarial multi-persona review (security / DBA / SRE) of `design.md` → writes `VERDICT.md` with a machine-parseable `STATUS:` line.
  - `/uiux` — Playwright visual + accessibility QA at 390x844, 768x1024, 1440x900 → `openspec/changes/<change>/qa/screenshots/` + `qa/REPORT.md`.
  - `/iso` — compliance traceability → updates `docs/compliance/ISO_TRACEABILITY_MATRIX.md` with file/symbol-level evidence.
  - `/architect` and `/sdet` — thin wrappers that delegate to the existing `/opsx-propose` and `/opsx-apply` prompts, so the pipeline surface is uniform while OpenSpec remains the single lifecycle.
- **Install the verified extensions project-locally** via `pi install -l`: `@bopstack/pi-codegraph`, `pi-playwright`, `pi-web-search` → writes `.pi/settings.json`. (`git-guard` and `safe-guard` do not exist on npm — see Assumptions/Non-Goals.)
- **Add `scripts/pipeline.sh`** — deterministic orchestrator chaining `pi -p --prompt-template .pi/prompts/<stage>.md "..."` invocations (slash commands are TUI-only; print mode requires `--prompt-template`). Stage gating reads an optional `routing.json` sidecar in the change directory; council verdicts halt the pipeline on `STATUS: REJECTED`.
- **Keep OpenSpec governance**: `opsx-*` prompts, `openspec-*` skills, the three lifecycle gates, and `AGENTS.md` remain authoritative. `AGENTS.md` gains a short section documenting the agent pipeline. A delta to the `governance-workflow` spec makes the council gate a governed requirement (see Capabilities).

## Capabilities

### New Capabilities
- `native-agent-pipeline`: The deterministic native Pi agent pipeline — the council/uiux/iso prompt templates and their inputs/outputs, the `VERDICT.md` gate semantics, the `routing.json` sidecar convention, `pipeline.sh` orchestration rules, and the project-local extension set declared in `.pi/settings.json`.

### Modified Capabilities
- `governance-workflow`: Requirement change — when a change opts into council review (via `routing.json` or explicit record), the apply workflow SHALL block until the verdict is approved and SHALL record the verdict in `tasks.md`. This extends the existing lifecycle (premise validation → verification gate → archive decision) with an adversarial review gate; it does not replace it.

## Impact

- **Removed**: `omp-install.sh`, `.oh-my-opencode-pi-*.json`, `.oh-my-opencode-pi-debug/`; stale omp references in root docs and `BOOTSTRAP_REPORT.md`.
- **Added**: `.pi/prompts/{architect,council,sdet,uiux,iso}.md`, `.pi/settings.json`, `scripts/pipeline.sh`, `routing.json` convention, `openspec/changes/<change>/qa/` and `VERDICT.md` artifact conventions.
- **Unchanged**: Go backend (`go-b2b-starter/`), Next.js frontend (`next_b2b_starter/`), DB schema, SQLC models, Stytch B2B integration (auth/RBAC/sessions untouched — no local credential storage introduced), and the OpenSpec `opsx-*`/`openspec-*` workflow files.
- **Verification**: extension presence in the pi tool manifest after install; `pi -p --prompt-template` smoke run for each new stage; `pipeline.sh` dry-run on a sample change; `bash -n scripts/pipeline.sh`.

## Assumptions

- `git-guard` and `safe-guard` do not exist on npm (verified 404). Tool-level guardrails therefore remain `AGENTS.md` policy plus review gates; if a real guard extension is published later it can be added without this change.
- Slash commands are a TUI editor feature; print mode loads templates only via `--prompt-template` (verified in pi `docs/usage.md` §33/§226).
- `.oh-my-opencode-pi-stats.json` background failures (136 failed) justify replacement rather than repair; root-causing omp/Pantheon is out of scope.
- The `.pi/side-agent-*` scripts may belong to omp side-agent machinery; their removal is verified during apply and kept if still referenced by surviving tooling.

## Non-Goals

- NOT rewriting the OpenSpec governance workflow or its prompts/skills.
- NOT fixing or debugging omp/Pantheon itself.
- NOT installing packages that do not exist on npm (`git-guard`, `safe-guard`).
- NOT introducing local credential storage or modifying Stytch tenant policy state (rollback for this change is a Git revert only; Stytch runtime SSOT is unaffected).
- NOT building new application features.
