-- Procurement module queries (add-supplier-inquiry-agent)
-- Org-scoped suppliers/products/runs/recipients/responses/orders + audit log.
-- Transitions are guarded with conditional UPDATEs (transaction-isolated
-- state checks) so outbox/webhook redelivery cannot double-apply.

-- ============================================================
-- Suppliers
-- ============================================================

-- name: CreateSupplier :one
INSERT INTO procurement.suppliers (
    organization_id, contact_id, nit, delivery_days, min_order_amount, notes, is_active
) VALUES (
    $1, $2, $3, $4, $5, $6, TRUE
) RETURNING *;

-- name: GetSupplier :one
SELECT * FROM procurement.suppliers
WHERE id = $1 AND organization_id = $2;

-- name: GetSupplierByContact :one
SELECT * FROM procurement.suppliers
WHERE contact_id = $1 AND organization_id = $2
LIMIT 1;

-- name: ListSuppliersByOrganization :many
SELECT * FROM procurement.suppliers
WHERE organization_id = $1
ORDER BY id DESC
LIMIT $2 OFFSET $3;

-- Suppliers joined with contact display name and phone (list view).
-- name: ListSuppliersWithContact :many
SELECT sup.*, COALESCE(c.display_name, '') AS display_name, c.phone_number AS contact_phone
FROM procurement.suppliers sup
JOIN crm.contacts c ON c.id = sup.contact_id AND c.organization_id = sup.organization_id
WHERE sup.organization_id = $1
ORDER BY sup.id DESC
LIMIT $2 OFFSET $3;

-- name: ListActiveSuppliersByOrganization :many
SELECT * FROM procurement.suppliers
WHERE organization_id = $1 AND is_active = TRUE
ORDER BY id ASC;

-- name: UpdateSupplier :one
UPDATE procurement.suppliers
SET
    delivery_days = COALESCE($3, delivery_days),
    min_order_amount = COALESCE($4, min_order_amount),
    notes = COALESCE($5, notes),
    is_active = $6,
    updated_at = NOW()
WHERE id = $1 AND organization_id = $2
RETURNING *;

-- name: ListSuppliersByIDs :many
SELECT * FROM procurement.suppliers
WHERE organization_id = $1 AND id = ANY($2::int[])
ORDER BY id ASC;

-- Suppliers with their contact display name (drafting greeting, D11).
-- name: ListSuppliersWithDisplay :many
SELECT sup.*, COALESCE(c.display_name, '') AS display_name
FROM procurement.suppliers sup
JOIN crm.contacts c ON c.id = sup.contact_id AND c.organization_id = sup.organization_id
WHERE sup.organization_id = $1 AND sup.id = ANY($2::int[])
ORDER BY sup.id ASC;

-- ============================================================
-- Products
-- ============================================================

-- name: CreateProduct :one
INSERT INTO procurement.products (organization_id, name, sku, unit, is_active)
VALUES ($1, $2, $3, $4, TRUE)
RETURNING *;

-- name: GetProduct :one
SELECT * FROM procurement.products
WHERE id = $1 AND organization_id = $2;

-- name: ListProductsByOrganization :many
SELECT * FROM procurement.products
WHERE organization_id = $1
ORDER BY id DESC
LIMIT $2 OFFSET $3;

-- name: ListProductsByIDs :many
SELECT * FROM procurement.products
WHERE organization_id = $1 AND id = ANY($2::int[])
ORDER BY id ASC;

-- name: UpdateProduct :one
UPDATE procurement.products
SET
    name = COALESCE($3, name),
    sku = COALESCE($4, sku),
    unit = COALESCE($5, unit),
    is_active = $6,
    updated_at = NOW()
WHERE id = $1 AND organization_id = $2
RETURNING *;

-- ============================================================
-- Inquiry runs
-- ============================================================

-- name: CreateInquiryRun :one
INSERT INTO procurement.inquiry_runs (
    organization_id, status, source, nota, created_by_member_id
) VALUES (
    $1, 'draft', 'manual', $2, $3
) RETURNING *;

-- name: GetInquiryRun :one
SELECT * FROM procurement.inquiry_runs
WHERE id = $1 AND organization_id = $2;

-- name: ListInquiryRunsByOrganization :many
SELECT * FROM procurement.inquiry_runs
WHERE organization_id = $1
ORDER BY id DESC
LIMIT $2 OFFSET $3;

-- Guarded run transition: only applies when the run is in the expected
-- status; returns no row when the state moved concurrently.
-- name: UpdateRunStatusFrom :one
UPDATE procurement.inquiry_runs
SET
    status = $4,
    sent_at = CASE WHEN $4 = 'sending' THEN COALESCE(sent_at, NOW()) ELSE sent_at END,
    completed_at = CASE
        WHEN $4 IN ('completed', 'partially_answered', 'failed', 'cancelled')
        THEN COALESCE(completed_at, NOW())
        ELSE completed_at
    END,
    updated_at = NOW()
