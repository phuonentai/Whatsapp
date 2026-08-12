STATUS: APPROVED
MARKET: PASS

# Council Verdict — knowledge-doc-permissions

**Change:** Knowledge Base v2 — permisos por documento (ACL) + rail de dos modos
**Scope:** Market-in-scope (`requires_market_read: true`; AI/RAG retrieval surface + Ley 1581 compliance exposure)
**Review basis:** design.md, proposal.md, tasks.md, delta specs (cognitive-streaming, knowledge-base-ui, knowledge-doc-permissions), living specs (cognitive-streaming, data-transfer, whatsapp-compliance, paywall, ai-usage-metering), code evidence (`cognitive.sql`, `rag_service.go`, `embedding_service.go`, migrations 000008/000009, `documents/routes.go`).
**Prior verdict / revision:** none — first review.

## Verdict

**APPROVED.** No REJECT-level defects. The central security decision — ACL enforced inside the shared SQL retrieval path, never UI-only — is the correct architecture and is grounded in the actual code (`SearchSimilarDocuments` in `cognitive.sql` today has no visibility filter; `documents.documents` has no `visibility` column; migration 000009 gives `document_embeddings` `ON DELETE CASCADE` for the deletion-cascade requirement). Design and delta specs are mutually coherent. Both mandatory market sections are present, and all accepted residuals carry a named owner and trigger. Findings below are implementation obligations, not blockers.

## Per-Persona Findings

### 1. Staff Security Engineer

- **LOW — File-asset raw read path not enumerated (residual).** `documents.documents.file_asset_id → file_manager.file_assets`; no user-facing file-serve route was found in `documents/routes.go` (only upload/list/delete), so no live bypass is identified — but task 2.1's consumer-grep SHALL also enumerate every read path for document *content* (file asset fetch, `extracted_text` exposure, embedding-by-document reads) to guarantee an `admin_only` doc cannot be fetched by its file asset ID. Add this to the spike scope.
- **INFO — Role threading into RAG is a genuine contract ripple.** `EmbeddingService.SearchSimilarDocuments(ctx, orgID, text, limit)` takes no role today; the design's open question is real and correctly spiked (task 2.1, with documented fallback: pass from `org_context` middleware). The interface change propagates through `interface.go`, repo, and `rag_service.go` — scoped in tasks.
- **PASS — No title/404 leak:** restricted doc behaves as nonexistent; scenario-tested in the delta spec.
- **PASS — Admin-only management is a tightening:** current routes gate on `resource:create/view/delete`; moving upload/delete/rename/visibility to `org:manage` reduces the management surface (compliance-positive). Behavior change for members holding `resource:*` is captured as accepted residual R2.

### 2. Staff DBA

- **PASS — Migration is expand-safe:** single additive column, `NOT NULL DEFAULT 'workspace'` (metadata-only on PG ≥11, no table rewrite), CHECK constraint, backfill in the same migration; rollback = drop column. No expand-contract violation.
- **INFO — ACL JOIN cost acceptable:** filter JOINs `document_embeddings` to `documents` on PK; `idx_doc_embeddings_organization` already exists. Small tables; no index addition strictly required (consider partial index on `visibility` only if doc volume grows).
- **INFO — Retry re-embedding should be transactional:** `DeleteDocumentEmbeddings` + re-insert against `UNIQUE(document_id, chunk_index)` should run in one transaction to avoid transient partial state and unique violations on concurrent retry.

### 3. SRE

- **PASS — No new external dependencies, no distributed-lock surface, no new LLM invocations; rollback is a single drop-column + git revert (no Stytch state).**
- **LOW — Observability gap (residual):** add a metric/log for RAG retrievals that return zero chunks *because of* the ACL filter (vs. genuine no-match), so the compliance filter is auditable and false "no encontré" rates are distinguishable. Not required for approval; record in tasks.

### 4. Staff Product/GTM

- **PASS — Unit economics coherent:** no delta in per-query LLM cost (same retrieval, filtered); `visibility` column adds no embedding or metering cost; pricing/plan surfaces untouched (`ai-usage-metering`, `paywall` unchanged — verified).
- **PASS — Compliance as market value:** `admin_only` is the documented answer for docs containing contact data under Ley 1581 consent; export traceability (title/status/visibility) is a defensible Colombian-market differentiator.
- **PASS — Activation surface:** guided empty state + onboarding checklist link; honest "No encontré esto en tus documentos" anti-hallucination guard differentiates vs. "answer-anything" assistants (residual R3 with owner/trigger).

### 5. Colombia IT & Market

- **PASS — Ley 1581 / Habeas Data:** the ACL is a consent-aligned data-minimization control; export addition builds on the existing CSV/Habeas Data export contract (`data-transfer` spec: withdrawn-consent PII masking) without breaking it. Deletion cascades embeddings (verified `ON DELETE CASCADE`) so the index cannot retain contact data after doc deletion; citation fallback "Documento no disponible" closes the stale-citation leak.
- **INFO — R1 (index as compliance surface) is the correct framing:** default `workspace` + admin-only management + export traceability is a defensible posture; keep legal review of defaults as the recorded trigger.
- **N/A — DIAN/invoicing:** out of scope for this change.

## Market Read

The change is in-scope because it alters the *AI data surface* (RAG retrieval now visibility-filtered) and the *compliance surface* (Ley 1581 export traceability of indexed documents), not because it changes pricing. Cost math is neutral: same LLM invocation and credit metering, `visibility` adds no embedding cost; no plan/price/paywall delta (verified against `ai-usage-metering` and `paywall` specs). The market upside is compliance-driven: a documented, server-enforced mechanism to keep contact PII out of the RAG index unless explicitly scoped to admins is a credible differentiator for Colombian SMBs facing Habeas Data audits, and the export gains per-document traceability. Risks are recorded as accepted residuals with owners and triggers: R1 (index-as-compliance-surface; owner product/compliance; trigger exposure incident or audit), R2 (member expectation of self-management; owner product; trigger client feedback), R3 (honesty-vs-answer-everything positioning; owner product/GTM; trigger qualitative feedback). No unverified external fact is asserted as a premise — the market claims are qualitative, hedged, and do not drive the architecture. Residuals are accepted and tracked: **MARKET: PASS**.

## Implementation Obligations (from findings above)

1. Task 2.1 spike SHALL enumerate all document-*content* read paths (file asset fetch, `extracted_text`, embedding-by-document) in addition to `SearchSimilarDocuments` consumers.
2. Retry re-embedding SHALL be transactional.
3. Add a distinguishable metric for ACL-filtered zero-chunk retrievals.
4. Record the Stytch policy dependency: the ACL uses the effective `org:manage` role; confirm availability via the `org_context` middleware in the spike (as designed).
