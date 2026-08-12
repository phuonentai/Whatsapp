// Package conversationscope implements the conversation row-scoping contract
// (conversation-row-scoping change).
//
// Domain rules (no Stytch SDK / transport imports):
//
//   - Una conversación es visible para un miembro si (regla de unión):
//     assignee_stytch_member_id = miembro, O el contacto pertenece a una
//     crm.companies cuyo owner_account_id (vía accounts.stytch_member_id) es
//     el miembro, O el miembro tiene `inbox:view_all` (u `org:manage`), O la
//     conversación está sin asignar y el miembro tiene `inbox:view_unassigned`.
//
//   - La feature `conversation_row_scoping` (solo planes pagos) es el
//     interruptor: cuando el flag es false (free tier / suscripción inactiva),
//     la bandeja se comporta como antes del change (org-scope).
//
// El contrato tipado rol→scope de este paquete es SOLO tipos de compilación y
// fallback dev/mock (paridad con internal/modules/auth/rbac.go): la fuente
// runtime de decisiones de autorización es la política Stytch cacheada.
package conversationscope

import (
	"context"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/moasq/go-b2b-starter/internal/platform/authcontext"
	"github.com/moasq/go-b2b-starter/internal/platform/features"
)

// scopeKey es la clave de contexto para el Scope del request.
type scopeKey struct{}

const ginScopeKey = "conversation_scope"

// SetScope almacena el Scope del request en el contexto de Gin y en el
// context.Context del request.
func SetScope(c *gin.Context, s Scope) {
	c.Set(ginScopeKey, s)
	c.Request = c.Request.WithContext(WithScope(c.Request.Context(), s))
}

// GetScope recupera el Scope del request desde el contexto de Gin. Devuelve
// un Scope deshabilitado (FlagEnabled=false) si no fue seteado — nunca nil,
// para que los repos callers no nedejen de aplicar el predicado.
func GetScope(c *gin.Context) Scope {
	if v, ok := c.Get(ginScopeKey); ok {
		if s, ok := v.(Scope); ok {
			return s
		}
	}
	return Scope{}
}

// WithScope adjunta un Scope a un context.Context.
func WithScope(ctx context.Context, s Scope) context.Context {
	return context.WithValue(ctx, scopeKey{}, s)
}

// FromContext recupera el Scope desde un context.Context (workers sin
// middleware obtienen un Scope deshabilitado: org-scope).
func FromContext(ctx context.Context) Scope {
	if v, ok := ctx.Value(scopeKey{}).(Scope); ok {
		return v
	}
	return Scope{}
}

// ResolveFromRequest construye el Scope desde el RequestContext (identidad +
// permisos) y el flag de entitlement `conversation_row_scoping`. Si el
// entitlement no está en el contexto de Gin (grupo sin EntitlementMiddleware),
// el flag se resuelve como false (comportamiento org-scope pre-cambio).
func ResolveFromRequest(c *gin.Context) Scope {
	reqCtx := authcontext.GetRequestContext(c)
	if reqCtx == nil || reqCtx.Identity == nil {
		return Scope{}
	}
	flagEnabled := false
	if ent := features.GetEntitlement(c); ent != nil {
		flagEnabled = ent.Features["conversation_row_scoping"]
	}
	return Resolve(reqCtx.Identity.UserID, reqCtx.Identity.Permissions, flagEnabled)
}

// Permission strings (recurso:acción) del scope de bandeja. Constantes
// espejo del fallback dev/mock en internal/modules/auth (PermInboxViewAll,
// PermInboxViewUnassigned, PermInboxReassign).
const (
	PermInboxViewAll       = "inbox:view_all"
	PermInboxViewUnassigned = "inbox:view_unassigned"
	PermInboxReassign      = "inbox:reassign"
	PermInboxReply         = "inbox:reply"
	PermOrgManage          = "org:manage"
)

// Scope describe lo que un miembro puede ver de crm.conversations en un
// request. Es la entrada única del query layer y del middleware de RLS.
type Scope struct {
	// MemberID es el stytch_member_id del miembro (FK lógico; sin tabla
	// local de miembros). Vacío = contexto sin miembro (workers).
	MemberID string

	// ViewAll: inbox:view_all u org:manage (scope org-wide explícito).
	ViewAll bool

	// ViewUnassigned: inbox:view_unassigned (cola de no-asignados).
	ViewUnassigned bool

	// Reassign: inbox:reassign (re-asignar a otro miembro del org).
	Reassign bool

	// FlagEnabled: entitlement conversation_row_scoping (solo planes pagos).
	// false → org-scope (comportamiento pre-cambio).
	FlagEnabled bool
}

// Enabled reports whether row scoping applies for this member.
func (s Scope) Enabled() bool {
	return s.FlagEnabled
}

// ViewScope es el parámetro de vista de la bandeja (query `scope`).
type ViewScope string

