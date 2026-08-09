-- CRM extended queries (new entities for v2 CRM module)

-- Contacts (extended)

-- name: ListContactsByOrganizationFiltered :many
SELECT * FROM crm.contacts
WHERE organization_id = $1
  AND (NULLIF($2, '') IS NULL OR source = $2)
  AND (NULLIF($3, '') IS NULL OR lead_status = $3)
  AND ($4::int = 0 OR company_id = $4)
  AND ($5::int = 0 OR assigned_to = $5)
ORDER BY last_message_at DESC NULLS LAST
LIMIT $6 OFFSET $7;

-- name: SearchContacts :many
SELECT * FROM crm.contacts
WHERE organization_id = $1
  AND (display_name ILIKE '%' || $2 || '%'
    OR email ILIKE '%' || $2 || '%'
    OR phone_number ILIKE '%' || $2 || '%'
    OR numero_documento ILIKE '%' || $2 || '%')
ORDER BY last_message_at DESC NULLS LAST
LIMIT $3 OFFSET $4;

-- name: UpdateContact :one
UPDATE crm.contacts
SET
    email = COALESCE(NULLIF($3, ''), email),
    company_id = COALESCE($4, company_id),
    display_name = COALESCE(NULLIF($5, ''), display_name),
    source = COALESCE(NULLIF($6, ''), source),
    lead_status = COALESCE(NULLIF($7, ''), lead_status),
    job_title = COALESCE(NULLIF($8, ''), job_title),
    assigned_to = COALESCE($9, assigned_to),
    tipo_documento = COALESCE(NULLIF($10, ''), tipo_documento),
    numero_documento = COALESCE(NULLIF($11, ''), numero_documento),
    avatar_url = COALESCE(NULLIF($12, ''), avatar_url),
    metadata = CASE WHEN $13::jsonb IS NOT NULL THEN metadata || $13 ELSE metadata END,
    updated_at = NOW()
WHERE id = $1 AND organization_id = $2
RETURNING *;

-- name: DeleteContact :exec
DELETE FROM crm.contacts
WHERE id = $1 AND organization_id = $2;

-- Companies

-- name: CreateCompany :one
INSERT INTO crm.companies (
    organization_id, name, nit, tipo_empresa, sector, ciudad, departamento,
    website, phone, address, notes, metadata, owner_account_id
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13
) RETURNING *;

-- name: GetCompanyByID :one
SELECT c.*,
    (SELECT COUNT(*) FROM crm.contacts WHERE company_id = c.id) AS total_contactos,
    (SELECT COUNT(*) FROM crm.deals WHERE company_id = c.id) AS total_negocios
FROM crm.companies c
WHERE c.id = $1 AND c.organization_id = $2;

-- name: ListCompaniesByOrganization :many
SELECT c.*,
    (SELECT COUNT(*) FROM crm.contacts WHERE company_id = c.id) AS total_contactos,
    (SELECT COUNT(*) FROM crm.deals WHERE company_id = c.id) AS total_negocios
FROM crm.companies c
WHERE c.organization_id = $1
ORDER BY c.name ASC
LIMIT $2 OFFSET $3;

-- name: SearchCompanies :many
SELECT c.*,
    (SELECT COUNT(*) FROM crm.contacts WHERE company_id = c.id) AS total_contactos,
    (SELECT COUNT(*) FROM crm.deals WHERE company_id = c.id) AS total_negocios
FROM crm.companies c
WHERE c.organization_id = $1
  AND (c.name ILIKE '%' || $2 || '%'
    OR c.nit ILIKE '%' || $2 || '%'
    OR c.sector ILIKE '%' || $2 || '%'
    OR c.ciudad ILIKE '%' || $2 || '%')
ORDER BY c.name ASC
LIMIT $3 OFFSET $4;

-- name: UpdateCompany :one
UPDATE crm.companies
SET
    name = COALESCE(NULLIF($3, ''), name),
    nit = COALESCE(NULLIF($4, ''), nit),
    tipo_empresa = COALESCE(NULLIF($5, ''), tipo_empresa),
    sector = COALESCE(NULLIF($6, ''), sector),
    ciudad = COALESCE(NULLIF($7, ''), ciudad),
    departamento = COALESCE(NULLIF($8, ''), departamento),
    website = COALESCE(NULLIF($9, ''), website),
    phone = COALESCE(NULLIF($10, ''), phone),
    address = COALESCE(NULLIF($11, ''), address),
    notes = COALESCE(NULLIF($12, ''), notes),
    metadata = CASE WHEN $13::jsonb IS NOT NULL THEN metadata || $13 ELSE metadata END,
    owner_account_id = COALESCE($14, owner_account_id),
    updated_at = NOW()
WHERE id = $1 AND organization_id = $2
RETURNING *;

