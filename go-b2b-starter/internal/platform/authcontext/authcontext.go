// Package authcontext provides the platform-owned request context seam.
//
// Modules read the authenticated identity, organization, and account IDs for
// the current request through this package instead of importing the auth
// module. The auth middleware populates the context; handlers and services
// read it back through the accessors defined here.
package authcontext

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// Context keys for storing auth data.
// Using unexported type to prevent collisions with other packages.
type contextKey string

const (
	// identityKey is the context key for storing the authenticated Identity.
	identityKey contextKey = "auth_identity"

	// requestContextKey is the context key for storing the RequestContext.
	requestContextKey contextKey = "auth_request_context"
)

// Role represents a user role in the system.
type Role string

// String returns the string representation of the role.
func (r Role) String() string {
	return string(r)
}

// Core RBAC roles. These must match the roles configured in the auth provider.
const (
	RoleMember  Role = "member"
	RoleManager Role = "manager"
	RoleAdmin   Role = "admin"
)

// Permission represents an authorization permission in "resource:action" format.
type Permission string

// NewPermission creates a permission from resource and action.
func NewPermission(resource, action string) Permission {
	return Permission(fmt.Sprintf("%s:%s", resource, action))
}

// String returns the string representation of the permission.
func (p Permission) String() string {
	return string(p)
}

// Resource returns the resource part of the permission.
func (p Permission) Resource() string {
	parts := strings.SplitN(string(p), ":", 2)
	if len(parts) > 0 {
		return parts[0]
	}
	return ""
}

// Action returns the action part of the permission.
func (p Permission) Action() string {
	parts := strings.SplitN(string(p), ":", 2)
	if len(parts) > 1 {
		return parts[1]
	}
	return ""
}

// IsValid checks if the permission has both resource and action parts.
func (p Permission) IsValid() bool {
	parts := strings.SplitN(string(p), ":", 2)
	return len(parts) == 2 && parts[0] != "" && parts[1] != ""
}

// Matches checks if this permission matches another permission.
func (p Permission) Matches(other Permission) bool {
	return p == other
}

// MatchesWithWildcard checks if this permission matches another, supporting
// wildcards ("*:*", "resource:*", "*:action").
func (p Permission) MatchesWithWildcard(other Permission) bool {
	if p == other {
		return true
	}

	myResource := p.Resource()
	myAction := p.Action()
	otherResource := other.Resource()
	otherAction := other.Action()

	resourceMatch := myResource == "*" || myResource == otherResource
	actionMatch := myAction == "*" || myAction == otherAction

	return resourceMatch && actionMatch
}

// Identity represents an authenticated user in a provider-agnostic way.
type Identity struct {
	// UserID is the unique identifier for the user from the auth provider.
	UserID string `json:"user_id"`

	// Email is the user's email address.
	Email string `json:"email"`

	// EmailVerified indicates whether the email has been verified.
	EmailVerified bool `json:"email_verified"`

	// OrganizationID is the auth provider's organization/tenant identifier.
	// This is a string UUID from the provider, NOT the database int32 ID.
	// Use RequestContext.OrganizationID for the database ID.
	OrganizationID string `json:"organization_id"`

	// Roles contains the user's role assignments (e.g., "admin", "member").
	Roles []Role `json:"roles"`

	// Permissions contains the derived permissions in "resource:action" format.
	Permissions []Permission `json:"permissions"`

	// ExpiresAt is when the token/session expires.
	ExpiresAt time.Time `json:"expires_at"`

	// Raw contains provider-specific data for debugging or advanced use cases.
	Raw map[string]any `json:"raw,omitempty"`
}

// HasRole checks if the identity has a specific role.
func (i *Identity) HasRole(role Role) bool {
	for _, r := range i.Roles {
		if r == role {
			return true
		}
	}
	return false
}

// HasPermission checks if the identity has a specific permission.
func (i *Identity) HasPermission(permission Permission) bool {
	for _, p := range i.Permissions {
		if p == permission {
			return true
		}
	}
	return false
}

