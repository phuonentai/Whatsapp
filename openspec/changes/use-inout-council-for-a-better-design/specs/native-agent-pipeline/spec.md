## MODIFIED Requirements

### Requirement: Council verdict marker contract

`VERDICT.md` SHALL contain exactly one marker line matching `^STATUS: (APPROVED|REJECTED)$` as the first `STATUS:`-prefixed line in the file. The pipeline SHALL treat `STATUS: APPROVED` as proceed, `STATUS: REJECTED` as trigger for the bounded design-revision loop (and as a non-zero halt once the revision cap is exhausted), and any absent or ambiguous marker as an inconclusive halt. The pipeline SHALL parse the marker line rather than substring-grep for "STATUS: REJECTED". A `REJECTED` verdict SHALL additionally contain a numbered "Required design changes" list that the revise stage consumes as its authoritative input. The council template SHALL direct a re-review to read the prior `VERDICT.md` and `revision.md` when both exist in the change directory before evaluating the revised `design.md`.

#### Scenario: Approved verdict proceeds

- **WHEN** `VERDICT.md` contains a line `STATUS: APPROVED` and no other `STATUS:` line precedes it
- **THEN** the pipeline SHALL continue to the next stage

#### Scenario: Rejected verdict triggers the bounded revision loop

- **WHEN** `VERDICT.md` contains a line `STATUS: REJECTED` as the first `STATUS:` line and the revision cap is not exhausted
- **THEN** the pipeline SHALL run the revise stage with the verdict's numbered required design changes as input
- **AND** SHALL re-run the council stage on the revised design
- **AND** SHALL NOT halt immediately

#### Scenario: Revision cap exhausted halts

- **WHEN** the council verdict is `STATUS: REJECTED` and the revision loop has reached its cap (`MAX_COUNCIL_REVISIONS`)
- **THEN** the pipeline SHALL stop with a non-zero exit code
- **AND** SHALL record the halt reason and the revision history in the pipeline log

#### Scenario: Prose mentioning rejection does not halt

- **WHEN** `VERDICT.md` contains prose such as "rejected items: X" but its marker line is `STATUS: APPROVED`
- **THEN** the pipeline SHALL proceed

### Requirement: pipeline.sh orchestrates stages deterministically

`scripts/pipeline.sh` SHALL run stages in order (architect → council → [revise ⇄ council, bounded by `MAX_COUNCIL_REVISIONS`] → sdet → uiux → iso, with council/revise/uiux/iso conditional per gating) using `set -Eeuo pipefail`, SHALL log each stage and revision iteration to `logs/pipeline/`, SHALL halt on any stage failure, SHALL support `--dry-run` (print commands without executing) and per-stage overrides (`--with-council`, `--with-uiux`, `--skip-iso`, `--skip-revise`, `--max-revisions <N>`, `--reset-revisions`), SHALL track revision progress in the change directory (`revision.json`) so the cap is enforced across re-runs, SHALL clear `revision.json` on an approved re-review and on `--reset-revisions`, and SHALL exit non-zero on a rejected council verdict whose revision cap is exhausted.

#### Scenario: Dry run prints commands

- **WHEN** `scripts/pipeline.sh <change> --dry-run` is executed
- **THEN** the script SHALL print each stage command it would run
- **AND** SHALL NOT execute any `pi -p` invocation

#### Scenario: Stage failure halts the pipeline

- **WHEN** any stage command exits non-zero (including a rejected council verdict after the revision cap is exhausted)
- **THEN** the pipeline SHALL stop
- **AND** SHALL exit non-zero
- **AND** SHALL leave a log file under `logs/pipeline/`

#### Scenario: Revision cap is enforced across pipeline re-runs

- **WHEN** `revision.json` in the change directory records the revision count and the pipeline is re-run
- **THEN** the loop SHALL resume from the recorded count rather than restarting
- **AND** SHALL halt with a non-zero exit once the recorded count reaches the cap

#### Scenario: Approved re-review clears the revision counter

- **WHEN** the council re-review returns `STATUS: APPROVED`
- **THEN** the pipeline SHALL delete `revision.json`
- **AND** SHALL proceed to the next stage with a fresh counter

#### Scenario: Reset-revisions starts a fresh revision cycle

- **WHEN** `scripts/pipeline.sh <change> --reset-revisions` is executed
- **THEN** the pipeline SHALL delete `revision.json` before running any stage
- **AND** SHALL NOT delete `revision.md` or any planning artifact

#### Scenario: Skip-revise preserves the immediate halt

- **WHEN** `scripts/pipeline.sh <change> --skip-revise` is executed and the council verdict is `STATUS: REJECTED`
- **THEN** the pipeline SHALL halt immediately with a non-zero exit code without running the revise stage

## ADDED Requirements

### Requirement: Council-driven design revision loop

A dedicated `revise` stage SHALL convert council output into design input: when the council verdict is `REJECTED`, the revise stage SHALL read the verdict's numbered required design changes, revise the change's planning artifacts (`proposal.md`, `design.md`, `tasks.md`, and delta specs under `specs/`) to address each item, and verify coverage before the design is re-submitted to the council. The revise stage SHALL delegate artifact reconciliation to the `opsx-update` workflow logic and SHALL NEVER edit application source code. The revise stage SHALL run with a restricted read/write tool allowlist (no shell); the pipeline SHALL provide the artifact paths via a transient `.revise-context.json` (written from `openspec status --change <name> --json`) and SHALL run `openspec validate <change>` after each pass. Each revise pass SHALL write a machine-parseable `revision.md` sidecar recording the covered required-change numbers and the artifacts revised, and SHALL record the pass in `revision.json`.

#### Scenario: Rejected verdict drives a revision pass

- **WHEN** `VERDICT.md` is `STATUS: REJECTED` with a numbered required-changes list and the cap is not exhausted
- **THEN** the revise stage SHALL revise the planning artifacts to address the numbered items
- **AND** SHALL write `revision.md` recording the covered items and revised artifacts
- **AND** SHALL increment the revision count in `revision.json`

#### Scenario: Revision verifies required-change coverage

- **WHEN** the revise stage completes a pass
- **THEN** SHALL verify each numbered required change from the verdict is addressed by the revised artifacts
- **AND** SHALL record any item it could not address in `revision.md` as residual risk for the council re-review

#### Scenario: Revise never edits application source code

- **WHEN** the revise stage runs
- **THEN** SHALL modify only planning artifacts inside the change directory
- **AND** SHALL NOT create, modify, or delete application source code or files outside the change directory

#### Scenario: Approved or inconclusive verdicts skip revision

- **WHEN** `VERDICT.md` is `STATUS: APPROVED` or the marker is absent/ambiguous
- **THEN** the revise stage SHALL NOT run
- **AND** the pipeline SHALL proceed (approved) or halt inconclusive (absent/ambiguous marker)
