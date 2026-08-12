# BOOTSTRAP_REPORT

Generated: 2026-08-11T04:59:03Z
Working directory: /home/phuongbinhnguyentai/project/Whatsapp

> **STATUS UPDATE (2026-08-11):** The omp/Pantheon orchestration layer recorded in this report was
> detected as flaky (136 failed background tasks in its own telemetry) and has been **replaced** by the
> deterministic native Pi agent pipeline: `scripts/pipeline.sh` + `.pi/prompts/{architect,council,sdet,uiux,iso}.md`.
> See `openspec/changes/remove-pantheon-native-agents/`. omp artifacts (`omp-install.sh`,
> `.oh-my-opencode-pi-*.json`, `.oh-my-opencode-pi-debug/`) are removed from the repository; this report is
> kept as a historical record of the original bootstrap.

## 1. Detected Pi Runtime

- Runtime (original bootstrap): omp
- Version: omp/17.2.12
- Runtime (current, post-replacement): native Pi (`pi -p --prompt-template` pipeline)

## 2. OpenSpec Detection

- Result: OpenSpec CLI 1.6.0

## 3. Installed System Dependencies

- node:/home/phuongbinhnguyentai/.config/nvm/versions/node/v24.18.0/bin/node v24.18.0
- npm:/home/phuongbinhnguyentai/.config/nvm/versions/node/v24.18.0/bin/npm
- git:/usr/bin/git
- curl:/usr/bin/curl
- jq:/usr/bin/jq
- lsof:/usr/bin/lsof
- gh:/usr/bin/gh

## 4. Installed Pi Packages

- npm:oh-my-opencode-pi (historical — omp layer since replaced by the native pipeline)
- npm:pi-side-agents
- npm:@bopstack/pi-codegraph
- npm:pi-playwright
- npm:pi-web-search

> Current project-local extension set (`.pi/settings.json`, per the native pipeline):
> `@bopstack/pi-codegraph`, `pi-playwright`, `pi-web-search`. `oh-my-opencode-pi` is no longer installed.

## 5. Failed Pi Packages

- npm:git-guard
- npm:safe-guard
  - Reason: package does not exist on the npm registry (HTTP 404 on resolution, confirmed via `npm view` and bun). Non-critical per bootstrap policy; excluded from the plugin list (omp) and NOT part of the native pipeline's `.pi/settings.json`.

## 6. CodeGraph Status

- initialization failed (warning)
  - Detail: no `codegraph` CLI exists in this environment (`codegraph init`, `npx codegraph init`, and `omp run codegraph init` all failed). The `@bopstack/pi-codegraph@0.1.1` extension IS installed and enabled via `~/.omp/agent/settings.json` (historical); it exposes no standalone CLI binary (bin: null), so repo indexing must be triggered through omp/pi tooling instead. In the current native pipeline, codegraph is invoked through the `codegraph` tool exposed by the project-local `@bopstack/pi-codegraph` extension.

## 7. Playwright Status

- chromium + system deps installed

## 8. Created Configuration Files (historical — omp layer)

- /home/phuongbinhnguyentai/.omp/agent/settings.json
- /home/phuongbinhnguyentai/.omp/prompts/architect.md
- /home/phuongbinhnguyentai/.omp/prompts/sdet.md
- /home/phuongbinhnguyentai/.omp/prompts/uiux.md
- /home/phuongbinhnguyentai/.omp/prompts/iso-auditor.md
- /home/phuongbinhnguyentai/.omp/prompts/autopilot.md
- /home/phuongbinhnguyentai/project/Whatsapp/scripts/run-qa-server.sh
- /home/phuongbinhnguyentai/project/Whatsapp/scripts/bootstrap-stack.sh
- /home/phuongbinhnguyentai/project/Whatsapp/openspec/changes/00-bootstrap-health-check/ (proposal.md, design.md, tasks.md, routing.json)

> The current native pipeline's prompt templates live in the repository at
> `.pi/prompts/{architect,council,sdet,uiux,iso}.md` (project-local, git-tracked) — not under `~/.omp/`.

## 9. Created Prompt Files

- Historical omp prompts: /home/phuongbinhnguyentai/.omp/prompts/{architect,sdet,uiux,iso-auditor,autopilot}.md
- Current native pipeline prompts: /home/phuongbinhnguyentai/project/Whatsapp/.pi/prompts/{architect,council,sdet,uiux,iso}.md

## 10. Created Scripts

- /home/phuongbinhnguyentai/project/Whatsapp/scripts/run-qa-server.sh
- /home/phuongbinhnguyentai/project/Whatsapp/scripts/bootstrap-stack.sh
- /home/phuongbinhnguyentai/project/Whatsapp/scripts/pipeline.sh (native pipeline orchestrator, replaces omp)

## 11. Demo Spec Location

- /home/phuongbinhnguyentai/project/Whatsapp/openspec/changes/00-bootstrap-health-check/ (proposal.md, design.md, tasks.md, routing.json)

## 12. Demo Implementation Status

- not implemented (BOOTSTRAP_IMPLEMENT_DEMO=false)

## 13. Verification Results

PASS (original bootstrap):
- omp responds to --help
- settings.json is valid JSON
- architect.md exists
- sdet.md exists
- uiux.md exists
- iso-auditor.md exists
- autopilot.md exists
- openspec/changes exists
- ISO_TRACEABILITY_MATRIX.md exists
- run-qa-server.sh exists and executable
- logs/bootstrap exists

FAIL:
- none

## 14. Remaining Manual Actions

- Initialize the CodeGraph index through the project-local `@bopstack/pi-codegraph` extension (no standalone CLI exists; use the pi `codegraph` tool).
- git-guard and safe-guard are not on the npm registry (HTTP 404); they are intentionally NOT part of the native pipeline's extension set. Tool-level guardrails remain `AGENTS.md` policy + review gates.
- Playwright system deps were installed with apt-get; the install-deps step may require re-running after OS upgrades.

## 15. Blockers

- none
