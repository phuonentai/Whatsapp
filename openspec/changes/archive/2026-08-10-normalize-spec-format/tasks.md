## 1. Normalize living spec framing [OPS-GOV]

- [x] 1.1 Convert `## ADDED Requirements` / `## MODIFIED Requirements` to a single `## Requirements` header in all 38 failing specs under `openspec/specs/`, merging concatenated section blocks (6 specs: `crm-core-data`, `crm-frontend`, `stytch-authorization`, `whatsapp-config-api`, `whatsapp-config-frontend`, `whatsapp-webhook-ingress`) into one coherent `## Requirements` section with all content preserved.
- [x] 1.2 Prepend a brief `## Purpose` section to each of the 36 specs missing one (both `module-registry` and `vertical-playbooks` already have it). Reparent the orphaned `stytch-go v16→v18` body in `stytch-authorization/spec.md` line 1 under a proper `### Requirement: stytch-go upgraded from v16 to v18` header.
- [x] 1.3 Verify no content lost during framing merge: requirement + scenario title counts in each converted spec MUST equal the pre-conversion counts.

## 2. Restore drifted content [OPS-GOV]

- [x] 2.1 `stytch-authorization`: add the 4 restored requirements (Role normalization, RBAC API endpoint authentication, RBACService implementation backed by Stytch policy, DTOs retained as API contract) with their scenarios from the change delta spec `specs/stytch-authorization/spec.md`. Do NOT add the abandoned password-auth requirement.
- [x] 2.2 `crm-core-data`: add the 3 restored requirements (Conversation entity with status tracking, Message entity with WhatsApp-specific fields, Event subscriber processes MessageReceived events asynchronously) with their scenarios from `specs/crm-core-data/spec.md`.
- [x] 2.3 Run `openspec validate --specs` and confirm 52/52 pass. Confirmed-valid specs MUST remain unchanged by the content restore steps.

## 3. Add CI spec-validation gate [OPS-GOV]

- [x] 3.1 Add a `spec-validation` job to `.github/workflows/ci.yml` that runs `openspec validate --specs` and fails on any invalid spec. Verify with `actionlint` or YAML lint if available in the repo tooling.

## 4. Verification

- [x] 4.1 Run `openspec validate --specs` → expected: 52 passed, 0 failed (Totals line).
- [x] 4.2 Run `openspec validate normalize-spec-format --type change` → change deltas valid.
- [x] 4.3 Confirm no git diff outside `openspec/specs/`, `openspec/changes/normalize-spec-format/`, and `.github/workflows/ci.yml`.

## Verification Results

- `openspec validate --specs` → **passed** — Totals: 54 passed, 0 failed (54 items). Count grew from 52 to 54 because `add-ci-pipeline` was archived mid-session, folding `ci-pipeline` and `test-tooling` living specs.
- `openspec validate normalize-spec-format --type change` → **passed** — "Change 'normalize-spec-format' is valid".
- `openspec validate frontend-api-client --type spec` → **passed** (spec restored from archived delta, empty-file fold defect fixed).
- Git scope check → **passed** — all edits confined to `openspec/specs/` (20 tracked spec files + 31 pre-existing untracked spec dirs), `openspec/changes/normalize-spec-format/`, and `.github/workflows/ci.yml`. Pre-existing uncommitted work elsewhere untouched.
- YAML lint of `.github/workflows/ci.yml` → **passed** (python yaml.safe_load). `actionlint` not installed in repo tooling.

**Notes:**
- Task 1.1 normalized 36 specs (not 38 as proposed): `frontend-api-client` had no `## ADDED/MODIFIED` header to detect (empty file) and was handled in task 2 as a content restore; `lean-data-isolation` already had correct framing and failed only on a latent SHALL/MUST defect fixed separately.
- Content-loss check (task 1.3): req/scenario title counts identical pre/post across all converted specs. Only `frontend-api-client` changed (0→4 reqs, intentional restore).

