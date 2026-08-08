## ADDED Requirements

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
