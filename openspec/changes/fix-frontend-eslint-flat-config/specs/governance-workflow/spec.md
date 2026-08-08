## ADDED Requirements

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
