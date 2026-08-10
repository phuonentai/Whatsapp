BEGIN;

-- Add organization_id to file_assets for tenant isolation
ALTER TABLE file_manager.file_assets
ADD COLUMN organization_id INTEGER NOT NULL DEFAULT 0 REFERENCES organizations.organizations(id) ON DELETE CASCADE;

-- Update existing indexes to include organization_id
DROP INDEX IF EXISTS file_manager.idx_file_assets_entity;
CREATE INDEX idx_file_assets_entity ON file_manager.file_assets(organization_id, entity_type, entity_id);
CREATE INDEX idx_file_assets_org ON file_manager.file_assets(organization_id);

COMMIT;
