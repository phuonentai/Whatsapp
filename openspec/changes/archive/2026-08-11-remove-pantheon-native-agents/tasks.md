# Tasks

## 1. Extensions (project-local install) [OPS-GOV]

- [x] 1.1 Run `pi install -l npm:@bopstack/pi-codegraph`; verify `.pi/settings.json` created with the package in `extensions` — verify: `grep -q codegraph .pi/settings.json`
- [x] 1.2 Run `pi install -l npm:pi-playwright`; verify — `grep -q playwright .pi/settings.json`
- [x] 1.3 Run `pi install -l npm:pi-web-search`; verify — `grep -q web-search .pi/settings.json`
- [x] 1.4 Confirm extension tools are exposed: run `pi -p "list the tool names you can call"` (or inspect the startup manifest) and confirm `codegraph`/`playwright`/`web_search` tool families are present; do NOT add `git-guard` or `safe-guard` (not published on npm) — verify: `.pi/settings.json` contains exactly the three packages (verified via `pi list`: 3 project packages, no guard packages)

## 2. Prompt templates (.pi/prompts/) [OPS-GOV]

- [x] 2.1 Create `.pi/prompts/council.md`: YAML frontmatter with `description`; instructs adversarial review of `openspec/changes/<change>/design.md` + `proposal.md` in three personas (Staff Security Engineer, Staff DBA, SRE); requires output `VERDICT.md` with exactly one marker line `^STATUS: (APPROVED|REJECTED)$` as the first `STATUS:` line; on rejection lists required changes — verify: frontmatter parses, marker contract documented
- [x] 2.2 Create `.pi/prompts/uiux.md`: frontmatter with `description`; Playwright visual + accessibility QA at 390x844, 768x1024, 1440x900; writes `openspec/changes/<change>/qa/screenshots/` and `qa/REPORT.md`; assumes dev server via `scripts/run-qa-server.sh` — verify: file exists with frontmatter
- [x] 2.3 Create `.pi/prompts/iso.md`: frontmatter with `description`; updates `docs/compliance/ISO_TRACEABILITY_MATRIX.md` using file paths / symbol names / AST matches (codegraph) as evidence; maps to ISO 27001 / 9001 / 42001 — verify: file exists with frontmatter
- [x] 2.4 Create `.pi/prompts/architect.md`: thin wrapper that delegates to the `/opsx-propose` workflow (no parallel propose implementation) — verify: file exists, delegates by reference
- [x] 2.5 Create `.pi/prompts/sdet.md`: thin wrapper that delegates to the `/opsx-apply` workflow (keeps the verification gate) — verify: file exists, delegates by reference
- [x] 2.6 Smoke-test one headless stage: run `pi -p --prompt-template .pi/prompts/council.md "<change-name> review design.md"` against a sample change and confirm `VERDICT.md` is produced with a valid `STATUS:` marker — verify: marker line matches `^STATUS: (APPROVED|REJECTED)$`

## 3. Pipeline orchestrator (scripts/pipeline.sh) [OPS-GOV]

- [x] 3.1 Create `scripts/pipeline.sh` with `set -Eeuo pipefail`: stage order architect → council → sdet → uiux → iso; every stage invoked as `pi -p --prompt-template .pi/prompts/<stage>.md "<instruction>"` (never slash commands); per-stage logs under `logs/pipeline/`; halt on any non-zero exit; exit 2 on `STATUS: REJECTED` — verify: `bash -n scripts/pipeline.sh`
- [x] 3.2 Implement council verdict parsing by first `^STATUS:` marker line (APPROVED → proceed, REJECTED → halt, absent/ambiguous → inconclusive halt); include fixture tests for a verdict whose prose mentions "rejected" but whose marker is APPROVED — verify: `scripts/pipeline.sh <sample> --dry-run` passes and marker-parse logic handled
- [x] 3.3 Implement advisory `routing.json` gating with defaults (council only when `complexity: high`, playwright only when flagged, iso always) and overrides `--with-council`, `--with-uiux`, `--skip-iso` — verify: dry-run with/without routing.json prints expected stage sets
- [x] 3.4 Implement `--dry-run` mode (print exact commands, execute nothing) and ensure `scripts/pipeline.sh <change> --dry-run` exits 0 on a sample change — verify: `scripts/pipeline.sh <sample-change> --dry-run`
- [x] 3.5 Ensure `logs/pipeline/` exists (created on first run or by script) and `.gitignore` covers it if not already — verify: `mkdir -p logs/pipeline` + `.gitignore` entry present

## 4. Remove omp/Pantheon [OPS-GOV]

- [x] 4.1 Verify ownership of side-agent machinery: `grep -rln "side-agent" --include="*.sh" --include="*.md" --include="*.json" .` and decide keep/remove per design D7 — verify: decision recorded in this file

  **Decision (2026-08-11): KEEP and document.** `.pi/side-agent-start.sh`, `.pi/side-agent-finish.sh`, `.pi/side-agents/`, and `.pi/side-agent-skills/` are **not** omp/Pantheon machinery. They belong to the `pi-side-agents` worktree agent mechanism (TypeScript extension; `PI_SIDE_PARENT_REPO` / `PI_SIDE_AGENT_ID` env vars; git worktree + merge-lock protocol). `scripts/bootstrap-stack.sh` still installs `npm:pi-side-agents`, so the machinery is referenced by surviving tooling and SHALL remain. Documented as non-omp machinery in `AGENTS.md` (Agent Pipeline section) and `BOOTSTRAP_REPORT.md`. No change to omp artifact removal (the four omp paths are separate).