WHERE id = $1 AND organization_id = $2 AND status = $3
RETURNING *;

-- ============================================================
-- Recipients
-- ============================================================

-- name: CreateInquiryRecipient :one
INSERT INTO procurement.inquiry_recipients (
    organization_id, run_id, supplier_id, contact_id, drafted_message, status
) VALUES (
    $1, $2, $3, $4, $5, 'pending'
) RETURNING *;

-- name: GetInquiryRecipient :one
SELECT * FROM procurement.inquiry_recipients
WHERE id = $1 AND organization_id = $2;

-- name: ListRunRecipients :many
SELECT * FROM procurement.inquiry_recipients
WHERE run_id = $1 AND organization_id = $2
ORDER BY id ASC;

-- Recipients with their contact phone (fan-out send target).
-- name: ListRunRecipientsWithPhone :many
SELECT r.*, c.phone_number AS contact_phone
FROM procurement.inquiry_recipients r
JOIN crm.contacts c ON c.id = r.contact_id AND c.organization_id = r.organization_id
WHERE r.run_id = $1 AND r.organization_id = $2
ORDER BY r.id ASC;

-- Guarded transition pending -> sent (idempotent under redelivery: a second
-- dispatch finds no 'pending' row and is a no-op).
-- name: MarkRecipientSent :one
UPDATE procurement.inquiry_recipients
SET status = 'sent', provider_message_id = $3, sent_at = NOW(), updated_at = NOW()
WHERE id = $1 AND organization_id = $2 AND status = 'pending'
RETURNING *;

-- Guarded transition pending|sent -> answered.
-- name: MarkRecipientAnswered :one
UPDATE procurement.inquiry_recipients
SET status = 'answered', answered_at = NOW(), updated_at = NOW()
WHERE id = $1 AND organization_id = $2 AND status IN ('pending', 'sent')
RETURNING *;

-- Guarded transition sent -> timed_out (lazy read-time reconciliation, D12).
-- name: MarkRecipientTimedOut :one
UPDATE procurement.inquiry_recipients
SET status = 'timed_out', updated_at = NOW()
WHERE id = $1 AND organization_id = $2 AND status = 'sent'
RETURNING *;

-- Guarded transition pending|sent -> failed (send dead-lettered).
-- name: MarkRecipientFailed :one
UPDATE procurement.inquiry_recipients
SET status = 'failed', updated_at = NOW()
WHERE id = $1 AND organization_id = $2 AND status IN ('pending', 'sent')
RETURNING *;

-- Active-recipient lookup by phone: the hot path for every inbound
-- whatsapp.message.received event (procurement subscriber + agent skip check).
-- name: ListActiveRecipientsByPhone :many
SELECT r.*
FROM procurement.inquiry_recipients r
JOIN procurement.inquiry_runs run ON run.id = r.run_id
JOIN crm.contacts c ON c.id = r.contact_id AND c.organization_id = r.organization_id
WHERE r.organization_id = $1
  AND c.phone_number = $2
  AND run.status IN ('sending', 'awaiting_responses')
  AND r.status IN ('pending', 'sent')
ORDER BY r.id DESC;

-- Recipients of a run that are still awaiting a reply (for run completion
-- evaluation and the lazy timeout reconciliation).
-- name: ListAwaitingRecipients :many
SELECT * FROM procurement.inquiry_recipients
WHERE run_id = $1 AND organization_id = $2 AND status = 'sent'
ORDER BY id ASC;

-- Expired sent recipients (lazy timeout, D12): sent earlier than the window.
-- name: ListExpiredSentRecipients :many
SELECT r.*
FROM procurement.inquiry_recipients r
JOIN procurement.inquiry_runs run ON run.id = r.run_id
WHERE r.organization_id = $1
  AND r.run_id = $2
  AND run.status = 'awaiting_responses'
  AND r.status = 'sent'
  AND r.sent_at <= NOW() - ($3::int * INTERVAL '1 hour')
ORDER BY r.id ASC;

-- ============================================================
-- Responses (idempotent insert on (recipient_id, raw_message_id))
-- ============================================================

-- name: CreateInquiryResponse :one
INSERT INTO procurement.inquiry_responses (
    organization_id, recipient_id, raw_message_id, extracted, resumen, confidence, requiere_humano
) VALUES (
    $1, $2, $3, $4, $5, $6, $7
)
ON CONFLICT (recipient_id, raw_message_id) DO NOTHING
RETURNING *;

