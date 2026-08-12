# native-agent-pipeline Specification

## Purpose
TBD - created by archiving change remove-pantheon-native-agents. Update Purpose after archive.
## Requirements
### Requirement: Prompt-template agent stages exist as native Pi prompt templates

The system SHALL provide agent stages as prompt templates under `.pi/prompts/` with YAML frontmatter containing a `description` field, so each stage is invocable both interactively (`/council`, `/uiux`, `/iso`) and headlessly. The stages SHALL be: `council.md` (adversarial multi-persona review of `design.md` producing `VERDICT.md`), `uiux.md` (Playwright visual and accessibility QA producing `openspec/changes/<change>/qa/screenshots/` and `qa/REPORT.md`), `iso.md` (compliance traceability updating `docs/compliance/ISO_TRACEABILITY_MATRIX.md`), `architect.md` (delegates to the `/opsx-propose` workflow), and `sdet.md` (delegates to the `/opsx-apply` workflow). The `architect` and `sdet` stages SHALL NOT re-implement OpenSpec propose/apply behavior.

#### Scenario: Council stage is invocable

- **WHEN** a user types `/council` in the pi TUI or the pipeline invokes `pi -p --prompt-template .pi/prompts/council.md` with a change name
- **THEN** the council prompt template SHALL be loaded
- **AND** the agent SHALL review `openspec/changes/<change>/design.md` and `proposal.md` in five personas (security engineer, DBA, SRE, product/GTM, Colombia IT & market)
- **AND** for a market-in-scope change the agent SHALL additionally apply the council market-read lens
- **AND** SHALL write `openspec/changes/<change>/VERDICT.md`

#### Scenario: Architect and sdet delegate to OpenSpec

- **WHEN** the `architect` or `sdet` stage is invoked
- **THEN** the stage SHALL instruct the agent to run the existing `/opsx-propose` or `/opsx-apply` workflow respectively
- **AND** SHALL NOT introduce a parallel proposal/apply implementation

### Requirement: Headless invocation uses --prompt-template

All non-interactive pipeline invocations SHALL load prompt templates via `pi -p --prompt-template <path> "..."`. The pipeline SHALL NOT rely on slash-command expansion, which is TUI-only. Each headless stage SHALL read its input artifacts from the change directory on disk rather than from session memory.

#### Scenario: Pipeline loads a stage headlessly

- **WHEN** `scripts/pipeline.sh` runs a stage
- **THEN** the command SHALL be of the form `pi -p --prompt-template .pi/prompts/<stage>.md "<stage instruction with change name>"`
- **AND** the stage SHALL pass or fail based on its exit code and any written artifacts

### Requirement: Council verdict marker contract

`VERDICT.md` SHALL contain exactly one marker line matching `^STATUS: (APPROVED|REJECTED)$` as the first `STATUS:`-prefixed line in the file. The pipeline SHALL treat `STATUS: APPROVED` as proceed, `STATUS: REJECTED` as halt (non-zero exit), and any absent or ambiguous marker as an inconclusive halt. The pipeline SHALL parse the marker line rather than substring-grep for "STATUS: REJECTED". `VERDICT.md` SHALL also contain a `MARKET: PASS | CONDITIONAL | FAIL | N/A` line; the `MARKET:` line is not `STATUS:`-prefixed and SHALL NOT affect the marker parse. For market-in-scope changes, a `MARKET: FAIL` verdict without an explicitly accepted residual recorded in the design's `## Market Risk` section SHALL NOT be accompanied by `STATUS: APPROVED`, and a `MARKET: CONDITIONAL` verdict SHALL NOT be accompanied by `STATUS: APPROVED` unless each condition is fixed by design revision or recorded as an accepted residual.

#### Scenario: Approved verdict proceeds

- **WHEN** `VERDICT.md` contains a line `STATUS: APPROVED` and no other `STATUS:` line precedes it
- **THEN** the pipeline SHALL continue to the next stage

#### Scenario: Rejected verdict halts

