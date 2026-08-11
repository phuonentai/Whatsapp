## Context

App has reactive WhatsApp outbound only (CRM replies, agent drafts). Broadcast/marketing is the biggest product gap and the repeat-billing hook. This change delivers the audience layer: saved contact segments, snapshot semantics, and an AI audience builder. It deliberately stops short of sending — templates, scheduler, and credits are future changes that will consume the artifacts built here.

Current machinery being reused:
- `ListContactsByOrganizationFiltered` (`internal/db/postgres/sqlc/query/crm_extended.sql:5`) — org-scoped filters: `source`, `lead_status`, `company_id`, `assigned_to`.
- `crm.tags` + `crm.entity_tags(entity_type, entity_id)` — indexed M2M tags (migration 000013).
- `crm.contacts.consent_status` (`none|requested|granted|withdrawn`, migration 000019) — Ley 1581 consent machine.
- Metered LLM client (`modules/cognitive/infra/ai/assistant_provider.go`, token ledger) with `llmdomain.WithOrgID` / `WithPiiFacts` pattern (see `agent_service.go:148-169`).
- Module pattern: `internal/modules/<name>/` with `domain/ app/ infra/ handler.go module.go provider.go routes.go`, dig DI.

## Goals / Non-Goals

**Goals:**
- Segment CRUD with whitelisted, validated filter specs (no raw SQL anywhere).
- One evaluation pipeline for preview count AND audience snapshot — same SQLC queries, same hard gates.
- Hard compliance gates (consent = `granted`, valid E.164 phone) enforced outside user control, after user filters.
- AI audience builder: NL → filter spec → validated → preview count → human saves. Metered, PII-safe.
- Campaign draft lifecycle (`draft → ready`) with idempotent recipient snapshot, ready for a future scheduler.

**Non-Goals:**
- No sending, scheduler, template/HSM, opt-out keyword loop, credits billing, delivery backfill (future changes).
- No changes to existing SQLC query contracts (`ListContactsByOrganizationFiltered` stays untouched; new queries are added).
- No Stytch API/RBAC policy changes.

## Decisions

### D1. Segment = whitelisted JSONB filter spec, AND semantics
`filter_spec` is a JSON array `[{"field", "op", "value"}]`, AND-combined. Allowed fields/ops:
| field | ops | notes |
|---|---|---|
| `source` | eq | contact source |
| `lead_status` | eq | `nuevo|contactado|calificado|descalificado|cliente` |
| `company_id` | eq | FK to `crm.companies` |
| `assigned_to` | eq | FK to `organizations.accounts` |
| `tag_ids` | any | any-of semantics, join `entity_tags` |
| `recency_days` | lte | `last_message_at >= now() - N days` |
| `search` | contains | ILIKE over name/email/phone/documento |

Validation rejects unknown fields/ops/empty values with HTTP 400. *Alternative rejected:* store generated SQL — injection risk, spec drift, no validation path; store raw user query objects — unbounded.

### D2. Hard gates are layered after user filters, not part of the spec
Eval always appends: `consent_status = 'granted'` AND valid E.164 `phone_number`. User cannot filter these out of `filter_spec`. Gates are non-negotiable in `SegmentEvaluator` (domain service), so every consumer (preview, snapshot, future scheduler) gets them for free. *Alternative rejected:* gates as spec fields — user could remove them and blow compliance (Ley 1581).

### D3. Audience snapshot at launch into `campaign_recipients`
Launch = evaluate segment → dedup → insert one row per contact (`pending`), `campaign.status` `draft → ready`, `recipient_count` set. Snapshot is the bill of materials: exact quota math, audit trail, per-recipient send results, idempotent retry. *Alternative rejected:* live eval at send time — audience drifts mid-send, no audit, no billing basis.

### D4. One segment per campaign (v1)
`campaigns.segment_id` single FK. *Alternative rejected:* union of segments — dedup edge cases, no SMB demand yet. Richer audiences = future `filter_spec` growth, not campaign composition.