const (
	// ViewScopeDefault aplica la regla de unión completa del scope.
	ViewScopeDefault ViewScope = ""
	// ViewScopeMine: assignee = miembro u owner de empresa = miembro.
	ViewScopeMine ViewScope = "mine"
	// ViewScopeQueue: solo no-asignados (requiere ViewUnassigned).
	ViewScopeQueue ViewScope = "queue"
	// ViewScopeAll: todas (requiere ViewAll).
	ViewScopeAll ViewScope = "all"
)

// ParseViewScope normaliza el parámetro scope de la query de lista.
func ParseViewScope(raw string) ViewScope {
	switch ViewScope(raw) {
	case ViewScopeMine, ViewScopeQueue, ViewScopeAll:
		return ViewScope(raw)
	default:
		return ViewScopeDefault
	}
}

// Resolve construye el Scope a partir de los permisos del miembro (identidad
// resuelta por el middleware desde la política Stytch cacheada) y el flag de
// entitlement. `org:manage` equivale a view_all para visibilidad (escenario
// "Todos" del spec inbox-ui), sin conceder acciones destructivas nuevas.
func Resolve(memberID string, permissions []authcontext.Permission, flagEnabled bool) Scope {
	scope := Scope{
		MemberID:    memberID,
		FlagEnabled: flagEnabled,
	}
	for _, p := range permissions {
		switch p.String() {
		case PermInboxViewAll, PermOrgManage:
			scope.ViewAll = true
		case PermInboxViewUnassigned:
			scope.ViewUnassigned = true
		case PermInboxReassign:
			scope.Reassign = true
		}
	}
	return scope
}

// ---------------------------------------------------------------------------
// Contrato tipado rol→scope (compilación + fallback dev/mock únicamente).
//
// La fuente runtime de rol→permiso es la política Stytch cacheada; estos maps
// espejan los permisos nuevos en el fallback dev/mock (task 1.3) y permiten
// que el código compile contra el contrato y que los tests de mock-auth tengan
// paridad. NUNCA deben usarse para decisiones de autorización en runtime.
// ---------------------------------------------------------------------------

// RoleScopeContract define los permisos de scope+acción de bandeja por rol
// normalizado (patrón strings.TrimPrefix(roleID, "stytch_")).
type RoleScopeContract struct {
	// Role es el id de rol normalizado (member/manager/admin).
	Role string

	// ScopePermissions: permisos de scope concedidos al rol.
	ScopePermissions []string

	// ActionPermissions: permisos de acción de bandeja concedidos al rol.
	ActionPermissions []string
}

// roleScopeContracts es el mapa de compilación/fallback. Alineado con la
// asignación documentada en tasks 1.1:
//
//	admin   → inbox:view_all, inbox:view_unassigned, inbox:reassign (+ org:manage)
//	manager → inbox:view_all, inbox:reassign (+ inbox:view, inbox:reply)
//	member  → ninguno de scope por defecto (view_unassigned = decisión de
//	          producto para roles de ventas; no se concede en el fallback base)
var roleScopeContracts = map[string]RoleScopeContract{
	"admin": {
		Role:             "admin",
		ScopePermissions: []string{PermInboxViewAll, PermInboxViewUnassigned, PermInboxReassign},
		ActionPermissions: []string{PermInboxView, PermInboxReply, PermOrgManage},
	},
	"manager": {
		Role:             "manager",
		ScopePermissions: []string{PermInboxViewAll, PermInboxReassign},
		ActionPermissions: []string{PermInboxView, PermInboxReply},
	},
	"member": {
		Role:              "member",
		ScopePermissions:  []string{},
		ActionPermissions: []string{PermInboxView, PermInboxReply},
	},
}

// PermInboxView es el permiso de lectura de bandeja (inbox-member-tier).
const PermInboxView = "inbox:view"

// NormalizeRoleID quita el prefijo del provider (patrón
// strings.TrimPrefix(roleID, "stytch_")).
func NormalizeRoleID(roleID string) string {
	return strings.TrimPrefix(strings.TrimSpace(roleID), "stytch_")
}

// RoleContract devuelve el contrato tipado de un rol normalizado. Devuelve
// un contrato vacío (sin permisos de scope) para roles desconocidos —
// fail-closed por compilación, nunca una ampliación.
func RoleContract(roleID string) RoleScopeContract {
	if c, ok := roleScopeContracts[NormalizeRoleID(roleID)]; ok {
		return c
	}
	return RoleScopeContract{Role: NormalizeRoleID(roleID)}
}

// HasScopePermission reports si el contrato del rol incluye un permiso de
// scope (fallback dev/mock únicamente).
func HasScopePermission(roleID, permission string) bool {
	contract := RoleContract(roleID)
	for _, p := range contract.ScopePermissions {
		if p == permission {
			return true
		}
	}
	for _, p := range contract.ActionPermissions {
		if p == permission {
			return true
		}
	}
	return false
}
