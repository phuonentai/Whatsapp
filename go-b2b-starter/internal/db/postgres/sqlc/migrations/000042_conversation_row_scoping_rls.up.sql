-- 000042_conversation_row_scoping_rls.up.sql
-- conversation-row-scoping — paso 4: RLS opt-in a nivel miembro (primera
-- implementación real de RLS del sistema; patrón del spec vivo
-- lean-data-isolation).
--
-- OPT-IN: aplicar SOLO cuando el deploy del middleware de scope (SET LOCAL de
-- session vars en la transacción del request) y el endpoint de re-asignación
-- están desplegados. Si se aplica antes, los paths interactivos devuelven
-- cero filas (fail-closed) — comportamiento intencional pero inaceptable en
-- un rollout parcial.
--
-- Mecánica:
--   - Rol `app_session` (NOLOGIN, BYPASSRLS): bypass para ingestión de
--     webhook (INSERT) y workers de background (lecturas sin contexto de
--     miembro). El rol de la app debe poder `SET ROLE app_session`: se
--     intenta `GRANT app_session TO CURRENT_USER`; en entornos donde las
--     migraciones corren como superusuario esto es suficiente. Si el rol de
--     la app es distinto, otorgar manualmente:
--       GRANT app_session TO <app_db_role>;
--   - Session vars (seteadas SOLO con SET LOCAL en la transacción del
--     request — nunca SET a nivel sesión sobre el pool):
--       app.current_organization_id  (int)
--       app.current_member_id        (stytch_member_id text)
--       app.is_view_all              (bool)
--       app.is_view_unassigned       (bool)
--   - Políticas:
--       SELECT/UPDATE/DELETE → predicado de unión de scope (mismo resolver
--         que el query layer; defense-in-depth).
--       UPDATE WITH CHECK → solo org (permite re-asignar a un miembro visible
--         para el actor: el USING ya acota las filas actualizables).
--       INSERT → permitido si `app.current_organization_id` no está seteado
--         (servicio/webhook sin contexto de miembro) o si coincide con el
--         org de la fila (path interactivo).
--   - Sin FORCE ROW LEVEL SECURITY: si el rol de la app es el dueño de la
--     tabla (p. ej. dev local con POSTGRES_USER=postgres), RLS se omite y el
--     enforcement queda en el query layer. En producción con rol de app no
--     dueño, RLS aplica normalmente.

-- ============================================================
-- 1. Rol de servicio app_session (bypass de scope)
-- ============================================================

DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'app_session') THEN
    CREATE ROLE app_session NOLOGIN BYPASSRLS;
  END IF;
END $$;

-- El rol que corre la app debe poder cambiar a app_session para ingestión y
-- workers. Si el migrador no tiene permiso de grant (no superusuario), se
-- registra un NOTICE y el ops debe otorgarlo manualmente.
DO $$
BEGIN
  IF CURRENT_USER <> 'app_session' THEN
    EXECUTE format('GRANT app_session TO %I', CURRENT_USER);
    RAISE NOTICE 'granted app_session membership to %', CURRENT_USER;
  END IF;
EXCEPTION WHEN insufficient_privilege THEN
  RAISE NOTICE 'no privilege to grant app_session to %; grant manually: GRANT app_session TO <app_db_role>', CURRENT_USER;
END $$;

-- ============================================================
-- 2. Privilegios para app_session (tabla + secuencia + schema)
-- ============================================================

GRANT USAGE ON SCHEMA crm TO app_session;
GRANT SELECT, INSERT, UPDATE, DELETE ON crm.conversations TO app_session;
GRANT USAGE, SELECT ON SEQUENCE crm.conversations_id_seq TO app_session;

-- ============================================================
-- 3. RLS + políticas sobre crm.conversations
-- ============================================================

ALTER TABLE crm.conversations ENABLE ROW LEVEL SECURITY;

-- SELECT: union scope resolver (assignee | owner de empresa | view_all |
-- cola con view_unassigned) acotado al org del request. Los paths de
-- servicio sin contexto de miembro (workers/webhook con org var no seteada)
-- conservan el query layer org-scoped como control (Decisión 9: sin
-- inanición bajo RLS; equivalente al bypass del rol app_session).
CREATE POLICY conversations_scope_select
  ON crm.conversations
  FOR SELECT
  TO PUBLIC
  USING (
    NULLIF(current_setting('app.current_organization_id', true), '')::integer IS NULL
    OR (
      organization_id = NULLIF(current_setting('app.current_organization_id', true), '')::integer
      AND (
        COALESCE(NULLIF(current_setting('app.is_view_all', true), '')::boolean, false)
        OR assignee_stytch_member_id = NULLIF(current_setting('app.current_member_id', true), '')
        OR EXISTS (
          SELECT 1
          FROM crm.contacts ct
          JOIN crm.companies co
            ON co.id = ct.company_id AND co.organization_id = ct.organization_id
          JOIN organizations.accounts a
            ON a.id = co.owner_account_id AND a.organization_id = co.organization_id
          WHERE ct.id = crm.conversations.contact_id
            AND ct.organization_id = crm.conversations.organization_id
            AND a.stytch_member_id = NULLIF(current_setting('app.current_member_id', true), '')
        )
        OR (
          assignee_stytch_member_id IS NULL
          AND COALESCE(NULLIF(current_setting('app.is_view_unassigned', true), '')::boolean, false)
        )
      )
    )
  );

