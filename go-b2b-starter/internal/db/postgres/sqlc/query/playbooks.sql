-- name: ListActivePlaybooks :many
SELECT * FROM modules.playbooks
WHERE is_active = true
ORDER BY key;

-- name: GetPlaybookByKey :one
SELECT * FROM modules.playbooks
WHERE key = $1
LIMIT 1;

-- name: UpsertOrgPlaybook :one
INSERT INTO modules.organization_playbooks (organization_id, playbook_key, seeded_pipeline_id)
VALUES ($1, $2, $3)
ON CONFLICT (organization_id, playbook_key)
DO UPDATE SET applied_at = CURRENT_TIMESTAMP
RETURNING *;

-- name: ListOrgPlaybooks :many
SELECT * FROM modules.organization_playbooks
WHERE organization_id = $1
ORDER BY playbook_key;

-- name: GetOrgPlaybook :one
SELECT * FROM modules.organization_playbooks
WHERE organization_id = $1 AND playbook_key = $2
LIMIT 1;

-- name: DeleteOrgPlaybook :exec
DELETE FROM modules.organization_playbooks
WHERE organization_id = $1 AND playbook_key = $2;
