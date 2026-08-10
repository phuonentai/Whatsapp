-- Sales analytics aggregation queries (read-only over existing CRM/invoicing
-- tables). Every query is org-scoped; organization_id is always the first
-- argument.

-- name: RevenueByPeriod :many
SELECT date_trunc($4::text, i.created_at)::date AS periodo,
       COALESCE(SUM(i.amount), 0)::numeric AS monto_total
FROM invoicing.invoices i
WHERE i.organization_id = $1
  AND i.status = 'valid'
  AND i.created_at >= $2::timestamptz
  AND i.created_at < $3::timestamptz
GROUP BY 1
ORDER BY 1;

-- name: TopCustomersByRevenue :many
SELECT COALESCE(c.name, ct.display_name, ct.phone_number, 'Cliente sin nombre') AS nombre,
       COALESCE(SUM(i.amount), 0)::numeric AS monto_total
FROM invoicing.invoices i
JOIN crm.deals d ON d.id = i.deal_id
LEFT JOIN crm.companies c ON c.id = d.company_id
LEFT JOIN crm.contacts ct ON ct.id = d.contact_id
WHERE i.organization_id = $1
  AND i.status = 'valid'
GROUP BY 1
ORDER BY monto_total DESC
LIMIT $2;

-- name: FunnelByStageAggregates :many
SELECT p.es_predeterminado AS es_predeterminado,
       s.nombre AS etapa,
       s.id AS stage_id,
       COUNT(d.id)::int AS cantidad,
       COALESCE(SUM(d.monto), 0)::numeric AS monto_total
FROM crm.deals d
JOIN crm.pipelines p ON p.id = d.pipeline_id
LEFT JOIN crm.pipeline_stages s ON s.id = d.stage_id
WHERE d.organization_id = $1
  AND d.estado = 'abierto'
GROUP BY p.es_predeterminado, s.id
ORDER BY s.id ASC NULLS LAST;

-- name: DealStateCounts :many
SELECT estado AS estado,
       COUNT(*)::int AS cantidad,
       COALESCE(SUM(monto), 0)::numeric AS monto_total
FROM crm.deals
WHERE organization_id = $1
  AND estado IN ('ganado', 'perdido', 'abandonado')
GROUP BY 1
ORDER BY 1;

-- name: InactiveContacts :many
SELECT phone_number AS telefono,
       COALESCE(display_name, '') AS nombre,
       last_message_at AS ultimo_mensaje_at,
       CASE WHEN last_message_at IS NULL THEN 'sin_actividad' ELSE 'inactivo' END AS clasificacion
FROM crm.contacts
WHERE organization_id = $1
  AND (last_message_at < $2::timestamptz OR last_message_at IS NULL)
ORDER BY last_message_at ASC NULLS FIRST;
