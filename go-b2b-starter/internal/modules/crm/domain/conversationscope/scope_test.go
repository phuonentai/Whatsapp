package conversationscope

import (
	"testing"

	"github.com/moasq/go-b2b-starter/internal/platform/authcontext"
)

func perms(list ...string) []authcontext.Permission {
	out := make([]authcontext.Permission, 0, len(list))
	for _, p := range list {
		out = append(out, authcontext.Permission(p))
	}
	return out
}

// Resolve: regla de unión — assignee | owner | view_all | (NULL + view_unassigned).
func TestResolve_ViewAllFromPermission(t *testing.T) {
	s := Resolve("member-1", perms("inbox:view", "inbox:view_all"), true)
	if !s.Enabled() || !s.ViewAll || s.ViewUnassigned || s.Reassign {
		t.Fatalf("unexpected scope: %+v", s)
	}
	if s.MemberID != "member-1" {
		t.Fatalf("member id not propagated: %+v", s)
	}
}

func TestResolve_OrgManageEqualsViewAll(t *testing.T) {
	// org:manage concede visibilidad org-wide (escenario "Todos" del spec
	// inbox-ui) sin conceder inbox:reassign ni destructivas.
	s := Resolve("admin-1", perms("org:manage"), true)
	if !s.ViewAll {
		t.Fatalf("org:manage should imply view_all, got %+v", s)
	}
	if s.Reassign {
		t.Fatalf("org:manage must not grant reassign implicitly: %+v", s)
	}
}

func TestResolve_QueuePermission(t *testing.T) {
	s := Resolve("member-2", perms("inbox:view_unassigned"), true)
	if !s.ViewUnassigned || s.ViewAll {
		t.Fatalf("unexpected scope: %+v", s)
	}
}

func TestResolve_FlagOffIsOrgScope(t *testing.T) {
	// Free tier: flag false → scope deshabilitado (bandeja org-scope).
	s := Resolve("member-3", perms("inbox:view"), false)
	if s.Enabled() {
		t.Fatalf("flag off must disable scoping: %+v", s)
	}
}

func TestResolve_NoScopePermissions(t *testing.T) {
	// Miembro con inbox:view pero sin permisos de scope → nada visible salvo
	// sus asignaciones/ownership (query layer).
	s := Resolve("member-4", perms("inbox:view"), true)
	if s.ViewAll || s.ViewUnassigned || s.Reassign {
		t.Fatalf("member without scope perms must have none: %+v", s)
	}
}

// Composición supervisor: view_all + reply/reassign sin org:manage ni
// destructivas — es el scope; las acciones destructivas se rechazan en el
// middleware de permiso, no en el scope.
func TestResolve_SupervisorComposition(t *testing.T) {
	s := Resolve("manager-1", perms("inbox:view_all", "inbox:reply", "inbox:reassign"), true)
	if !s.ViewAll || !s.Reassign {
		t.Fatalf("supervisor composition lost: %+v", s)
	}
}

// ParseViewScope: normalización del parámetro scope de la lista.
func TestParseViewScope(t *testing.T) {
	cases := map[string]ViewScope{
		"":      ViewScopeDefault,
		"mine":  ViewScopeMine,
		"queue": ViewScopeQueue,
		"all":   ViewScopeAll,
		"other": ViewScopeDefault,
		"ALL":   ViewScopeDefault,
	}
	for raw, want := range cases {
		if got := ParseViewScope(raw); got != want {
			t.Fatalf("ParseViewScope(%q) = %q, want %q", raw, got, want)
		}
	}
}

// Contrato rol→scope (fallback dev/mock, NUNCA fuente runtime).
func TestRoleContract_Normalize(t *testing.T) {
	if got := NormalizeRoleID("stytch_admin"); got != "admin" {
		t.Fatalf("NormalizeRoleID(stytch_admin) = %q", got)
	}
	if got := NormalizeRoleID(" admin "); got != "admin" {
		t.Fatalf("NormalizeRoleID should trim: %q", got)
	}
}

func TestRoleContract_ManagerHasViewAllAndReassign(t *testing.T) {
	if !HasScopePermission("stytch_manager", PermInboxViewAll) {
		t.Fatal("manager fallback must include inbox:view_all")
	}
	if !HasScopePermission("manager", PermInboxReassign) {
		t.Fatal("manager fallback must include inbox:reassign")
	}
	if HasScopePermission("manager", PermInboxViewUnassigned) {
		t.Fatal("manager fallback must NOT include view_unassigned (decisión de producto)")
	}
}

func TestRoleContract_AdminHasAllScopes(t *testing.T) {
	for _, p := range []string{PermInboxViewAll, PermInboxViewUnassigned, PermInboxReassign} {
		if !HasScopePermission("admin", p) {
			t.Fatalf("admin fallback must include %s", p)
		}
	}
}

func TestRoleContract_MemberHasNoScopePermissions(t *testing.T) {
	if HasScopePermission("member", PermInboxViewAll) ||
		HasScopePermission("member", PermInboxViewUnassigned) ||
		HasScopePermission("member", PermInboxReassign) {
		t.Fatal("member fallback must have no scope permissions by default")
	}
}

func TestRoleContract_UnknownRoleFailsClosed(t *testing.T) {
	if HasScopePermission("unknown_role", PermInboxViewAll) {
		t.Fatal("unknown role must fail closed (no scope permissions)")
	}
}