-- UPDATE: filas actualizables = filas visibles (USING = scope), salvo paths de
-- servicio sin contexto de miembro (workers/webhook con org var no seteada),
-- que conservan el query layer org-scoped como control (Decisión 9: sin
-- inanición de workers). El WITH CHECK solo exige el org del request para
-- permitir re-asignación (el destino deja de estar en el scope del actor tras
-- el cambio, lo que bloquearía el UPDATE si el WITH CHECK repitiera el
-- predicado completo).
CREATE POLICY conversations_scope_update
  ON crm.conversations
  FOR UPDATE
  TO PUBLIC
  USING (
    NULLIF(current_setting('app.current_organization_id', true), '')::integer IS NULL
    OR (
      organization_id = NULLIF(current_setting('app.current_organization_id', true), '')::integer
      AND (
        COALESCE(NULLIF(current_setting('app.is_view_all', true), '')::boolean, false)
        OR assignee_stytch_member_id = NULLIF(current_setting('app.current_member_id', true), '')
        OR EXISTS (
          SELECT 1
          FROM crm.contacts ct
          JOIN crm.companies co
            ON co.id = ct.company_id AND co.organization_id = ct.organization_id
          JOIN organizations.accounts a
            ON a.id = co.owner_account_id AND a.organization_id = co.organization_id
          WHERE ct.id = crm.conversations.contact_id
            AND ct.organization_id = crm.conversations.organization_id
            AND a.stytch_member_id = NULLIF(current_setting('app.current_member_id', true), '')
        )
        OR (
          assignee_stytch_member_id IS NULL
          AND COALESCE(NULLIF(current_setting('app.is_view_unassigned', true), '')::boolean, false)
        )
      )
    )
  )
  WITH CHECK (
    NULLIF(current_setting('app.current_organization_id', true), '')::integer IS NULL
    OR organization_id = NULLIF(current_setting('app.current_organization_id', true), '')::integer
  );

-- DELETE: filas visibles, salvo paths de servicio sin contexto de miembro
-- (no existe borrado de conversaciones por webhook; los UPDATE/DELETE del
-- webhook se limitan a metadata de sistema).
CREATE POLICY conversations_scope_delete
  ON crm.conversations
  FOR DELETE
  TO PUBLIC
  USING (
    NULLIF(current_setting('app.current_organization_id', true), '')::integer IS NULL
    OR (
      organization_id = NULLIF(current_setting('app.current_organization_id', true), '')::integer
      AND (
        COALESCE(NULLIF(current_setting('app.is_view_all', true), '')::boolean, false)
        OR assignee_stytch_member_id = NULLIF(current_setting('app.current_member_id', true), '')
        OR EXISTS (
          SELECT 1
          FROM crm.contacts ct
          JOIN crm.companies co
            ON co.id = ct.company_id AND co.organization_id = ct.organization_id
          JOIN organizations.accounts a
            ON a.id = co.owner_account_id AND a.organization_id = co.organization_id
          WHERE ct.id = crm.conversations.contact_id
            AND ct.organization_id = crm.conversations.organization_id
            AND a.stytch_member_id = NULLIF(current_setting('app.current_member_id', true), '')
        )
        OR (
          assignee_stytch_member_id IS NULL
          AND COALESCE(NULLIF(current_setting('app.is_view_unassigned', true), '')::boolean, false)
        )
      )
    )
  );

-- INSERT: la ingestión (webhook) nunca debe ser bloqueada por el scope; si el
-- org del request no está seteado (servicio sin contexto de miembro) se
-- permite el INSERT; si está seteado, la fila debe pertenecer a ese org.
CREATE POLICY conversations_scope_insert
  ON crm.conversations
  FOR INSERT
  TO PUBLIC
  WITH CHECK (
    NULLIF(current_setting('app.current_organization_id', true), '')::integer IS NULL
    OR organization_id = NULLIF(current_setting('app.current_organization_id', true), '')::integer
  );

COMMENT ON POLICY conversations_scope_select ON crm.conversations IS
  'conversation-row-scoping: scope de lectura por miembro (assignee | owner de empresa | view_all | cola con view_unassigned) + org del request';
COMMENT ON POLICY conversations_scope_update ON crm.conversations IS
  'conversation-row-scoping: filas actualizables = visibles; WITH CHECK solo org (permite re-asignación)';
COMMENT ON POLICY conversations_scope_delete ON crm.conversations IS
  'conversation-row-scoping: borrado solo de filas visibles';
COMMENT ON POLICY conversations_scope_insert ON crm.conversations IS
  'conversation-row-scoping: INSERT permitido al servicio (org no seteado) o dentro del org del request';