- **WHEN** `VERDICT.md` contains a line `STATUS: REJECTED` as the first `STATUS:` line
- **THEN** the pipeline SHALL stop with a non-zero exit code
- **AND** SHALL record the halt reason in the pipeline log

#### Scenario: Market line does not affect the marker parse

- **WHEN** `VERDICT.md` contains `MARKET: FAIL` as its first `MARKET:` line and `STATUS: APPROVED` as its first `STATUS:` line for a market-in-scope change with no accepted residual
- **THEN** the `MARKET:` line SHALL NOT change the marker parse result
- **AND** the verdict SHALL be treated as a contract violation by the council prompt (the council SHALL NOT produce this combination)

#### Scenario: Prose mentioning rejection does not halt

- **WHEN** `VERDICT.md` contains prose such as "rejected items: X" but its marker line is `STATUS: APPROVED`
- **THEN** the pipeline SHALL proceed

### Requirement: pipeline.sh orchestrates stages deterministically

`scripts/pipeline.sh` SHALL run stages in order (architect → council → sdet → uiux → iso, with council/uiux/iso conditional per gating) using `set -Eeuo pipefail`, SHALL log each stage to `logs/pipeline/`, SHALL halt on any stage failure, SHALL support `--dry-run` (print commands without executing) and per-stage overrides (`--with-council`, `--with-uiux`, `--skip-iso`), and SHALL exit non-zero on a rejected council verdict.

#### Scenario: Dry run prints commands

- **WHEN** `scripts/pipeline.sh <change> --dry-run` is executed
- **THEN** the script SHALL print each stage command it would run
- **AND** SHALL NOT execute any `pi -p` invocation

#### Scenario: Stage failure halts the pipeline

- **WHEN** any stage command exits non-zero (including a rejected council verdict)
- **THEN** the pipeline SHALL stop
- **AND** SHALL exit non-zero
- **AND** SHALL leave a log file under `logs/pipeline/`

### Requirement: Advisory routing.json sidecar

A change directory MAY contain `routing.json` with optional fields `requires_council`, `requires_playwright`, `requires_iso`, `requires_market_read`, and `complexity` (`low` | `medium` | `high`). Absent or partial `routing.json` SHALL fall back to defaults: council required only when complexity is `high` or `requires_market_read` is `true`, playwright required only when explicitly flagged, iso always. `routing.json` SHALL be advisory and SHALL NOT replace or override OpenSpec artifacts or the lifecycle gates.

#### Scenario: High complexity change requires council by default

- **WHEN** `routing.json` contains `{"complexity": "high"}` and no `requires_council` field
- **THEN** the pipeline SHALL run the council stage

#### Scenario: Market-read routing requires council at low complexity

- **WHEN** `routing.json` contains `{"requires_market_read": true, "complexity": "medium"}` and no `requires_council` field
- **THEN** the pipeline SHALL run the council stage
- **AND** the council SHALL apply the market-read lens

#### Scenario: Explicit override forces a stage

- **WHEN** `scripts/pipeline.sh <change> --with-uiux` is executed on a change whose `routing.json` does not require playwright
- **THEN** the pipeline SHALL run the uiux stage

### Requirement: Project-local extension set declared in .pi/settings.json

The repository SHALL declare exactly the verified project-local extensions in `.pi/settings.json` via `pi install -l`: `@bopstack/pi-codegraph`, `pi-playwright`, and `pi-web-search`. The repository SHALL NOT declare `git-guard` or `safe-guard` (not published on npm). Tool-level guardrails SHALL remain `AGENTS.md` policy and review gates.

#### Scenario: Extensions installed project-locally

- **WHEN** `pi install -l npm:@bopstack/pi-codegraph`, `npm:pi-playwright`, and `npm:pi-web-search` have been run
- **THEN** `.pi/settings.json` SHALL list the three packages in its extensions array
- **AND** the pi tool manifest SHALL expose tools provided by those extensions

### Requirement: omp/Pantheon artifacts removed from the repository

