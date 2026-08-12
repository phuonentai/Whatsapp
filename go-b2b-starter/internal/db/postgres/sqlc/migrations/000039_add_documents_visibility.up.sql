-- Add per-document visibility (ACL for RAG retrieval).
-- Default 'workspace' backfills existing rows in the same statement (metadata-only
-- on PG >= 11: NOT NULL + constant DEFAULT does not rewrite the table).

ALTER TABLE documents.documents
    ADD COLUMN visibility VARCHAR(20) NOT NULL DEFAULT 'workspace';

ALTER TABLE documents.documents
    ADD CONSTRAINT documents_visibility_check
    CHECK (visibility IN ('workspace', 'admin_only'));

COMMENT ON COLUMN documents.documents.visibility IS
    'Document visibility ACL: workspace (all org members) | admin_only (org:manage only).';
