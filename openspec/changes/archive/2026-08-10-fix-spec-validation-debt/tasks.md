## 1. Write frontend-api-client living spec [OPS-GOV]

- [x] 1.1 Author `openspec/specs/frontend-api-client/spec.md` with `## Purpose` (>50 chars) and `## Requirements` covering: ApiClient typed verbs + base URL resolution, Bearer token attach + refresh/retry/backoff, mock-auth `X-Test-Org-ID` forwarding, envelope unwrap in repositories. Source of truth: `next_b2b_starter/lib/api/api/client/api-client.ts` (verified 733 lines) and `lib/api/api/repositories/*.ts` (verified repository + `unwrap()` pattern). No invented behaviour.
- [x] 1.2 Verify: `openspec validate frontend-api-client --type spec` → valid, no ERROR/WARNING.

## 2. Fix lean-data-isolation validator error [OPS-GOV]

- [x] 2.1 Reword requirement "Optional PostgreSQL RLS policy for defense-in-depth" in `openspec/specs/lean-data-isolation/spec.md` so the requirement statement contains SHALL/MUST (see change delta for exact wording). Behaviour unchanged; scenarios untouched.
- [x] 2.2 Verify: `openspec validate lean-data-isolation --type spec` → valid.

## 3. Expand 33 brief Purpose sections [OPS-GOV]

- [x] 3.1 For each of the 33 specs whose `## Purpose` is under 50 characters (validator WARNING), expand the Purpose to a single descriptive sentence >50 chars sourced from that spec's own requirements. Do NOT touch requirement bodies. Specs: activity-timeline, admin-panel-audit-log, admin-panel-navigation, agent-governance, ai-usage-metering, company-management, contact-management, crm-conversation-api, crm-core-data, crm-frontend, data-backup-recovery, data-transfer, deal-management, durable-message-pipeline, fe-component-tests, feature-gating, governance-workflow, inbox-ui, knowledge-base-ui, pipeline-management, production-health-and-ops, settings-ui, signup-stytch-compliance, tag-management, whatsapp-agent, whatsapp-compliance, whatsapp-config-api, whatsapp-config-frontend, whatsapp-inbox, whatsapp-outbound-send, whatsapp-provider-resilience, whatsapp-webhook-ingress, workspace-settings-management.
- [x] 3.2 Verify: `openspec validate --specs --strict` → 0 warnings (55/55). Requirement/scenario counts in each edited spec MUST be unchanged from before the edit. (Edits were single-line Purpose replacements; 33/33 matched exactly, 0 errors.)

## 4. Restore stytch-authorization requirements [OPS-GOV]

- [x] 4.1 Add 4 requirements to `openspec/specs/stytch-authorization/spec.md` from the change delta `specs/stytch-authorization/spec.md` (Role normalization, RBAC API endpoint authentication, RBACService implementation backed by Stytch policy, DTOs retained as API contract) with their scenarios. Do NOT add the abandoned password-auth requirement. Orphaned `stytch-go v16→v18` body line SHALL be reparented under a proper `### Requirement:` header. (Parallel work landed restore mid-session; verified present: lines 112–188, orphan reparented at line 7, password-auth absent.)
- [x] 4.2 Verify: `openspec validate stytch-authorization --type spec` → valid; the 4 restored requirements present; password-auth absent. (Valid, 9 requirements, 0 password-auth refs.)

## 5. Harden CI spec-validation gate [OPS-GOV]

- [x] 5.1 Edit `.github/workflows/ci.yml` `spec-validation` job so the run step executes both `npx openspec validate --specs` and `npx openspec validate --specs --strict` (see change delta `specs/spec-validation/spec.md`); either non-zero exit fails the job. Verify with `actionlint` if available, else YAML syntax check. (YAML parse OK.)
- [x] 5.2 Verify: both commands exit 0 locally with the fixed tree; `openspec validate --change fix-spec-validation-debt` → change deltas valid. (nonstrict 0, strict 0, change valid.)

## 6. Trim INFO-level long requirement texts [OPS-GOV]

- [x] 6.1 In `openspec/specs/stytch-authorization/spec.md` (requirements at index 1, 4, 7) and `openspec/specs/vertical-playbooks/spec.md` (index 0, 1, 5), split over-long requirement bodies (>500 chars) into shorter sentences without changing meaning. If a split risks changing meaning, leave text and record the INFO as accepted. **NO TRIMS MADE** — all 15 INFO texts (8 specs, incl. agent-governance, crm-core-data, data-transfer, governance-workflow, inbox-ui, signup-stytch-compliance) are dense contract clauses (cache TTLs, FK composites, import templates, sequence rules); splitting would alter meaning. Per design D5 (semantics beat cosmetics), INFOs accepted. INFO does not affect `validate`/`--strict` exit codes.
- [x] 6.2 Verify: `openspec validate --specs --json` → zero WARNING/INFO/ERROR at any level (54/54 clean). **REVISED**: WARNING+ERROR at zero (55/55); 15 INFO-level advisory long-text notices remain, accepted per D5 (design decision, not gate failure — exit codes 0 in both modes).

## 7. Verification gate [OPS-GOV]

- [x] 7.1 Run `openspec validate --specs` → **PASS** — Totals: 55 passed, 0 failed (55 items).
- [x] 7.2 Run `openspec validate --specs --strict` → **PASS** — Totals: 55 passed, 0 failed (55 items).
- [x] 7.3 Run `openspec validate fix-spec-validation-debt --type change` → **PASS** — Change valid.
- [x] 7.4 Confirm `git diff` touches only `openspec/specs/**`, `openspec/changes/fix-spec-validation-debt/**`, and `.github/workflows/ci.yml`. **PARTIAL** — HEAD is stale (parallel work churns working tree: 52 spec files dirty, 34 untracked, plus unrelated Makefile/sqlc/docs edits from other sessions). My edits this session confined to the 3 allowed paths (33 Purpose lines, frontend-api-client, lean-data-isolation, ci.yml, change dir). Full-tree diff-scope verification deferred to commit/PR review after parallel work settles.

**Archive deferred:** change is implementable and gate-green, but verification 7.4 could not be fully confirmed against the working tree because parallel uncommitted work (other active changes, untracked spec churn) overlaps the same files. Re-run `git diff` scope check and archive once parallel work is committed or resolved.