The repository SHALL NOT contain omp/Pantheon orchestration artifacts: `omp-install.sh`, `.oh-my-opencode-pi-workflow.json`, `.oh-my-opencode-pi-stats.json`, or `.oh-my-opencode-pi-debug/`. Documentation SHALL NOT assert omp/Pantheon as the active pipeline. Any `.pi/side-agent-*.sh` or `.pi/side-agents/` files SHALL be removed only if no surviving tooling references them; otherwise they SHALL be kept and documented as non-omp machinery.

#### Scenario: Omp artifacts absent after removal

- **WHEN** the change is applied
- **THEN** `omp-install.sh`, `.oh-my-opencode-pi-workflow.json`, `.oh-my-opencode-pi-stats.json`, and `.oh-my-opencode-pi-debug/` SHALL NOT exist in the repository root

#### Scenario: Side-agent scripts kept when referenced

- **WHEN** `.pi/side-agent-start.sh` or `.pi/side-agent-finish.sh` is referenced by surviving tooling (e.g., `scripts/bootstrap-stack.sh`)
- **THEN** the scripts SHALL remain in the repository
- **AND** their role SHALL be documented as independent of omp/Pantheon

### Requirement: Council market-read lens

For market-in-scope changes — those touching any of (a) billing/pricing/paywall/quotas/credits, (b) AI usage metering, model routing, LLM cost, or agent behavior, (c) the WhatsApp/Meta channel (ingress/outbound/templates/campaigns), (d) compliance (Ley 1581, the local invoicing ecosystem — DIAN, tools like SIIGO — consumer law, data retention), or (e) the marketing site, signup funnel, onboarding, or activation — the council SHALL review market viability in addition to engineering correctness: unit economics (USD AI cost per action vs price point in COP and plan/fee margins), pricing coherence with feature gating and credit guards, activation/churn surface, competitive substitution (Meta native WhatsApp AI, local incumbents), and channel/regulatory dependency (WhatsApp Business Platform policy drift, Ley 1581, the local invoicing ecosystem — DIAN electronic invoicing via tools like SIIGO — Ley 1480). In-scope designs SHALL include a `## Market & Unit Economics` section and a `## Market Risk` section (with named risks, owners, and triggers for accepted residuals); their absence SHALL be flagged as a design defect. Council market findings SHALL be grounded in repository evidence; external market facts (provider pricing, regulator posture, competitor claims) SHALL be marked as assumptions to validate and SHALL NOT be asserted as premises. Out-of-scope changes SHALL record `MARKET: N/A` and require no market sections.

#### Scenario: Fail verdict without residual is rejected

- **WHEN** a market-in-scope change is reviewed, the design's `## Market Risk` records no accepted residual, and the council finds negative projected unit margin or an unmitigated existential channel risk
- **THEN** the council SHALL write `MARKET: FAIL`
- **AND** SHALL write `STATUS: REJECTED`

#### Scenario: Conditional verdict permits approval with recorded conditions

- **WHEN** the council finds a market risk whose mitigation depends on evidence not in the design, and the design records the condition as an accepted residual with owner and trigger
- **THEN** the council SHALL write `MARKET: CONDITIONAL`
- **AND** MAY write `STATUS: APPROVED`

#### Scenario: In-scope design missing market sections

- **WHEN** a market-in-scope change's `design.md` lacks the `## Market & Unit Economics` or `## Market Risk` section
- **THEN** the council SHALL flag the absence as a design defect
- **AND** SHALL issue `REJECTED` if the defect is not remedied in revision

#### Scenario: Out-of-scope change records N/A

- **WHEN** a change touches none of the market-in-scope surfaces
- **THEN** the council SHALL write `MARKET: N/A`
- **AND** SHALL NOT require market sections in `design.md`

#### Scenario: Unverified external market premise is flagged

- **WHEN** a design asserts an external market fact (provider pricing, regulator posture, competitor capability) as a premise without repository evidence
- **THEN** the council SHALL flag the claim as an assumption to validate
- **AND** SHALL record the flag in the Market Read findings

