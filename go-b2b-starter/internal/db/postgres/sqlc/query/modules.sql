-- name: ListActiveModules :many
SELECT * FROM modules.modules
WHERE is_active = true
ORDER BY key;

-- name: GetModuleByKey :one
SELECT * FROM modules.modules
WHERE key = $1
LIMIT 1;

-- name: ListModulesByOrg :many
SELECT m.* FROM modules.modules m
JOIN modules.organization_modules om ON om.module_key = m.key
WHERE om.organization_id = $1 AND m.is_active = true
ORDER BY m.key;

-- name: ListOrgModules :many
SELECT * FROM modules.organization_modules
WHERE organization_id = $1
ORDER BY module_key;

-- name: GetOrgModule :one
SELECT * FROM modules.organization_modules
WHERE organization_id = $1 AND module_key = $2
LIMIT 1;

-- name: UpsertOrgModule :one
INSERT INTO modules.organization_modules (organization_id, module_key, config)
VALUES ($1, $2, $3)
ON CONFLICT (organization_id, module_key)
DO UPDATE SET config = EXCLUDED.config, enabled_at = CURRENT_TIMESTAMP
RETURNING *;

-- name: DeleteOrgModule :exec
DELETE FROM modules.organization_modules
WHERE organization_id = $1 AND module_key = $2;