-- name: DeleteCompany :exec
DELETE FROM crm.companies
WHERE id = $1 AND organization_id = $2;

-- Pipelines

-- name: CreatePipeline :one
INSERT INTO crm.pipelines (organization_id, nombre, es_predeterminado, orden)
VALUES ($1, $2, $3, $4) RETURNING *;

-- name: GetPipelineByID :one
SELECT * FROM crm.pipelines
WHERE id = $1 AND organization_id = $2;

-- name: ListPipelinesByOrganization :many
SELECT p.*,
    COALESCE(
        jsonb_agg(
            jsonb_build_object(
                'id', ps.id,
                'nombre', ps.nombre,
                'orden', ps.orden,
                'color', ps.color,
                'probabilidad', ps.probabilidad
            )
            ORDER BY ps.orden ASC
        ) FILTER (WHERE ps.id IS NOT NULL),
        '[]'::jsonb
    ) AS etapas
FROM crm.pipelines p
LEFT JOIN crm.pipeline_stages ps ON ps.pipeline_id = p.id
WHERE p.organization_id = $1
GROUP BY p.id
ORDER BY p.orden ASC;

-- name: GetDefaultPipelineByOrganization :one
SELECT * FROM crm.pipelines
WHERE organization_id = $1 AND es_predeterminado = true
LIMIT 1;

-- name: UpdatePipeline :one
UPDATE crm.pipelines
SET
    nombre = COALESCE(NULLIF($3, ''), nombre),
    es_predeterminado = COALESCE($4, es_predeterminado),
    orden = COALESCE($5, orden),
    updated_at = NOW()
WHERE id = $1 AND organization_id = $2
RETURNING *;

-- name: DeletePipeline :exec
DELETE FROM crm.pipelines
WHERE id = $1 AND organization_id = $2;

-- Pipeline Stages

-- name: CreatePipelineStage :one
INSERT INTO crm.pipeline_stages (pipeline_id, nombre, orden, color, probabilidad)
VALUES ($1, $2, $3, $4, $5) RETURNING *;

-- name: ListStagesByPipeline :many
SELECT * FROM crm.pipeline_stages
WHERE pipeline_id = $1
ORDER BY orden ASC;

-- name: GetStageByID :one
SELECT * FROM crm.pipeline_stages
WHERE id = $1;

-- name: UpdatePipelineStage :one
UPDATE crm.pipeline_stages
SET
    nombre = COALESCE(NULLIF($3, ''), nombre),
    orden = COALESCE($4, orden),
    color = COALESCE(NULLIF($5, ''), color),
    probabilidad = COALESCE($6, probabilidad),
    updated_at = NOW()
WHERE id = $1 AND pipeline_id = $2
RETURNING *;

-- name: DeletePipelineStage :exec
DELETE FROM crm.pipeline_stages
WHERE id = $1 AND pipeline_id = $2;

-- Deals

-- name: CreateDeal :one
INSERT INTO crm.deals (
    organization_id, nombre, contact_id, company_id, pipeline_id, stage_id,
    monto, moneda, fecha_cierre_esperada, estado, probabilidad, notas, metadata, assigned_to
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14
) RETURNING *;

-- name: GetDealByID :one
SELECT d.*,
    c.display_name AS contact_name,
    c.phone_number AS contact_phone,
    co.name AS company_name
FROM crm.deals d
LEFT JOIN crm.contacts c ON c.id = d.contact_id
LEFT JOIN crm.companies co ON co.id = d.company_id
WHERE d.id = $1 AND d.organization_id = $2;

-- name: ListDealsByOrganization :many
SELECT d.*,
    c.display_name AS contact_name,
    c.phone_number AS contact_phone,
    co.name AS company_name
FROM crm.deals d
LEFT JOIN crm.contacts c ON c.id = d.contact_id
LEFT JOIN crm.companies co ON co.id = d.company_id
WHERE d.organization_id = $1
  AND ($2::int = 0 OR d.pipeline_id = $2)
  AND ($3::int = 0 OR d.stage_id = $3)
  AND (NULLIF($4, '') IS NULL OR d.estado = $4)
  AND ($5::int = 0 OR d.contact_id = $5)
ORDER BY d.created_at DESC
LIMIT $6 OFFSET $7;

-- name: UpdateDeal :one
UPDATE crm.deals
SET
    nombre = COALESCE(NULLIF($3, ''), nombre),
    contact_id = COALESCE($4, contact_id),
    company_id = COALESCE($5, company_id),
    monto = COALESCE($6, monto),
    moneda = COALESCE(NULLIF($7, ''), moneda),
    fecha_cierre_esperada = COALESCE($8, fecha_cierre_esperada),
    estado = COALESCE(NULLIF($9, ''), estado),
    probabilidad = COALESCE($10, probabilidad),
    notas = COALESCE(NULLIF($11, ''), notas),
    metadata = CASE WHEN $12::jsonb IS NOT NULL THEN metadata || $12 ELSE metadata END,
    assigned_to = COALESCE($13, assigned_to),
    updated_at = NOW()
