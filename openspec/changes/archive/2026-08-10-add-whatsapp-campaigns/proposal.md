## Why

Competitors (Wati, Zoko, ManyChat) win LatAm on proactive messaging — promos, recordatorios, novedades. This app only supports reactive outbound (replies, agent drafts). Broadcast/marketing campaigns are the biggest product gap and the clearest repeat-billing hook. The cheapest first unlock is the audience layer: saved contact segments with snapshot semantics, plus an AI audience builder.

## What Changes

- Add `segments` entity: org-scoped, saved contact filter specs (JSONB), validated against a whitelist of fields/ops — no raw SQL.
- Add segment evaluation that reuses the existing filtered contact query plus `crm.entity_tags` join, with mandatory hard gates appended after user filters: `consent_status = 'granted'` and valid E.164 phone.
- Add `campaigns` (draft state machine) and `campaign_recipients` (audience snapshot) tables. Snapshot is taken at launch: dedup, per-recipient status, idempotent insert — bill of materials for the future scheduler/billing.
- Add AI audience builder: natural language → LLM → filter spec JSON, validated against the same whitelist, shown to the user as a preview count before save. Metered through the existing cognitive ledger, org-tagged, PII-safe (no contact PII in prompt).
- New endpoints under `/crm/campanas` (segments + campaigns), RBAC: `org:manage` for writes, `org:view` for reads, behind existing auth + org_context + subscription middleware.
- Thin FE: segment manager + preview count (FE scope minimal; scheduler UI is future).

## Capabilities

### New Capabilities
- `whatsapp-campaigns`: segment CRUD + whitelisted filter-spec eval, hard-gate invariants (consent, valid phone), campaign draft lifecycle, audience snapshot idempotency, AI audience builder contract (metered, masked, human-approve-before-save).

### Modified Capabilities
- None. Existing specs' requirements are unchanged (segment eval adds a query, does not alter `contact-management` list endpoint contracts; metering reuses `ai-usage-metering` pattern without changing it).

## Impact

- **DB**: new tables `crm.segments`, `crm.campaigns`, `crm.campaign_recipients`; new SQLC queries for eval/count/snapshot; `make sqlc` regenerates models. Backend only — no schema changes to existing tables.
- **BE**: new module under `internal/modules/` following the crm/whatsapp/agent module pattern (domain → app → infra, dig DI). Segment eval service reuses `ListContactsByOrganizationFiltered`; AI builder reuses the cognitive metered LLM client (`llmdomain.LLMClient` with `WithOrgID`/`WithPiiFacts`, same pattern as `agent_service.go:148`).
- **Auth/RBAC (Stytch)**: no Stytch API contract changes; new routes reuse existing `org:manage` / `org:view` permission checks enforced by the existing org_context middleware. Session validation, webhook verification, and the JWKS cache path are untouched.
- **FE**: segment manager page + preview count in `next_b2b_starter/`.
- **Out of scope (future changes)**: template/HSM management, scheduler/sending, opt-out keyword loop, campaign credits billing, delivery status backfill.
- **Rollback**: feature-gate the module in the module registry (orgs unaffected until enabled); Git state rollback = revert the change; drop the three new tables via down migration. No Stytch tenant policy changes are made, so no Stytch-side rollback is required.

## Non-Goals

- No sending engine, scheduler, or template rendering (future changes).
- No local credential storage: no new secrets, tokens, or passwords in PostgreSQL; campaign data stores only org-scoped CRM foreign keys, never credentials.
- No changes to identity/session authority — Stytch B2B remains sole authority for members, orgs, sessions, and RBAC; this change adds no local auth state.
