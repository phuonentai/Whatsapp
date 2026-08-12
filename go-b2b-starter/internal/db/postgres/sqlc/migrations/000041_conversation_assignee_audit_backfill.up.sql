-- 000041_conversation_assignee_audit_backfill.up.sql
-- conversation-row-scoping — pasos 2 y 3.
--
-- Paso 2: audit pre-migración. Cuantifica las filas que caerán a la cola de
-- no-asignados tras el backfill: conversaciones cuyo contacto no tiene
-- assigned_to ni company.owner_account_id, y casos donde el account
-- referenciado no tiene stytch_member_id (imposible puentear accounts(id) →
-- stytch_member_id). Los resultados alimentan la comunicación de staffing
-- día uno (riesgo R2 del design).
--
-- El audit es solo lectura; se ejecuta ANTES de aplicar la migración (se
-- re-ejecuta tras el ADD COLUMN con el mismo resultado).

-- ---------------------------------------------------------------------------
-- A. Conversaciones que caerán a cola (sin ruta de asignación)
-- ---------------------------------------------------------------------------
-- SELECT
--   COUNT(*) AS total_conversaciones,
--   COUNT(*) FILTER (WHERE ct.assigned_to IS NULL AND co.id IS NULL) AS sin_contacto_empresa,
--   COUNT(*) FILTER (WHERE ct.assigned_to IS NULL AND co.owner_account_id IS NULL) AS sin_owner,
--   COUNT(*) FILTER (WHERE a.stytch_member_id IS NULL) AS owner_sin_stytch_member_id
-- FROM crm.conversations c
-- LEFT JOIN crm.contacts ct ON ct.id = c.contact_id AND ct.organization_id = c.organization_id
-- LEFT JOIN crm.companies co ON co.id = ct.company_id AND co.organization_id = ct.organization_id
-- LEFT JOIN organizations.accounts a
--   ON a.id = COALESCE(ct.assigned_to, co.owner_account_id)
--  AND a.organization_id = c.organization_id;

-- ---------------------------------------------------------------------------
-- B. Desglose por organización (para el plan de staffing por tenant)
-- ---------------------------------------------------------------------------
-- SELECT
--   c.organization_id,
--   COUNT(*) AS total,
--   COUNT(*) FILTER (WHERE a.stytch_member_id IS NULL) AS caeran_a_cola
-- FROM crm.conversations c
-- LEFT JOIN crm.contacts ct ON ct.id = c.contact_id AND ct.organization_id = c.organization_id
-- LEFT JOIN crm.companies co ON co.id = ct.company_id AND co.organization_id = ct.organization_id
-- LEFT JOIN organizations.accounts a
--   ON a.id = COALESCE(ct.assigned_to, co.owner_account_id)
--  AND a.organization_id = c.organization_id
-- GROUP BY c.organization_id
-- ORDER BY caeran_a_cola DESC;

-- ---------------------------------------------------------------------------
-- Paso 3: backfill por lotes (idempotente).
--
-- Prioridad: contacts.assigned_to → fallback companies.owner_account_id
-- (puente vía organizations.accounts.stytch_member_id). Si ambos NULL o el
-- account referenciado no tiene stytch_member_id, la conversación queda NULL
-- (cola). Se actualiza solo conversaciones aún con assignee NULL.
--
-- Batching: loop de lotes por rango de id (BATCH_SIZE filas por UPDATE) en
-- lugar de un UPDATE full-table único, para no mantener una transacción larga
-- con WAL alto ni contención con el poll de 5s de la bandeja.
-- ---------------------------------------------------------------------------

DO $$
DECLARE
  batch_size INT := 5000;
  min_id BIGINT;
  max_id BIGINT;
  cur_id BIGINT;
BEGIN
  SELECT MIN(id), MAX(id) INTO min_id, max_id FROM crm.conversations;
  IF min_id IS NULL THEN
    RAISE NOTICE 'crm.conversations vacía; backfill omitido';
    RETURN;
  END IF;

  cur_id := min_id;
  WHILE cur_id <= max_id LOOP
    UPDATE crm.conversations c
    SET assignee_stytch_member_id = a.stytch_member_id
    FROM crm.contacts ct
    LEFT JOIN crm.companies co
      ON co.id = ct.company_id AND co.organization_id = ct.organization_id
    LEFT JOIN organizations.accounts a
      ON a.id = COALESCE(ct.assigned_to, co.owner_account_id)
     AND a.organization_id = c.organization_id
    WHERE c.id BETWEEN cur_id AND cur_id + batch_size - 1
      AND c.assignee_stytch_member_id IS NULL
      AND ct.id = c.contact_id
      AND ct.organization_id = c.organization_id
      AND a.stytch_member_id IS NOT NULL;

    cur_id := cur_id + batch_size;
  END LOOP;

  RAISE NOTICE 'Backfill de assignee_stytch_member_id completado (ids %..%)', min_id, max_id;
END $$;
