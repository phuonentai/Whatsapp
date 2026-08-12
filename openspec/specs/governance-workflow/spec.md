## Purpose

Defines the OpenSpec governance workflow: premise validation before delta specs, verification gate before completion, and archive decision.
## Requirements
### Requirement: Premise validation before writing delta specs

The propose workflow SHALL verify factual premises asserted in the proposal against the actual codebase before writing delta specs. Claims about existing components, routes, endpoints, permissions, or build status MUST be verified via codebase inspection; premises that cannot be verified MUST be recorded as explicit Assumptions in the proposal rather than stated as facts.

#### Scenario: Verifiable premise passes

- **WHEN** a proposal asserts "the CRM page exists at `/dashboard/crm`"
- **THEN** the agent SHALL confirm the route file exists in the codebase before proceeding
- **AND** the proposal SHALL proceed to delta spec authoring

#### Scenario: Unverifiable premise is downgraded

- **WHEN** a proposal asserts "a full React UI exists" but the codebase contains no CRUD UI call sites
- **THEN** the agent SHALL not treat the assertion as verified
- **AND** the premise SHALL be recorded under an Assumptions section in the proposal with the evidence gap noted

### Requirement: Verification gate before change completion

The apply workflow SHALL run the verification commands recorded in `tasks.md` (build, lint, test, or other per-task verification criteria) before a change may be reported as complete. If any verification command fails, the change SHALL remain in-progress and the failure SHALL be recorded in `tasks.md`.

#### Scenario: All verification commands pass

- **WHEN** all tasks are implemented and their verification commands complete successfully
- **THEN** the change SHALL be reported as complete
- **AND** the verification results SHALL be recorded

#### Scenario: Verification fails

- **WHEN** the verification command (e.g., `pnpm build`) exits with errors
- **THEN** the change SHALL NOT be reported as complete
- **AND** the failure SHALL be noted in `tasks.md`
- **AND** the change SHALL remain in-progress

### Requirement: Archive decision after completion

The apply workflow SHALL NOT leave a completed change in a silent limbo state. After the verification gate passes, the workflow SHALL either invoke the archive workflow or record an explicit "Archive deferred: <reason>" entry in the change's `tasks.md`.

#### Scenario: Archive on green gate

- **WHEN** the verification gate passes and the user confirms
- **THEN** the agent SHALL run the archive workflow

#### Scenario: Deferred archive is recorded

- **WHEN** the verification gate passes but archiving is intentionally deferred
- **THEN** the agent SHALL append an explicit "Archive deferred: <reason>" entry to the change's `tasks.md`
- **AND** the entry SHALL be visible in subsequent status checks

### Requirement: Archive blocks on incomplete verification tasks

The archive workflow SHALL block archiving when the change has incomplete tasks whose verification criteria are not met, instead of archiving with only a warning. Non-verification gaps (e.g., optional documentation) SHALL remain confirm-and-proceed.

#### Scenario: Incomplete verification task blocks archive

- **WHEN** a change has incomplete verification tasks (e.g., "Run full test suite and verify all pass")
- **THEN** the archive workflow SHALL refuse to archive
- **AND** SHALL explain which verification task is blocking

#### Scenario: Non-verification gap allows confirmed archive

- **WHEN** a change has only non-verification incomplete tasks (e.g., optional doc note)
- **THEN** the archive workflow SHALL warn and require explicit user confirmation before proceeding

### Requirement: Durable workflow documentation

The repository SHALL document the three gates (premise validation, verification gate, archive decision) in AGENTS.md so the contract survives regeneration of `.opencode/skills/` by `openspec update`.

#### Scenario: Gates documented in AGENTS.md

- **WHEN** an agent reads AGENTS.md
- **THEN** the Mandatory Workflow section SHALL describe the premise validation step for proposals, the verification gate for apply, and the archive-after-completion rule
- **AND** it SHALL warn that `openspec update` may regenerate `.opencode/skills/` files and overwrite local edits

### Requirement: Verification commands are runnable

The apply workflow SHALL only record verification commands in `tasks.md` that are runnable with the project's current toolchain, and the repository SHALL maintain tooling (scripts, configs, dependencies) that makes each recorded command executable without workarounds. Frontend lint verification SHALL reference a Next-16-compatible invocation (`pnpm lint` backed by `eslint .` with flat config); a change SHALL NOT defer a verification command as "blocked by pre-existing tooling" without an owning change that restores the tooling.

#### Scenario: Lint command is runnable

- **WHEN** a frontend change records `pnpm lint` as a verification command
- **THEN** the command SHALL execute with the project's flat ESLint configuration
- **AND** the change SHALL NOT be reported complete unless the command exits zero OR the remaining violations are recorded verbatim as a documented baseline in `tasks.md` with a follow-up change created to clear them

#### Scenario: Tooling is broken

- **WHEN** a verification command cannot run with the current toolchain (e.g., `next lint` removed, legacy config incompatible)
- **THEN** the failure SHALL be recorded in `tasks.md`
- **AND** a separate owning change SHALL be created to restore the tooling before the dependent change can pass its verification gate

### Requirement: Council review gate blocks apply until approved

When a change records a required council review — via `routing.json` with `requires_council: true`, or an explicit record in the change's `tasks.md` — the apply workflow SHALL block marking the change complete until `openspec/changes/<change>/VERDICT.md` contains a `STATUS: APPROVED` marker line. The apply workflow SHALL record the council verdict in `tasks.md` (approved or rejected with summary). A rejected verdict SHALL keep the change in-progress until the design is revised and re-reviewed.

#### Scenario: Required council review approved

- **WHEN** a change has `routing.json` with `requires_council: true` and `VERDICT.md` contains `STATUS: APPROVED`
- **THEN** the apply workflow SHALL proceed
- **AND** SHALL record the approved verdict in `tasks.md`

#### Scenario: Required council review rejected blocks completion

- **WHEN** a change has a required council review and `VERDICT.md` contains `STATUS: REJECTED`
- **THEN** the apply workflow SHALL NOT mark the change complete
- **AND** SHALL record the rejected verdict and required changes in `tasks.md`
- **AND** the change SHALL remain in-progress

#### Scenario: Council review not required is advisory

- **WHEN** a change has no required council review but a `VERDICT.md` exists
- **THEN** the apply workflow SHALL treat the verdict as advisory
- **AND** SHALL record it in `tasks.md` without blocking completion on a rejection

