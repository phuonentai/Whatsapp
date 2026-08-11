package auth

import (
	"context"

	"github.com/gin-gonic/gin"

	"github.com/moasq/go-b2b-starter/internal/platform/authcontext"
)

// Type aliases keep the request-context seam types available under the auth
// package for backward compatibility. New code should use authcontext types.
type (
	Role           = authcontext.Role
	Permission     = authcontext.Permission
	Identity       = authcontext.Identity
	RequestContext = authcontext.RequestContext
)

// NewPermission creates a permission from resource and action.
func NewPermission(resource, action string) Permission {
	return authcontext.NewPermission(resource, action)
}

// This file re-exports the request-context seam owned by the platform
// authcontext package. The auth middleware populates the context through the
// authcontext accessors, and modules read it back via authcontext directly.
// These aliases keep existing auth-module code (middleware, adapters) compiling
// without an import rewrite inside the module.

// SetIdentity stores the Identity in the Gin context.
func SetIdentity(c *gin.Context, identity *Identity) {
	authcontext.SetIdentity(c, identity)
}

// GetIdentity retrieves the Identity from the Gin context.
func GetIdentity(c *gin.Context) *Identity {
	return authcontext.GetIdentity(c)
}

// MustGetIdentity retrieves the Identity from the Gin context, panicking if unset.
func MustGetIdentity(c *gin.Context) *Identity {
	return authcontext.MustGetIdentity(c)
}

// SetRequestContext stores the RequestContext in the Gin context.
func SetRequestContext(c *gin.Context, reqCtx *RequestContext) {
	authcontext.SetRequestContext(c, reqCtx)
}

// GetRequestContext retrieves the RequestContext from the Gin context.
func GetRequestContext(c *gin.Context) *RequestContext {
	return authcontext.GetRequestContext(c)
}

// MustGetRequestContext retrieves the RequestContext from the Gin context, panicking if unset.
func MustGetRequestContext(c *gin.Context) *RequestContext {
	return authcontext.MustGetRequestContext(c)
}

// GetOrganizationID is a convenience function to get the database organization ID.
func GetOrganizationID(c *gin.Context) int32 {
	return authcontext.GetOrganizationID(c)
}

// GetAccountID is a convenience function to get the database account ID.
func GetAccountID(c *gin.Context) int32 {
	return authcontext.GetAccountID(c)
}

// WithIdentity adds the Identity to a context.Context.
func WithIdentity(ctx context.Context, identity *Identity) context.Context {
	return authcontext.WithIdentity(ctx, identity)
}

// IdentityFromContext retrieves the Identity from a context.Context.
func IdentityFromContext(ctx context.Context) *Identity {
	return authcontext.IdentityFromContext(ctx)
}

// WithRequestContext adds the RequestContext to a context.Context.
func WithRequestContext(ctx context.Context, reqCtx *RequestContext) context.Context {
	return authcontext.WithRequestContext(ctx, reqCtx)
}

// RequestContextFromContext retrieves the RequestContext from a context.Context.
func RequestContextFromContext(ctx context.Context) *RequestContext {
	return authcontext.RequestContextFromContext(ctx)
}