-- name: GetResponseByRecipientMessage :one
SELECT * FROM procurement.inquiry_responses
WHERE recipient_id = $1 AND raw_message_id = $2;

-- name: ListRunResponses :many
SELECT resp.*
FROM procurement.inquiry_responses resp
JOIN procurement.inquiry_recipients r ON r.id = resp.recipient_id
WHERE r.run_id = $1 AND r.organization_id = $2
ORDER BY resp.id ASC;

-- ============================================================
-- Orders (idempotent on (run_id, supplier_id), D13)
-- ============================================================

-- name: CreateOrder :one
INSERT INTO procurement.orders (
    organization_id, run_id, supplier_id, contact_id, items, notes,
    confirm_message, status, created_by_member_id, blocked_reason
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10
)
ON CONFLICT (run_id, supplier_id) DO NOTHING
RETURNING *;

-- name: GetOrderByRunSupplier :one
SELECT * FROM procurement.orders
WHERE run_id = $1 AND supplier_id = $2;

-- name: GetOrder :one
SELECT * FROM procurement.orders
WHERE id = $1 AND organization_id = $2;

-- name: UpdateOrderNegocioID :one
UPDATE procurement.orders
SET negocio_id = $3, updated_at = NOW()
WHERE id = $1 AND organization_id = $2
RETURNING *;

-- name: UpdateOrderConfirmSent :one
UPDATE procurement.orders
SET status = 'confirm_sent', updated_at = NOW()
WHERE id = $1 AND organization_id = $2 AND status = 'placed'
RETURNING *;

-- name: UpdateOrderSendBlocked :one
UPDATE procurement.orders
SET status = 'send_blocked', blocked_reason = $3, updated_at = NOW()
WHERE id = $1 AND organization_id = $2
RETURNING *;

-- name: UpdateOrderConfirmFailed :one
UPDATE procurement.orders
SET status = 'confirm_failed', updated_at = NOW()
WHERE id = $1 AND organization_id = $2 AND status = 'placed'
RETURNING *;

-- name: ListRunOrders :many
SELECT * FROM procurement.orders
WHERE run_id = $1 AND organization_id = $2
ORDER BY id ASC;

-- ============================================================
-- Board read model
-- ============================================================

-- name: GetRunBoardRows :many
SELECT
    r.id AS recipient_id,
    r.status AS recipient_status,
    r.sent_at,
    r.answered_at,
    r.provider_message_id,
    sup.id AS supplier_id,
    sup.nit,
    sup.delivery_days,
    sup.min_order_amount,
    c.id AS contact_id,
    COALESCE(c.display_name, '') AS display_name,
    c.phone_number,
    resp.extracted,
    resp.resumen,
    resp.confidence,
    resp.requiere_humano
FROM procurement.inquiry_recipients r
JOIN procurement.suppliers sup ON sup.id = r.supplier_id AND sup.organization_id = r.organization_id
JOIN crm.contacts c ON c.id = r.contact_id AND c.organization_id = r.organization_id
LEFT JOIN LATERAL (
    SELECT extracted, resumen, confidence, requiere_humano
    FROM procurement.inquiry_responses resp
    WHERE resp.recipient_id = r.id
    ORDER BY resp.id DESC
    LIMIT 1
) resp ON TRUE
WHERE r.run_id = $1 AND r.organization_id = $2
ORDER BY r.id ASC;

-- ============================================================
-- Cross-module helpers (orders path)
-- ============================================================

-- Create (or reuse) the supplier's CRM contact with NIT + org-declared
-- consent granted (D11). Duplicate (organization_id, phone_number) is
-- rejected by the contacts unique index -> handler maps to 400 Spanish.
-- name: CreateSupplierContact :one
INSERT INTO crm.contacts (
    organization_id, phone_number, display_name, source,
    tipo_documento, numero_documento, consent_status, consented_at
) VALUES (
    $1, $2, $3, 'manual', 'NIT', $4, 'granted', NOW()
) RETURNING *;

-- Default pipeline (auto-seeded "Pipeline de Ventas" per pipeline-management).
-- name: GetDefaultPipelineID :one
SELECT id FROM crm.pipelines
WHERE organization_id = $1 AND es_predeterminado = TRUE
ORDER BY id ASC
LIMIT 1;

-- Kill switch state (agent_settings); FALSE when no settings row exists yet.
-- name: GetAgentKillSwitch :one
SELECT EXISTS(
    SELECT 1 FROM agent.agent_settings
    WHERE organization_id = $1 AND kill_switch = TRUE
) AS kill_switch_on;

-- ============================================================
-- Audit log
-- ============================================================

-- name: InsertProcurementAudit :one
INSERT INTO procurement.audit_log (
    organization_id, entity_type, entity_id, action, decision, reason, member_id, metadata
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8
) RETURNING *;