### D5. AI audience builder: NL → LLM → JSON → whitelist validation → preview → human save
Endpoint `POST /crm/campanas/segments/ai-build`: user types "clientes mayoristas de Bogotá que escribieron este mes" → metered LLM call (org-tagged, `WithOrgID`) → strict JSON filter spec → same D1 validator → returns candidate spec + preview count. User edits or saves; nothing persists without explicit save. Prompt contains only the NL text + field/op dictionary — zero contact PII, masking decorator still applies via the shared client seam. *Alternatives rejected:* LLM writing SQL (injection + unvalidatable); deterministic parser (brittle Spanish); auto-save (compliance risk).

### D6. New eval queries; existing queries untouched
New SQLC queries: `ListSegmentContacts` (filters + tags join + recency + gates), `CountSegmentContacts`, `SnapshotCampaignRecipients` (INSERT ... SELECT ... ON CONFLICT (campaign_id, contact_id) DO NOTHING). Generated models via `make sqlc`. *Alternative rejected:* parametrizing `ListContactsByOrganizationFiltered` — changes existing contract, mixes concerns.

### D7. New `internal/modules/campaigns` module
`domain/` (Segment, Campaign, evaluator interface), `app/` (services: segment service, campaign service, AI builder), `infra/` (repos + eval via SQLC + LLM adapter implementing domain interface), `handler.go`, `routes.go`, `module.go`, `provider.go` — mirrors `crm`/`whatsapp` modules, registered in module registry. *Alternative rejected:* stuffing into `whatsapp` module — campaigns cross-cut CRM contacts; separate module keeps clean architecture and feature gating.

### D8. RBAC: `org:manage` writes, `org:view` reads, zero Stytch changes
Routes behind existing `auth` + `org_context` + `subscription` middleware, permission checks reuse existing Stytch role → permission mapping. No new roles, no policy changes.

### D9. Campaign launch is guarded + idempotent
`UPDATE campaigns SET status='ready', launched_at=NOW() WHERE id=$1 AND organization_id=$2 AND status='draft'` — single-row guard prevents double snapshot. Recipients insert is idempotent via unique `(campaign_id, contact_id)`.

## Risks / Trade-offs

- [Stale snapshot: contact deleted/changed after launch] → `campaign_recipients.contact_id` FK `ON DELETE CASCADE`; future scheduler marks missing contacts `skipped`. Preview always recomputes live, so draft-time counts are fresh.
- [Eval performance at scale] → existing indexes cover filters (`idx_contacts_lead_status`, company, assigned, `idx_entity_tags_*`); recency uses existing `last_message_at`; SMB scale (10⁴–10⁵ contacts) fine with index scans. Count query separate from list to avoid paging overhead.
- [LLM hallucinates filter values (bad field name / bad tag id)] → D1 validator rejects; v1 returns HTTP 400 with Spanish error, no auto-retry loop. Tag ids resolved against org's `crm.tags` before eval.
- [LLM prompt injection via NL input] → LLM outputs JSON only, validated against whitelist; no SQL interpolation; no contact PII in prompt; metered call means abuse costs tokens (org-visible via ai ledger).
- [Consent gate hides contacts from preview → surprise small audience] → preview shows gate exclusions explicitly ("X contactos excluidos por consentimiento").
- [Migration conflict with in-flight changes] → next migration number is 000029 (current max 000028); co-authors must coordinate numbering.

## Migration Plan

1. Migration `000029_create_campaign_segments.up.sql`: tables `crm.segments`, `crm.campaigns`, `crm.campaign_recipients` (+ unique indexes: segments `(organization_id, id)`, recipients `(campaign_id, contact_id)`; FKs org-scoped per 000016 pattern). Down migration drops all three.
2. SQLC: add `query/campaigns.sql`, run `make sqlc`, commit generated models.
3. Module `campaigns` registered + feature-gated in module registry (orgs opt-in).
4. Routes mounted behind org_context middleware.
5. FE: segment manager + preview + campaign draft page (thin).
6. Rollback: disable module gate → down migration → revert change in Git. No Stytch-side rollback needed (no policy changes).

## Open Questions

- Recency + search in v1 filter spec: included (D1) — confirm no SMB need for date-range operator (`gte`) in v1.
- `campaign_recipients` retention: keep rows forever for audit, or prune? v1: keep.
- FE i18n: Spanish-only UI labels consistent with existing CRM pages (yes).
