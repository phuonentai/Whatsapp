-- name: InsertInvoice :one
INSERT INTO invoicing.invoices (organization_id, deal_id, external_id, cufe, status, pdf_url, amount, currency)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING *;

-- name: GetInvoiceByDeal :one
SELECT * FROM invoicing.invoices
WHERE organization_id = $1 AND deal_id = $2
LIMIT 1;

-- name: GetInvoiceByExternalID :one
SELECT * FROM invoicing.invoices
WHERE organization_id = $1 AND external_id = $2
LIMIT 1;

-- name: GetInvoiceByExternalIDAny :one
SELECT * FROM invoicing.invoices
WHERE external_id = $1
LIMIT 1;

-- name: UpdateInvoiceStatus :one
UPDATE invoicing.invoices
SET status = $3, cufe = COALESCE($4, cufe), pdf_url = COALESCE($5, pdf_url), updated_at = NOW()
WHERE organization_id = $1 AND id = $2
RETURNING *;

-- name: UpdateInvoiceStatusByID :one
UPDATE invoicing.invoices
SET status = $2, cufe = COALESCE($3, cufe), pdf_url = COALESCE($4, pdf_url), updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: UpdateInvoiceNotifiedStatus :one
UPDATE invoicing.invoices
SET notified_status = $2, updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: ListInvoicesByStatus :many
SELECT * FROM invoicing.invoices
WHERE status = $1
ORDER BY id
LIMIT $2;

-- name: GetOrgConnection :one
SELECT * FROM invoicing.org_connections
WHERE organization_id = $1
LIMIT 1;

-- name: UpsertOrgConnection :one
INSERT INTO invoicing.org_connections (organization_id, provider, status, updated_at)
VALUES ($1, $2, $3, NOW())
ON CONFLICT (organization_id) DO UPDATE SET
    provider = EXCLUDED.provider,
    status = EXCLUDED.status,
    updated_at = NOW()
RETURNING *;

-- name: UpdateOrgConnectionStatus :one
UPDATE invoicing.org_connections
SET status = $2, last_error = $3, updated_at = NOW()
WHERE organization_id = $1
RETURNING *;

-- name: UpdateOrgConnectionCredentials :one
UPDATE invoicing.org_connections
SET client_id_enc = $2, client_secret_enc = $3, nit = $4,
    siigo_company_name = $5, last_error = NULL, updated_at = NOW()
WHERE organization_id = $1
RETURNING *;

-- name: DeleteOrgConnection :exec
DELETE FROM invoicing.org_connections
WHERE organization_id = $1;

-- name: GetOrgNumeration :one
SELECT * FROM invoicing.org_numerations
WHERE organization_id = $1
LIMIT 1;

-- name: UpsertOrgNumeration :one
INSERT INTO invoicing.org_numerations (organization_id, mode, resolution_id, prefijo, next_number, confirmed_at)
VALUES ($1, $2, $3, $4, $5, NOW())
ON CONFLICT (organization_id) DO UPDATE SET
    mode = EXCLUDED.mode,
    resolution_id = EXCLUDED.resolution_id,
    prefijo = EXCLUDED.prefijo,
    next_number = EXCLUDED.next_number,
    confirmed_at = NOW(),
    updated_at = NOW()
RETURNING *;

-- name: InsertImportRun :one
INSERT INTO invoicing.import_runs (organization_id, kind, counts, error)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: ListImportRunsByOrg :many
SELECT * FROM invoicing.import_runs
WHERE organization_id = $1
ORDER BY id DESC
LIMIT $2;

-- name: ListOrgConnectionsByStatus :many
SELECT * FROM invoicing.org_connections
WHERE provider = $1 AND status = $2
ORDER BY organization_id;
