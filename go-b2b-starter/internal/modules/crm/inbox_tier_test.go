package crm

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	"github.com/moasq/go-b2b-starter/internal/modules/auth"
	"github.com/moasq/go-b2b-starter/internal/platform/authcontext"
)

// Inbox tier route gates: members with inbox:view/inbox:reply may read and
// reply; close/reopen/template stays org:manage; admins keep org:manage for
// everything (enforcement server-side, not just hidden in the UI).

func inboxIdentity(perms ...string) *authcontext.Identity {
	ps := make([]authcontext.Permission, 0, len(perms))
	for _, p := range perms {
		ps = append(ps, authcontext.Permission(p))
	}
	return &authcontext.Identity{
		UserID:      "member-inbox",
		Roles:       []authcontext.Role{authcontext.RoleMember},
		Permissions: ps,
	}
}

// noInboxRoleIdentity has explicit permissions but no role fallback, so the
// role-based permission grants (member → inbox:view) do not apply.
func noInboxRoleIdentity(perms ...string) *authcontext.Identity {
	ps := make([]authcontext.Permission, 0, len(perms))
	for _, p := range perms {
		ps = append(ps, authcontext.Permission(p))
	}
	return &authcontext.Identity{
		UserID:      "custom-role-member",
		Roles:       []authcontext.Role{},
		Permissions: ps,
	}
}

func adminIdentityWithOrgManage() *authcontext.Identity {
	return &authcontext.Identity{
		UserID: "admin-inbox",
		Roles:  []authcontext.Role{authcontext.RoleAdmin},
		Permissions: []authcontext.Permission{
			authcontext.NewPermission("org", "manage"),
		},
	}
}

func runMiddleware(t *testing.T, identity *authcontext.Identity, middleware gin.HandlerFunc) int {
	t.Helper()
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodGet, "/crm/conversaciones", nil)
	authcontext.SetRequestContext(c, &authcontext.RequestContext{Identity: identity, OrganizationID: 1, AccountID: 1})
	authcontext.SetIdentity(c, identity)
	middleware(c)
	return rec.Code
}

func TestInboxReadRequiresInboxViewOrOrgManage(t *testing.T) {
	// Member with inbox:view passes.
	require.Equal(t, http.StatusOK,
		runMiddleware(t,
			inboxIdentity("inbox:view"),
			auth.RequireAnyPermissionFunc(auth.PermInboxView, auth.PermOrgManage)))

	// Member without inbox:view gets 403 (no role fallback grants it).
	require.Equal(t, http.StatusForbidden,
		runMiddleware(t,
			noInboxRoleIdentity("resource:view"),
			auth.RequireAnyPermissionFunc(auth.PermInboxView, auth.PermOrgManage)))

	// Admin with org:manage passes.
	require.Equal(t, http.StatusOK,
		runMiddleware(t,
			adminIdentityWithOrgManage(),
			auth.RequireAnyPermissionFunc(auth.PermInboxView, auth.PermOrgManage)))
}

func TestInboxReplyRequiresInboxReplyOrOrgManage(t *testing.T) {
	require.Equal(t, http.StatusOK,
		runMiddleware(t,
			inboxIdentity("inbox:view", "inbox:reply"),
			auth.RequireAnyPermissionFunc(auth.PermInboxReply, auth.PermOrgManage)))

	// Member with only inbox:view cannot send.
	require.Equal(t, http.StatusForbidden,
		runMiddleware(t,
			noInboxRoleIdentity("inbox:view"),
			auth.RequireAnyPermissionFunc(auth.PermInboxReply, auth.PermOrgManage)))
}

func TestInboxCloseReopenRequiresOrgManage(t *testing.T) {
	// Member with full inbox tier still cannot close/reopen.
	require.Equal(t, http.StatusForbidden,
		runMiddleware(t,
			inboxIdentity("inbox:view", "inbox:reply"),
			auth.RequirePermissionFunc("org", "manage")))

	require.Equal(t, http.StatusOK,
		runMiddleware(t,
			adminIdentityWithOrgManage(),
			auth.RequirePermissionFunc("org", "manage")))
}

func TestInboxSuggestionsRequireOrgManage(t *testing.T) {
	// Suggestions panel is admin-only in the member tier.
	require.Equal(t, http.StatusForbidden,
		runMiddleware(t,
			inboxIdentity("inbox:view"),
			auth.RequirePermissionFunc("org", "manage")))

	require.Equal(t, http.StatusOK,
		runMiddleware(t,
			adminIdentityWithOrgManage(),
			auth.RequirePermissionFunc("org", "manage")))
}
