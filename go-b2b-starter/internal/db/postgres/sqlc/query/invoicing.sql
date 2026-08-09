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
