## 1. Apply-side gates

- [x] 1.1 [OPS-GOV] Add a mandatory verification sub-step to the `all_done` branch in `.opencode/commands/opsx-apply.md`: run every verification command recorded in tasks.md; failures keep the change in-progress and are recorded in tasks.md
- [x] 1.2 [OPS-GOV] Mirror the verification gate in `.opencode/skills/openspec-apply-change/SKILL.md` (same wording as 1.1)
- [x] 1.3 [OPS-GOV] Replace the "suggest archive" step in opsx-apply.md + skill: after a green gate, require an archive decision — invoke `/opsx-archive` or record "**Archive deferred:** <reason>" in tasks.md

## 2. Propose-side premise validation

- [x] 2.1 [OPS-GOV] Add a premise-validation step to `.opencode/commands/opsx-propose.md`: verify factual claims (routes, components, endpoints, build status) against the codebase before writing delta specs; demote unverifiable premises to an Assumptions section
- [x] 2.2 [OPS-GOV] Mirror the premise-validation step in `.opencode/skills/openspec-propose/SKILL.md` and add the Assumptions section to its proposal template

## 3. Archive-side guardrails

- [x] 3.1 [OPS-GOV] Update `.opencode/commands/opsx-archive.md` and `.opencode/skills/openspec-archive-change/SKILL.md`: block archiving when incomplete tasks are verification tasks (explain which task blocks); keep confirm-and-proceed for non-verification gaps

## 4. Durable rules

- [x] 4.1 [OPS-GOV] Update `AGENTS.md` Mandatory Workflow: document the three gates (premise validation, verification gate, archive decision) and warn that `openspec update` may regenerate `.opencode/skills/` files
- [x] 4.2 [OPS-GOV] Add `[OPS-GOV]` to the tasks tag rule in `openspec/config.yaml`

## 5. Verification

- [x] 5.1 [OPS-GOV] Run `openspec doctor` and `openspec validate` — both remain green
- [x] 5.2 [OPS-GOV] Consistency review: the gate wording matches between each opsx-*.md command and its corresponding SKILL.md; all gate steps reference the same decisions (D1–D4)
