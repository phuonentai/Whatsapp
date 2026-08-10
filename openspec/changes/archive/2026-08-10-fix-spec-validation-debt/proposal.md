# Proposal: fix-spec-validation-debt

## Why

The OpenSpec tree — the repo's declared behavioural source of truth — fails its own validation: `openspec validate --specs` reports 38 of 54 living specs with issues (2 ERROR, 33 WARNING, 2 INFO). The CI gate added by `2026-08-10-add-ci-pipeline` runs `npx openspec validate --specs` but that command exits 0 even with failures, so the gate is green while the tree is red — governance gate 5 (verification) is silently undermined. One spec file (`frontend-api-client`) is completely empty (0 lines), and the `stytch-authorization` living spec has lost 4 requirements that are present in the archived `2026-07-28-stytch-debt-cleanup` delta and implemented in code.

## What Changes

- **Write `frontend-api-client` spec from scratch** — the file is empty; the capability exists in code (`next_b2b_starter/lib/api/api/client/api-client.ts`, 733 lines, plus 14 typed repositories). Spec authored from verified code, not invented.
- **Fix `lean-data-isolation` ERROR** — requirement "Optional PostgreSQL RLS policy for defense-in-depth" violates the requirement-text rule (must contain SHALL or MUST in the statement).
- **Expand 33 stub `## Purpose` sections to >50 characters** (validator minimum). Currently one-line stubs like "Specification for tag management." These are framing-only edits; requirement content is untouched.
- **Restore 4 requirements to `stytch-authorization`** — Role normalization, RBAC API endpoint authentication, RBACService implementation backed by Stytch policy, DTOs retained as API contract — from the archived `2026-07-28-stytch-debt-cleanup` delta. Verified present in code (`stytch_rbac_service.go`, `rbac_policy.go`, RBAC routes) and NOT superseded by any newer delta.
- **Harden the CI spec-validation gate** (`.github/workflows/ci.yml` `spec-validation` job) — run both `openspec validate --specs` (catches structure errors) and `openspec validate --specs --strict` (catches brief-Purpose warnings); either failing fails the job. Strict mode alone misses structural errors (verified: empty spec validates `True` under `--strict`); non-strict alone exits 0 despite errors.
- **Trim 2 INFO-level long requirement texts** in `stytch-authorization` and `vertical-playbooks` where the bloat is editorial, so the tree is fully clean at every severity level.

## Capabilities

### New Capabilities

- `frontend-api-client`: Contract for the Next.js frontend API client — `ApiClient` class (base URL resolution server/client side, typed request helpers, token handling), typed repository layer pattern, and envelope/unwrap conventions. Currently an empty spec file.

### Modified Capabilities

- `lean-data-isolation`: Requirement "Optional PostgreSQL RLS policy for defense-in-depth" reworded so the requirement statement contains SHALL/MUST (no behavioural change).
- `stytch-authorization`: ADDED 4 requirements (Role normalization, RBAC API endpoint authentication, RBACService implementation backed by Stytch policy, DTOs retained as API contract) with scenarios, restoring content lost from the living spec. The abandoned password-auth requirement stays excluded.
- `spec-validation`: Modified — the CI gate requirement gains strict-mode enforcement; the gate must fail on both structural errors and brief-Purpose warnings.
- 33 further capabilities: `## Purpose` expanded beyond 50 chars (framing-only, no requirement changes, no delta specs).

## Impact

- `openspec/specs/<capability>/spec.md` — 37 files touched: 1 rewritten (frontend-api-client), 1 requirement reworded (lean-data-isolation), 1 capability +4 requirements (stytch-authorization), 33 Purpose expansions, 2 editorial trims.
- `.github/workflows/ci.yml` — `spec-validation` job command changes (dual validation).
- No application code, database schema, API surface, or Stytch tenant policy changes. No auth flow or persistence modification — this change is spec-tree governance only.
- Risks: minimal; restore is additive and verified against archived deltas and code. Coordination: `stytch-authorization`, `whatsapp-*` and other active changes (`add-whatsapp-embedded-signup`, `add-mercadopago-billing`, `add-siigo-invoicing`) may edit specs being normalized — their deltas target requirement content, not `## Purpose` framing, so merge conflicts are unlikely; if they occur, resolve in favor of this change's framing plus their requirement deltas.
- Rollback: Git-only. All edits are markdown framing/content restore; `git revert` of the change's commits restores both the spec tree and CI config. No Stytch tenant state is involved.

## Non-Goals

- No local credential storage is introduced or modified; this change does not touch the Stytch auth runtime, session handling, or the DB/identity boundary.
- No re-architecture of the OpenSpec fold/archive process; the hardened CI gate is the regression guard.
- No content rewrite of requirements beyond restoration of archived contract (stytch-authorization) and the one validator-error reword (lean-data-isolation).
- The stale `normalize-spec-format` change (never started, premise partially unverified — its `crm-core-data` drift claim was false) is NOT adopted; this change supersedes it. It was removed from the working tree during this change's creation.
- No changes to spec content in the 33 Purpose-expanded capabilities.
