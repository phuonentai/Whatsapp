// Package scope provides the conversation row-scoping middleware
// (conversation-row-scoping change, task 3.3).
//
// The middleware:
//  1. Resuelve el Scope del miembro desde la identidad (permisos de la política
//     Stytch cacheada) y el entitlement `conversation_row_scoping`.
//  2. Lo adjunta al contexto de Gin + context.Context para el query layer.
//  3. Si la capa RLS opt-in está activa (flag de deploy), abre la transacción
//     del request, setea las session vars de scope con `set_config(..., true)`
//     (equivalente transaccional a SET LOCAL — NUNCA SET a nivel sesión sobre
//     el pool) y provee la store transaccional vía context (dbctx).
//
// Fail-closed + observabilidad: si la transacción/vars no pueden setearse con
// RLS activa, el request interactivo responde 500 explícito (nunca lista
// vacía silenciosa) y se registra una anomalía.
package scope

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	postgres "github.com/moasq/go-b2b-starter/internal/db/postgres/sqlc/gen"
	"github.com/moasq/go-b2b-starter/internal/modules/crm/domain/conversationscope"
	"github.com/moasq/go-b2b-starter/internal/platform/authcontext"
	"github.com/moasq/go-b2b-starter/internal/platform/dbctx"
	"github.com/moasq/go-b2b-starter/internal/platform/logger"
	loggerdomain "github.com/moasq/go-b2b-starter/internal/platform/logger/domain"
)

// MiddlewareConfig controla el comportamiento del middleware de scope.
type MiddlewareConfig struct {
	// RLSE enabled: cuando true, el middleware abre la transacción del
	// request y setea las session vars (la migración 000042 RLS debe estar
	// aplicada). Opt-in vía flag de deploy (POSTGRES_RLS_ENABLED).
	RLSEnabled bool
}

// NewMiddleware crea el middleware de scope de conversaciones. `pool` y
// `store` son obligatorios para el modo RLS (transacción + vars).
func NewMiddleware(pool *pgxpool.Pool, store postgres.Store, cfg MiddlewareConfig, log logger.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		reqCtx := authcontext.GetRequestContext(c)
		if reqCtx == nil || reqCtx.Identity == nil {
			// Sin identidad (debería estar cubierto por RequireAuth): fail-closed
			// solo si RLS activa; si no, no bloquear el path.
			if cfg.RLSEnabled {
				abortScopeError(c, log, "scope middleware: request sin identidad con RLS activa", nil)
				return
			}
			c.Next()
			return
		}

		scopeValue := conversationscope.ResolveFromRequest(c)
		conversationscope.SetScope(c, scopeValue)

		if !cfg.RLSEnabled {
			c.Next()
			return
		}

		// RLS activa: abrir transacción del request, setear vars de scope.
		ctx, cancel := context.WithTimeout(c.Request.Context(), 15*time.Second)
		defer cancel()

		tx, err := pool.Begin(ctx)
		if err != nil {
			abortScopeError(c, log, "scope middleware: no se pudo abrir transacción con RLS activa", err)
			return
		}
		defer func() {
			_ = tx.Rollback(context.Background())
		}()

		if err := setScopeVars(ctx, tx, reqCtx.OrganizationID, scopeValue); err != nil {
			abortScopeError(c, log, "scope middleware: no se pudieron setear session vars con RLS activa", err)
			return
		}

		// Proveer la store transaccional al query layer (los repositorios de
		// conversaciones la prefieren vía dbctx).
		c.Request = c.Request.WithContext(dbctx.WithStore(c.Request.Context(), postgres.New(tx)))

		c.Next()

		// Commit solo si no hubo panic; en error del handler el rollback del
		// defer deja la transacción descartada (reads-only de todos modos).
		if !c.IsAborted() {
			_ = tx.Commit(context.Background())
		}
	}
}

// setScopeVars setea las session vars de scope con set_config(is_local=true),
// equivalente transaccional a SET LOCAL. Cuando el flag de la feature está
// apagado (free tier), la política RLS debe permitir org-wide: se setea
// app.is_view_all = true (Decisión 8 del design).
func setScopeVars(ctx context.Context, tx pgx.Tx, orgID int32, s conversationscope.Scope) error {
	viewAllEffective := s.ViewAll || !s.FlagEnabled

	stmts := []struct {
		name  string
		value string
	}{
		{"app.current_organization_id", strconv.Itoa(int(orgID))},
		{"app.current_member_id", s.MemberID},
		{"app.is_view_all", strconv.FormatBool(viewAllEffective)},
		{"app.is_view_unassigned", strconv.FormatBool(s.ViewUnassigned)},
	}
	for _, st := range stmts {
		if _, err := tx.Exec(ctx, "SELECT set_config($1, $2, true)", st.name, st.value); err != nil {
			return fmt.Errorf("failed to set session var %s: %w", st.name, err)
		}
	}
	return nil
}

// abortScopeError falla el request con 500 explícito (nunca lista vacía
// silenciosa) y registra la anomalía.
func abortScopeError(c *gin.Context, log logger.Logger, msg string, err error) {
	fields := loggerdomain.Fields{
		"path":  c.Request.URL.Path,
		"error": msg,
	}
	if err != nil {
		fields["detail"] = err.Error()
	}
	log.Error("scope anomaly: RLS activa con vars de scope no disponibles", fields)
	c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
		"error":   "scope_unavailable",
		"success": false,
		"detail":  "No se pudo establecer el contexto de datos del request.",
	})
}
