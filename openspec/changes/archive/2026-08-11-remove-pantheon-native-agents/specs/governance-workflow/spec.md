# Governance Workflow

## Purpose

Extends the OpenSpec governance workflow with an adversarial design-review gate: when a change opts into council review, the apply workflow SHALL block completion until the verdict is approved, and the verdict SHALL be recorded in the change's `tasks.md`. All existing gates (premise validation, verification gate, archive decision) are unchanged.

## ADDED Requirements

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