- [x] 4.2 `git rm` omp/Pantheon artifacts: `omp-install.sh`, `.oh-my-opencode-pi-workflow.json`, `.oh-my-opencode-pi-stats.json`, `.oh-my-opencode-pi-debug/` — verify: none of the four paths exist (`ls` returns nothing)
- [x] 4.3 Remove omp/Pantheon references from root docs (`BOOTSTRAP_REPORT.md`, `DEVELOPMENT.md`, `README.md`, `AI-architecture.md`) where they assert it as the active pipeline; keep historical mention only if meaningful — verify: `grep -rin "oh-my-opencode\|pantheon" <docs>` returns only intentional historical notes (none by default)
- [x] 4.4 Add a short "Agent Pipeline" section to `AGENTS.md` documenting `/architect`, `/council`, `/sdet`, `/uiux`, `/iso`, `scripts/pipeline.sh`, and the `.pi/settings.json` extension set; keep `AGENTS.md` minimal — verify: section present, no OpenSpec gate text altered

## 5. Governance & compliance wiring [OPS-GOV]

- [x] 5.1 Confirm the `governance-workflow` delta spec (council gate blocks apply until `STATUS: APPROVED`; verdict recorded in `tasks.md`) matches apply behavior — verify: `openspec validate` passes
- [x] 5.2 Run `openspec validate` on the change and fix any errors — verify: `openspec validate` exits 0
- [x] 5.3 End-to-end smoke: run `scripts/pipeline.sh <sample-change> --dry-run` then a live `--with-council` run on a sample change; confirm artifacts (`VERDICT.md`, `qa/REPORT.md` when uiux runs) and logs — verify: pipeline exits 0 and artifacts exist
- [x] 5.4 Update `BOOTSTRAP_REPORT.md` status section if it documents the previous bootstrap state (mark omp replacement) — verify: report reflects the new stack

## Verification summary (run all at completion)

- [x] V.1 `bash -n scripts/pipeline.sh`
- [x] V.2 `grep -q codegraph .pi/settings.json && grep -q playwright .pi/settings.json && grep -q web-search .pi/settings.json`
- [x] V.3 `test ! -e omp-install.sh && test ! -e .oh-my-opencode-pi-workflow.json && test ! -e .oh-my-opencode-pi-stats.json && test ! -d .oh-my-opencode-pi-debug`
- [x] V.4 `scripts/pipeline.sh <sample-change> --dry-run` exits 0
- [x] V.5 `openspec validate` exits 0 (`openspec validate --changes`: 32 passed; `--specs`: 74 passed)

## Council verdict (required by `routing.json` → `requires_council: true`)

- [x] **Verdict: STATUS: APPROVED** (2026-08-11) — see `VERDICT.md`. No design-blocking defects; no rejection criteria triggered.
- [x] Verdict recorded in this file per the `governance-workflow` delta (approved with summary).
- [x] Archive-blocking findings from the verdict closed:
  - #1 verdict record — this entry.
  - #2 pinned-version claim — `.pi/settings.json` pins `npm:@bopstack/pi-codegraph@0.1.1`, `npm:pi-playwright@0.1.1`, `npm:pi-web-search@1.3.1`; `AGENTS.md` wording now matches.
  - #3 fixture reference — `scripts/tests/fixtures/` + `scripts/tests/test-verdict-parse.sh` exist; prose-mentions-"rejected" fixture passes (`bash scripts/tests/test-verdict-parse.sh` exits 0).
  - #4 exit-code reporting — `run_stage` in `scripts/pipeline.sh` captures `rc=$?` from the failed command directly (no `!` negation); verified reporting real exit code.


## Apply session notes (2026-08-11)

- **Rogue-process incident:** an aborted headless council run left an orphaned `pi -p` process that mutated the tree (created `scripts/pipeline.sh` + tests, edited `AGENTS.md`, `BOOTSTRAP_REPORT.md`, marked tasks complete) before being killed. The living `openspec/specs/governance-workflow/spec.md` was verified clean (no unauthorized fold-in). Lesson: headless stages need tool allowlists and role-framed prompts; council now runs with `--tools read,write`.
- **Council gate cycle (real):** first headless council run → `STATUS: REJECTED` with 2 blocking findings (missing `--approve` for headless project trust; omp-era `bootstrap-stack.sh`). Fixed both, re-reviewed → `STATUS: APPROVED` with 4 process cleanups (verdict record, pinned extension versions, fixture check, `run_stage` exit-code bug). All closed; see `VERDICT.md` and the Council verdict section above.
- **iso stage:** first two headless runs drifted into "completion assistant" output without updating the matrix; a role-framed instruction ("Audit the compliance evidence... Update the matrix. Do nothing else.") produced the evidence-row update (see `docs/compliance/ISO_TRACEABILITY_MATRIX.md`). Prompt templates `council.md`/`iso.md`/`uiux.md` carry role framing + finish conditions for headless use.
- **uiux stage** was not run live (no UI surface in this tooling-only change; requires a dev server). Gated off via `routing.json` (`requires_playwright: false`).
- **Manual ISO matrix update** was performed during apply; the hardened iso stage subsequently rewrote those rows with stronger evidence paths — kept.
