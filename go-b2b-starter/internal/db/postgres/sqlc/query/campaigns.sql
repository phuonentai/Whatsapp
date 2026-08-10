-- Campaign audience layer: segments, campaigns, recipient snapshot.

-- ============================================================
-- Segment evaluation (filters + mandatory hard gates)
-- All queries apply the hard gates unconditionally:
--   consent_status = 'granted' AND valid E.164 phone
-- They are NOT part of filter_spec and cannot be filtered out.
-- ============================================================

-- name: ListSegmentContacts :many
SELECT c.*
FROM crm.contacts c
WHERE c.organization_id = $1
  AND ($2::text IS NULL OR $2 = '' OR c.source = $2)
  AND ($3::text IS NULL OR $3 = '' OR c.lead_status = $3)
  AND ($4::int = 0 OR c.company_id = $4)
  AND ($5::int = 0 OR c.assigned_to = $5)
  AND ($6::int[] IS NULL OR EXISTS (
      SELECT 1
      FROM crm.entity_tags et
      JOIN crm.tags t ON t.id = et.tag_id AND t.organization_id = c.organization_id
      WHERE et.entity_type = 'contact' AND et.entity_id = c.id
        AND et.tag_id = ANY($6)))
  AND ($7::int = 0 OR c.last_message_at >= NOW() - ($7 * INTERVAL '1 day'))
  AND ($8::text IS NULL OR $8 = '' OR c.display_name ILIKE '%' || $8 || '%'
       OR c.email ILIKE '%' || $8 || '%'
       OR c.phone_number ILIKE '%' || $8 || '%'
       OR c.numero_documento ILIKE '%' || $8 || '%')
  AND c.consent_status = 'granted'
  AND c.phone_number ~ '^\+[1-9][0-9]{7,14}$'
ORDER BY COALESCE(c.last_message_at, c.created_at) DESC NULLS LAST
LIMIT $9 OFFSET $10;

-- name: CountSegmentContacts :one
SELECT
  COUNT(*) FILTER (WHERE c.consent_status = 'granted' AND c.phone_number ~ '^\+[1-9][0-9]{7,14}$') AS total,
  COUNT(*) FILTER (WHERE NOT (c.consent_status = 'granted' AND c.phone_number ~ '^\+[1-9][0-9]{7,14}$')) AS excluded_by_gates
FROM crm.contacts c
WHERE c.organization_id = $1
  AND ($2::text IS NULL OR $2 = '' OR c.source = $2)
  AND ($3::text IS NULL OR $3 = '' OR c.lead_status = $3)
  AND ($4::int = 0 OR c.company_id = $4)
  AND ($5::int = 0 OR c.assigned_to = $5)
  AND ($6::int[] IS NULL OR EXISTS (
      SELECT 1
      FROM crm.entity_tags et
      JOIN crm.tags t ON t.id = et.tag_id AND t.organization_id = c.organization_id
      WHERE et.entity_type = 'contact' AND et.entity_id = c.id
        AND et.tag_id = ANY($6)))
  AND ($7::int = 0 OR c.last_message_at >= NOW() - ($7 * INTERVAL '1 day'))
  AND ($8::text IS NULL OR $8 = '' OR c.display_name ILIKE '%' || $8 || '%'
       OR c.email ILIKE '%' || $8 || '%'
       OR c.phone_number ILIKE '%' || $8 || '%'
       OR c.numero_documento ILIKE '%' || $8 || '%');

-- ============================================================
-- Segments CRUD
-- ============================================================

-- name: CreateSegment :one
INSERT INTO crm.segments (organization_id, nombre, filter_spec, created_by)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: UpdateSegment :one
UPDATE crm.segments
SET nombre = $3, filter_spec = $4, updated_at = NOW()
WHERE id = $1 AND organization_id = $2
RETURNING *;

-- name: DeleteSegment :exec
DELETE FROM crm.segments
WHERE id = $1 AND organization_id = $2;

-- name: GetSegment :one
SELECT * FROM crm.segments
WHERE id = $1 AND organization_id = $2;

-- name: ListSegments :many
SELECT * FROM crm.segments
WHERE organization_id = $1
ORDER BY created_at DESC;

-- ============================================================
-- Campaigns + recipient snapshot
-- ============================================================

-- name: CreateCampaign :one
INSERT INTO crm.campaigns (organization_id, nombre, segment_id, created_by)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetCampaign :one
SELECT * FROM crm.campaigns
WHERE id = $1 AND organization_id = $2;

-- name: ListCampaigns :many
SELECT * FROM crm.campaigns
WHERE organization_id = $1
ORDER BY created_at DESC;

-- name: LaunchCampaign :one
UPDATE crm.campaigns
SET status = 'ready', recipient_count = $3, launched_at = NOW(), updated_at = NOW()
WHERE id = $1 AND organization_id = $2 AND status = 'draft'
RETURNING *;

-- name: SnapshotCampaignRecipients :execrows
INSERT INTO crm.campaign_recipients (campaign_id, contact_id)
SELECT $1, unnest($2::int[])
ON CONFLICT (campaign_id, contact_id) DO NOTHING;

-- name: ListCampaignRecipients :many
SELECT r.id, r.campaign_id, r.contact_id, r.status, r.whatsapp_message_id,
       r.error, r.created_at, r.updated_at,
       c.phone_number, c.display_name
FROM crm.campaign_recipients r
JOIN crm.contacts c ON c.id = r.contact_id
WHERE r.campaign_id = $1
ORDER BY r.id
LIMIT $2 OFFSET $3;
