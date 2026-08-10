-- name: CreateClientPayment :one
INSERT INTO payments.client_payments (organization_id, deal_id, invoice_id, amount_cop, commission_cop, mp_preference_id)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: GetClientPaymentByPreferenceID :one
SELECT * FROM payments.client_payments
WHERE mp_preference_id = $1
LIMIT 1;

-- name: GetClientPaymentByPaymentID :one
SELECT * FROM payments.client_payments
WHERE mp_payment_id = $1
LIMIT 1;

-- name: AttachPaymentID :one
UPDATE payments.client_payments
SET mp_payment_id = $2
WHERE id = $1 AND status = 'pending' AND mp_payment_id IS NULL
RETURNING *;

-- name: UpdatePaymentStatus :one
UPDATE payments.client_payments
SET status = $2, mp_payment_id = COALESCE($3, mp_payment_id), paid_at = COALESCE($4, paid_at)
WHERE id = $1 AND status = 'pending'
RETURNING *;