WHERE id = $1 AND organization_id = $2
RETURNING *;

-- name: UpdateDealStage :one
UPDATE crm.deals
SET
    stage_id = $3,
    updated_at = NOW()
WHERE id = $1 AND organization_id = $2
RETURNING *;

-- name: DeleteDeal :exec
DELETE FROM crm.deals
WHERE id = $1 AND organization_id = $2;

-- Activities

-- name: CreateActivity :one
INSERT INTO crm.activities (
    organization_id, contact_id, company_id, deal_id, conversation_id,
    tipo, asunto, contenido, estado, fecha_vencimiento, realizada_por, realizada_en, metadata
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13
) RETURNING *;

-- name: GetActivityByID :one
SELECT * FROM crm.activities
WHERE id = $1 AND organization_id = $2;

-- name: ListActivitiesByOrganization :many
SELECT a.*,
    acc.full_name AS realizada_por_nombre
FROM crm.activities a
LEFT JOIN organizations.accounts acc ON acc.id = a.realizada_por
WHERE a.organization_id = $1
  AND (NULLIF($2, '') IS NULL OR a.tipo = $2)
  AND (NULLIF($3, '') IS NULL OR ($3 = 'contact' AND a.contact_id = $4::int)
    OR ($3 = 'company' AND a.company_id = $4::int)
    OR ($3 = 'deal' AND a.deal_id = $4::int))
ORDER BY a.realizada_en DESC
LIMIT $5 OFFSET $6;

-- name: ListActivitiesByContact :many
SELECT a.*,
    acc.full_name AS realizada_por_nombre
FROM crm.activities a
LEFT JOIN organizations.accounts acc ON acc.id = a.realizada_por
WHERE a.contact_id = $1 AND a.organization_id = $2
ORDER BY a.realizada_en DESC
LIMIT $3 OFFSET $4;

-- name: ListActivitiesByDeal :many
SELECT a.*,
    acc.full_name AS realizada_por_nombre
FROM crm.activities a
LEFT JOIN organizations.accounts acc ON acc.id = a.realizada_por
WHERE a.deal_id = $1 AND a.organization_id = $2
ORDER BY a.realizada_en DESC
LIMIT $3 OFFSET $4;

-- name: ListActivitiesByCompany :many
SELECT a.*,
    acc.full_name AS realizada_por_nombre
FROM crm.activities a
LEFT JOIN organizations.accounts acc ON acc.id = a.realizada_por
WHERE a.company_id = $1 AND a.organization_id = $2
ORDER BY a.realizada_en DESC
LIMIT $3 OFFSET $4;

-- Tags

-- name: CreateTag :one
INSERT INTO crm.tags (organization_id, nombre, color)
VALUES ($1, $2, $3) RETURNING *;

-- name: GetTagByID :one
SELECT * FROM crm.tags
WHERE id = $1 AND organization_id = $2;

-- name: ListTagsByOrganization :many
SELECT * FROM crm.tags
WHERE organization_id = $1
ORDER BY nombre ASC;

-- name: UpdateTag :one
UPDATE crm.tags
SET
    nombre = COALESCE(NULLIF($3, ''), nombre),
    color = COALESCE(NULLIF($4, ''), color),
    updated_at = NOW()
WHERE id = $1 AND organization_id = $2
RETURNING *;

-- name: DeleteTag :exec
DELETE FROM crm.tags
WHERE id = $1 AND organization_id = $2;

-- Entity Tags

-- name: AttachTag :one
INSERT INTO crm.entity_tags (tag_id, entity_type, entity_id)
VALUES ($1, $2, $3)
ON CONFLICT (tag_id, entity_type, entity_id) DO NOTHING
RETURNING *;

-- name: DetachTag :exec
DELETE FROM crm.entity_tags
WHERE tag_id = $1 AND entity_type = $2 AND entity_id = $3;

-- name: ListTagsByEntity :many
SELECT t.*
FROM crm.tags t
JOIN crm.entity_tags et ON et.tag_id = t.id
WHERE et.entity_type = $1 AND et.entity_id = $2
ORDER BY t.nombre ASC;

-- name: ListEntitiesByTag :many
SELECT et.entity_type, et.entity_id
FROM crm.entity_tags et
WHERE et.tag_id = $1
ORDER BY et.entity_type, et.entity_id;

-- Usage Queries (for entitlement)

-- name: CountContactsByOrganization :one
SELECT COUNT(*) FROM crm.contacts WHERE organization_id = $1;

-- name: CountDealsByOrganization :one
SELECT COUNT(*) FROM crm.deals WHERE organization_id = $1;
