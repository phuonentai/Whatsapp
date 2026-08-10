## Why

`openspec validate --specs` fails on 38 of 52 living specs (14 pass, 38 fail). The living specs were populated from archived change deltas by copying delta files verbatim instead of folding them into living-spec format (`## Purpose` + `## Requirements`). As a result the OpenSpec tree — the repo's declared behavioural source of truth — fails its own validation, and no CI gate catches it. This undermines governance gate 5 (verification) and hides real content drift in three specs.

## What Changes

- **Normalize living-spec framing in all 38 failing specs** (OPS-GOV, `openspec/specs/`):
  - Replace `## ADDED Requirements` / `## MODIFIED Requirements` section markers with a single `## Requirements` header.
  - Prepend a brief `## Purpose` section where missing (35 of 38 lack one; `module-registry` and `vertical-playbooks` already have it).
  - Merge concatenated multi-section files (6 specs have two `## ADDED`/`## MODIFIED` blocks) into one coherent `## Requirements` section.
  - Reparent orphaned requirement bodies: `stytch-authorization/spec.md` line 1 contains a bare body ("stytch-go from v16 to v18") whose `### Requirement:` header is missing.
- **Restore drifted content in 2 specs from archived deltas** (delta specs in this change):
  - `stytch-authorization`: add back 4 requirements + scenarios present in the archived `2026-07-28-stytch-debt-cleanup` delta but missing from the living spec (verified present in code: `stytch_rbac_service.go`, `rbac_policy.go`, RBAC routes). Explicitly **exclude** the "Password authentication resolves organization before Stytch call" requirement — that feature was abandoned (`2026-08-08-abandon-password-auth`) and its deltas were deliberately never merged.
  - `crm-core-data`: add back 3 requirements ("Conversation entity with status tracking", "Message entity with WhatsApp-specific fields", "Event subscriber processes MessageReceived events asynchronously") from the archived `2026-07-27-setup-whatsapp-ingress-and-crm-core` delta. Verified genuinely missing — no newer delta supersedes them.
- **Add a CI gate** in `.github/workflows/ci.yml` (new `spec-validation` job) running `openspec validate --specs` so the tree cannot regress silently.

## Capabilities

### New Capabilities

- `spec-validation`: CI gate that runs `openspec validate --specs` and fails the build on any invalid or missing-purpose spec. This capability owns the repeatable validation step, not the content of any given capability.

### Modified Capabilities

- `stytch-authorization`: Restore 4 requirements (Role normalization, RBAC API endpoint authentication, RBACService implementation backed by Stytch policy, DTOs retained as API contract) and their scenarios; keep the abandoned password-auth requirement excluded. Framing normalized to living-spec format.
- `crm-core-data`: Restore 3 requirements (Conversation entity with status tracking, Message entity with WhatsApp-specific fields, Event subscriber processes MessageReceived events asynchronously). Framing normalized.
- `whatsapp-webhook-ingress`: framing-only normalization. Initial drift analysis flagged 3 missing scenarios, but they are **false positives** — the older `add-e2e-edge-coverage` scenarios were superseded by `harden-business-continuity` ("enqueued atomically"), which the living spec already reflects.
- `activity-timeline` and 34 further specs: framing-only normalization (no requirement/scenario content change).

## Impact

- `openspec/specs/**` — 38 living spec files reformatted (headers, Purpose); content restored in 3 of them.
- `.github/workflows/ci.yml` — new `spec-validation` job.
- No application code, database schema, or API surface changes.
- No behaviour change to any shipped feature; the restored requirements reflect contracts already implemented and archived before this change.
- Risks: minimal — content restoration is additive only (verified: no spec contains content absent from archived deltas, `extra=0` across all 38). Coordination needed for `whatsapp-webhook-ingress` (active change `add-whatsapp-embedded-signup` edits the same capability).

## Non-Goals

- No local credential storage is introduced; this change does not touch the auth runtime, Stytch API contracts, or the DB/identity boundary.
- No re-architecture of the OpenSpec fold process itself; the CI gate is the chosen regression guard.
- No rewriting of spec content beyond restoration of archived-contract requirements.
