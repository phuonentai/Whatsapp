# Authentication Guide

The authentication system uses Stytch B2B for identity management with JWT verification, RBAC, and multi-tenant organization context.

## Architecture

**Provider**: Stytch B2B handles user authentication and sessions
**Middleware**: Verifies JWTs and resolves organization/account context
**RBAC**: Role-based access control with permissions
**Resolvers**: Bridge auth provider IDs to database IDs

## JWT Verification

The system uses a two-tier verification strategy:

**1. Fast Path** - Verify JWT locally using cached public keys
**2. API Fallback** - Call Stytch API if local verification fails

This approach balances security with performance.

### Configuration

```env
STYTCH_PROJECT_ID=project-test-xxx
STYTCH_SECRET=secret-test-xxx
STYTCH_ENV=test  # or "live"
```

## Middleware

Three middleware functions protect routes:

### RequireAuth

Verifies JWT and extracts identity.

```go
router.Use(authMiddleware.RequireAuth())
```

**What it does:**
- Verifies JWT from `Authorization: Bearer {token}` header
- Extracts user identity (email, roles, permissions)
- Stores `auth.Identity` in request context
- Returns 401 if auth fails

### RequireOrganization

Resolves organization and account IDs from auth provider.

```go
router.Use(authMiddleware.RequireOrganization())
```

**What it does:**
- Gets organization ID from Stytch → resolves to database ID
- Gets user email → resolves to account ID
- Stores `auth.RequestContext` with IDs
- Returns 401 if resolution fails

**Note:** Always use after `RequireAuth()`.

### RequirePermission

Checks user has specific permission.

```go
router.POST("/resources",
    auth.RequirePermissionFunc("resource", "create"),
    handler.CreateResource)
```

**What it does:**
- Checks if user has permission (e.g., `"resource:create"`)
- Returns 403 if permission missing

**Note:** Use after `RequireOrganization()`.

## Using Context in Handlers

Access authentication info from request context:

```go
func (h *Handler) MyHandler(c *gin.Context) {
    // Get full context
    reqCtx := auth.GetRequestContext(c)
    orgID := reqCtx.OrganizationID    // int32
    accountID := reqCtx.AccountID      // int32
    email := reqCtx.Identity.Email     // string

    // Or use convenience functions
    orgID := auth.GetOrganizationID(c)
    accountID := auth.GetAccountID(c)
}
```

## RBAC System

### Roles

Defined in `internal/auth/roles.go`:

- `RoleAdmin` - Full system access
- `RoleManager` - Organization management
- `RoleMember` - Standard user access

### Permissions

Format: `"{resource}:{action}"`

**Common permissions:**
- `resource:view` - Read access
- `resource:create` - Create new items
- `resource:update` - Modify existing items
- `resource:delete` - Delete items
- `org:manage` - Organization administration

Defined in `internal/auth/permissions.go`.

### Permission Checks

```go
// In middleware (route-level)
router.POST("/resources",
    auth.RequirePermissionFunc("resource", "create"),
    handler.CreateResource)

// In code (programmatic)
if !auth.HasPermission(identity, "resource:delete") {
    return errors.New("permission denied")
}
```

## Resolver Pattern

Resolvers convert auth provider IDs to database IDs.

### Why Needed?

- Stytch uses string UUIDs for organizations
- Database uses int32 for primary keys
- Auth package can't depend on domain modules (circular dependency)

### How It Works

**1. Auth package defines interfaces:**

```go
type OrganizationResolver interface {
    ResolveByProviderID(ctx context.Context, providerID string) (int32, error)
}
```

**2. Domain modules implement via adapters:**

```go
type orgResolverAdapter struct {
    repo domain.OrganizationRepository
}

func (a *orgResolverAdapter) ResolveByProviderID(ctx context.Context, id string) (int32, error) {
    org, err := a.repo.GetByStytchID(ctx, id)
    if err != nil {
        return 0, err
    }
    return org.ID, nil
}
```

**3. Wired in initialization:**

Resolvers registered in `internal/bootstrap/init_mods.go` after organization module loads.

## Route Protection Patterns

### Public Route (No Auth)

```go
router.GET("/health", handler.Health)
```

### Authenticated Route

```go
apiGroup := router.Group("/api")
apiGroup.Use(authMiddleware.RequireAuth())
apiGroup.Use(authMiddleware.RequireOrganization())
{
    apiGroup.GET("/profile", handler.GetProfile)
}
```

### Permission-Protected Route

```go
apiGroup.POST("/resources",
    auth.RequirePermissionFunc("resource", "create"),
    handler.CreateResource)

apiGroup.DELETE("/resources/:id",
    auth.RequirePermissionFunc("resource", "delete"),
    handler.DeleteResource)
```

### Role-Protected Route

```go
adminGroup := router.Group("/admin")
adminGroup.Use(authMiddleware.RequireRole(auth.RoleAdmin))
{
    adminGroup.GET("/users", handler.ListUsers)
}
```

## Adding New Permissions

**1. Define permission constant** in `internal/auth/permissions.go`:

```go
const PermResourceView = Permission("resource:view")
const PermResourceCreate = Permission("resource:create")
```

**2. Assign to roles** in `internal/auth/rbac.go`:

```go
{
    RoleMember: {
        PermResourceView,
        // ... other permissions
    },
    RoleManager: {
        PermResourceView,
        PermResourceCreate,
        // ... other permissions
    },
}
```

**3. Protect routes**:

```go
router.POST("/resources",
    auth.RequirePermissionFunc("resource", "create"),
    handler.CreateResource)
```

## Common Patterns

### Check Organization Ownership

```go
func (h *Handler) GetResource(c *gin.Context) {
    orgID := auth.GetOrganizationID(c)
    resourceID := parseID(c.Param("id"))

    resource, err := h.service.GetResource(c.Request.Context(), resourceID)
    if err != nil {
        c.JSON(500, gin.H{"error": "failed to get resource"})
        return
    }

    // Verify resource belongs to user's organization
    if resource.OrganizationID != orgID {
        c.JSON(403, gin.H{"error": "access denied"})
        return
    }

    c.JSON(200, resource)
}
```

### Optional Authentication

```go
func (h *Handler) PublicResource(c *gin.Context) {
    // Try to get org ID (may be 0 if not authenticated)
    orgID := auth.GetOrganizationID(c)

    if orgID != 0 {
        // User is authenticated, show personalized data
    } else {
        // User is not authenticated, show public data
    }
}
```

## File Locations

| Component | Path |
|-----------|------|
| Auth provider interface | `internal/auth/auth.go` |
| Middleware | `internal/auth/middleware.go` |
| Context helpers | `internal/auth/context.go` |
| RBAC definitions | `internal/auth/rbac.go` |
| Roles | `internal/auth/roles.go` |
| Permissions | `internal/auth/permissions.go` |
| Resolvers | `internal/auth/resolvers.go` |
| Stytch adapter | `internal/auth/adapters/stytch/` |

## Next Steps

- **Database operations**: See [Database Guide](./database.md)
- **Building APIs**: See [API Development Guide](./api-development.md)
- **Stytch documentation**: https://stytch.com/docs/b2b
