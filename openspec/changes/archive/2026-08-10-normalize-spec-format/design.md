## Context

`openspec validate --specs` reports 38 of 52 living specs as invalid: they are framed as change deltas (`## ADDED Requirements` / `## MODIFIED Requirements`) rather than living specs (`## Purpose` + `## Requirements`). This happened because archived change deltas were copied into `openspec/specs/` verbatim during the archive fold instead of being normalized. A verification sweep against archived deltas shows the content is sound in 35 specs (framing-only defect) and drifted in 3 specs (missing requirements/scenarios). No CI gate currently runs spec validation, so the tree validated green historically but nothing prevents regression.

## Goals / Non-Goals

**Goals:**

- Make `openspec validate --specs` pass (52/52) by normalizing all 38 invalid living specs to living-spec framing.
- Restore archived-contract content lost during fold in exactly 3 specs: `stytch-authorization`, `crm-core-data`, `whatsapp-webhook-ingress`.
- Add a CI gate (`spec-validation` job) that runs `openspec validate --specs` to prevent silent regression.

**Non-Goals:**

- No change to application code, DB schema, API, or auth runtime (incl. no credential storage changes).
- No re-engineering of the OpenSpec fold process; the CI gate is the chosen regression guard for now.
- No speculative content rewriting beyond restoring archived-delta contracts.

## Decisions

**D1: Normalize in place; do not regenerate living specs from deltas.**
Rationale: `activity-timeline` diff showed the living spec is richer than any single archived delta (requirements merged across multiple changes). Regeneration would lose hand-merged content. The living spec is the source of truth; the fix is framing + additive restoration only.
Alternative considered: re-run `openspec archive` on archived changes — rejected, would re-merge stale deltas and re-enter the same verbatim-copy bug.

**D2: Convert `## ADDED/MODIFIED Requirements` to a single `## Requirements`; prepend `## Purpose` from content.**
Rationale: validator requires exact headers `## Purpose` and `## Requirements`. Six specs have two concatenated section blocks (`crm-core-data`, `crm-frontend`, `stytch-authorization`, `whatsapp-config-api`, `whatsapp-config-frontend`, `whatsapp-webhook-ingress`) — those are merged into one `## Requirements` section, preserving all content.
Purposes are derived per-spec from the requirement set; `module-registry` and `vertical-playbooks` already have a `## Purpose` and keep theirs.

**D3: Restore missing content in 3 specs from archived deltas; exclude the abandoned password-auth requirement.**
Rationale: verification showed `stytch-authorization` is missing 5 requirements (14 scenarios), `crm-core-data` 3 requirements (9 scenarios), `whatsapp-webhook-ingress` 3 scenarios — all present in archived deltas (`2026-07-28-stytch-debt-cleanup`, `2026-07-27-setup-whatsapp-ingress-and-crm-core`, ingress deltas). The password-auth requirement is excluded because `2026-08-08-abandon-password-auth` explicitly rejected it and never merged its deltas; restoring it would contradict the living `auth-passwordless-e2e` spec.
Reparent the orphaned `stytch-go v16→v18` body (line 1 of `stytch-authorization`) under a proper `### Requirement:` header.

**D4: Add CI gate as a new `spec-validation` job in `.github/workflows/ci.yml`.**
Rationale: the workflow has backend/frontend/backup-drill/e2e jobs but no OpenSpec validation. A dedicated job keeps the tree's source-of-truth invariant checkable on every push/PR without coupling to app builds.
Alternative considered: fold into the `frontend` job — rejected, spec validity is repo-wide, not frontend-scoped.

## Risks / Trade-offs

- [Content drift beyond verified scope] → Verification sweep compared requirement + scenario titles between living specs and the union of all archived deltas (`extra=0` across all 38); restoration is additive, never destructive.
- [`whatsapp-webhook-ingress` collision with active change `add-whatsapp-embedded-signup`] → Apply the framing conversion and scenario restoration first, then let the active change's archive step re-merge its deltas on top; keep its delta spec unchanged here.
- [Hand-edited Purpose text drifts from content] → Purpose is a 1-line derived summary, low risk; validator only checks presence, not phrasing.
- [CI gate blocks unrelated PRs until specs fixed] → Gate added in same change that fixes all 38 specs, so tree is green before the gate lands.
