-- Rollback: drop the visibility column (drops the CHECK constraint with it).

ALTER TABLE documents.documents
    DROP COLUMN IF EXISTS visibility;