// HasResourcePermission checks if the identity has permission for a resource and action.
func (i *Identity) HasResourcePermission(resource, action string) bool {
	return i.HasPermission(NewPermission(resource, action))
}

// RequestContext holds the resolved database IDs for the current request.
//
// This is set by the RequireOrganization middleware after looking up
// the organization and account in the database using the Identity's
// provider-specific IDs.
type RequestContext struct {
	// Identity contains the authenticated user information from the auth provider.
	Identity *Identity `json:"identity"`

	// OrganizationID is the database primary key (int32) for the organization.
	OrganizationID int32 `json:"organization_id"`

	// AccountID is the database primary key (int32) for the user's account.
	AccountID int32 `json:"account_id"`

	// ProviderOrgID preserves the original provider organization ID for reference.
	ProviderOrgID string `json:"provider_org_id,omitempty"`
}

// SetIdentity stores the Identity in the Gin context.
func SetIdentity(c *gin.Context, identity *Identity) {
	c.Set(string(identityKey), identity)
}

// GetIdentity retrieves the Identity from the Gin context.
func GetIdentity(c *gin.Context) *Identity {
	if val, exists := c.Get(string(identityKey)); exists {
		if identity, ok := val.(*Identity); ok {
			return identity
		}
	}
	return nil
}

// MustGetIdentity retrieves the Identity from the Gin context, panicking if unset.
func MustGetIdentity(c *gin.Context) *Identity {
	identity := GetIdentity(c)
	if identity == nil {
		panic("authcontext: MustGetIdentity called without Identity in context - ensure RequireAuth middleware is applied")
	}
	return identity
}

// SetRequestContext stores the RequestContext in the Gin context.
func SetRequestContext(c *gin.Context, reqCtx *RequestContext) {
	c.Set(string(requestContextKey), reqCtx)
}

// GetRequestContext retrieves the RequestContext from the Gin context.
func GetRequestContext(c *gin.Context) *RequestContext {
	if val, exists := c.Get(string(requestContextKey)); exists {
		if reqCtx, ok := val.(*RequestContext); ok {
			return reqCtx
		}
	}
	return nil
}

// MustGetRequestContext retrieves the RequestContext from the Gin context, panicking if unset.
func MustGetRequestContext(c *gin.Context) *RequestContext {
	reqCtx := GetRequestContext(c)
	if reqCtx == nil {
		panic("authcontext: MustGetRequestContext called without RequestContext in context - ensure RequireOrganization middleware is applied")
	}
	return reqCtx
}

// GetOrganizationID is a convenience function to get the database organization ID.
func GetOrganizationID(c *gin.Context) int32 {
	if reqCtx := GetRequestContext(c); reqCtx != nil {
		return reqCtx.OrganizationID
	}
	return 0
}

// GetAccountID is a convenience function to get the database account ID.
func GetAccountID(c *gin.Context) int32 {
	if reqCtx := GetRequestContext(c); reqCtx != nil {
		return reqCtx.AccountID
	}
	return 0
}

// WithIdentity adds the Identity to a context.Context.
func WithIdentity(ctx context.Context, identity *Identity) context.Context {
	return context.WithValue(ctx, identityKey, identity)
}

// IdentityFromContext retrieves the Identity from a context.Context.
func IdentityFromContext(ctx context.Context) *Identity {
	if val := ctx.Value(identityKey); val != nil {
		if identity, ok := val.(*Identity); ok {
			return identity
		}
	}
	return nil
}

// WithRequestContext adds the RequestContext to a context.Context.
func WithRequestContext(ctx context.Context, reqCtx *RequestContext) context.Context {
	return context.WithValue(ctx, requestContextKey, reqCtx)
}

// RequestContextFromContext retrieves the RequestContext from a context.Context.
func RequestContextFromContext(ctx context.Context) *RequestContext {
	if val := ctx.Value(requestContextKey); val != nil {
		if reqCtx, ok := val.(*RequestContext); ok {
			return reqCtx
		}
	}
	return nil
}
