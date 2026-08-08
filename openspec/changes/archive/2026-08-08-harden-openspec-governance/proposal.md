## Why

The spec-driven workflow (AGENTS.md → `.opencode/commands/opsx-*.md` → `.opencode/skills/openspec-*/SKILL.md`) has three structural gaps that let code drift from specs without detection: completed changes are never archived (deltas left unfolded), proposals assert factual premises that are never checked against the codebase, and apply reports completion without running the verification commands the tasks themselves define.

## What Changes

- **Verification gate in apply**: `opsx-apply.md` and `openspec-apply-change/SKILL.md` gain a mandatory verification step — before a change may be reported as complete, the agent MUST run the verification commands recorded in `tasks.md` and record results. Failures keep the change in-progress.
- **Archive enforcement after completion**: once the verification gate passes, the apply workflow MUST require an archive decision — invoke `/opsx-archive` by default, or record an explicit "Archive deferred: <reason>" entry. No silent limbo between "complete" and "archived".
- **Premise validation at proposal time**: `opsx-propose.md` and `openspec-propose/SKILL.md` gain a premise-validation step — factual claims (components, routes, endpoints, build status) MUST be verified against the codebase before writing delta specs; unverifiable premises are demoted to explicit Assumptions in the proposal.
- **Archive-side guardrails**: `opsx-archive.md` and `openspec-archive-change/SKILL.md` block archiving when incomplete tasks are verification tasks (instead of "inform and confirm"), while preserving confirm-and-proceed for non-verification gaps.
- **Durable rules**: AGENTS.md Mandatory Workflow documents the three gates and the `openspec update` regeneration risk; `openspec/config.yaml` tasks rule gains the `[OPS-GOV]` tag prefix for workflow/tooling tasks.

## Capabilities

### New Capabilities

- `governance-workflow`: the OpenSpec change lifecycle contract enforced by the opencode commands and skills — premise validation at proposal time, verification gates at apply time, and archive enforcement after completion.

### Modified Capabilities

- none (no existing application capability's requirements change; `.opencode/`, `AGENTS.md`, and `openspec/config.yaml` are the implementation surface of the new capability above.)

## Impact

- **Repo**: edits to `.opencode/commands/opsx-{apply,propose,archive}.md`, `.opencode/skills/openspec-{apply-change,propose,archive-change}/SKILL.md`, `AGENTS.md`, `openspec/config.yaml`.
- **Auth boundary**: no change. This change does not touch the Stytch B2B runtime SSOT — no credentials, sessions, or identity data are added locally; `stytch_member_id`/`stytch_organization_id` linkage and all Stytch B2B API contracts (JWKS verification, webhook signatures, circuit breaker) are unaffected.
- **Rollback**: all edits are plain-text reversions in git; no data migration. `config.yaml` tag addition is purely additive. Stytch tenant policy state requires no rollback because this change never mutates Stytch state.
- **Verification**: `openspec doctor` and `openspec validate` must remain green; every edited workflow document includes its new gate in the step text; a consistency review confirms the same gate wording appears in both the command and the skill for each workflow.

## Non-Goals

- Repairing the observed drift itself (the 39 TypeScript errors, missing sidebar entries, `add-whatsapp-inbox` folding/archiving, `add-crm-e2e-tests` spec rewrite) — tracked separately.
- Changing the `openspec` CLI binary or its behavior.
- Editing application code in `go-b2b-starter/` or `next_b2b_starter/`.
- Local storage of credentials, MFA tokens, or session tokens (forbidden by constitution; unchanged).
