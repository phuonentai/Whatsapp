BEGIN;

DROP INDEX IF EXISTS file_manager.idx_file_assets_org;
DROP INDEX IF EXISTS file_manager.idx_file_assets_entity;

ALTER TABLE file_manager.file_assets DROP COLUMN organization_id;

CREATE INDEX idx_file_assets_entity ON file_manager.file_assets(entity_type, entity_id);

COMMIT;
