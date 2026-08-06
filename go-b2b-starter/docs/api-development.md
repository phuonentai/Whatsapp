# API Development Guide

Step-by-step guide to building new API endpoints following Clean Architecture patterns.

## Overview

Building an API endpoint involves these layers:

1. **Domain** - Entity and repository interface
2. **Infrastructure** - Repository implementation
3. **Application** - Service with business logic
4. **API** - HTTP handler and routes

## Step 1: Database Layer

### Create Migration

Add migration files in `internal/db/postgres/sqlc/migrations/`:

```sql
-- 000015_create_resources.up.sql
CREATE TABLE app.resources (
    id SERIAL PRIMARY KEY,
    organization_id INT NOT NULL,
    name VARCHAR(255) NOT NULL,
    status VARCHAR(50) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_resources_org ON app.resources(organization_id);
```

### Write SQL Queries

In `internal/db/postgres/sqlc/query/resources.sql`:

```sql
-- name: GetResourceByID :one
SELECT * FROM app.resources WHERE id = $1;

-- name: CreateResource :one
INSERT INTO app.resources (organization_id, name, status)
VALUES ($1, $2, $3)
RETURNING *;

-- name: ListResources :many
SELECT * FROM app.resources
WHERE organization_id = $1
ORDER BY created_at DESC;
```

### Generate Code

```bash
make sqlc
```

### Create Store Interface

In `internal/db/adapters/resource_store.go`:

```go
type ResourceStore interface {
    GetResourceByID(ctx context.Context, id int32) (sqlc.Resource, error)
    CreateResource(ctx context.Context, arg sqlc.CreateResourceParams) (sqlc.Resource, error)
    ListResources(ctx context.Context, orgID int32) ([]sqlc.Resource, error)
}
```

### Implement Adapter

In `internal/db/postgres/adapter_impl/resource_store.go`:

```go
type resourceStore struct {
    store sqlc.Store
}

func NewResourceStore(store sqlc.Store) adapters.ResourceStore {
    return &resourceStore{store: store}
}

func (s *resourceStore) GetResourceByID(ctx context.Context, id int32) (sqlc.Resource, error) {
    return s.store.GetResourceByID(ctx, id)
}
```

### Register in DI

In `internal/db/inject.go`:

```go
container.Provide(func(sqlcStore sqlc.Store) adapters.ResourceStore {
    return adapter_impl.NewResourceStore(sqlcStore)
})
```

## Step 2: Domain Layer

### Create Entity

In `internal/resources/domain/entity.go`:

```go
type Resource struct {
    ID             int32
    OrganizationID int32
    Name           string
    Status         string
    CreatedAt      time.Time
    UpdatedAt      time.Time
}

func (r *Resource) Validate() error {
    if r.Name == "" {
        return ErrResourceNameRequired
    }
    return nil
}
```

### Define Repository Interface

In `internal/resources/domain/repository.go`:

```go
type ResourceRepository interface {
    Create(ctx context.Context, resource *Resource) (*Resource, error)
    GetByID(ctx context.Context, id int32) (*Resource, error)
    List(ctx context.Context, orgID int32) ([]*Resource, error)
}
```

## Step 3: Infrastructure Layer

### Implement Repository

In `internal/resources/infra/repositories/resource_repository.go`:

```go
type resourceRepository struct {
    store adapters.ResourceStore
}

func NewResourceRepository(store adapters.ResourceStore) domain.ResourceRepository {
    return &resourceRepository{store: store}
}

func (r *resourceRepository) Create(ctx context.Context, resource *domain.Resource) (*domain.Resource, error) {
    params := sqlc.CreateResourceParams{
        OrganizationID: resource.OrganizationID,
        Name:           resource.Name,
        Status:         resource.Status,
    }

    dbResource, err := r.store.CreateResource(ctx, params)
    if err != nil {
        return nil, fmt.Errorf("failed to create resource: %w", err)
    }

    return toDomainResource(dbResource), nil
}
```

## Step 4: Application Layer

### Define Service Interface

In `internal/resources/app/services/resource_service_interface.go`:

```go
type ResourceService interface {
    CreateResource(ctx context.Context, orgID int32, req *CreateResourceRequest) (*domain.Resource, error)
    GetResource(ctx context.Context, id int32) (*domain.Resource, error)
    ListResources(ctx context.Context, orgID int32) ([]*domain.Resource, error)
}
```

### Implement Service

In `internal/resources/app/services/resource_service.go`:

```go
type resourceService struct {
    repo domain.ResourceRepository
}

func NewResourceService(repo domain.ResourceRepository) ResourceService {
    return &resourceService{repo: repo}
}

func (s *resourceService) CreateResource(
    ctx context.Context,
    orgID int32,
    req *CreateResourceRequest,
) (*domain.Resource, error) {
    // Validate request
    if err := req.Validate(); err != nil {
        return nil, err
    }

    // Create entity
    resource := &domain.Resource{
        OrganizationID: orgID,
        Name:           req.Name,
        Status:         "active",
    }

    // Persist
    return s.repo.Create(ctx, resource)
}
```

## Step 5: API Layer

### Create Handler

In `internal/resources/handler.go`:

```go
type Handler struct {
    service services.ResourceService
}

func NewHandler(service services.ResourceService) *Handler {
    return &Handler{service: service}
}

func (h *Handler) CreateResource(c *gin.Context) {
    // Get auth context
    reqCtx := auth.GetRequestContext(c)
    if reqCtx == nil {
        c.JSON(401, gin.H{"error": "unauthorized"})
        return
    }

    // Parse request
    var req services.CreateResourceRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(400, gin.H{"error": "invalid request"})
        return
    }

    // Call service
    resource, err := h.service.CreateResource(c.Request.Context(), reqCtx.OrganizationID, &req)
    if err != nil {
        c.JSON(500, gin.H{"error": "failed to create resource"})
        return
    }

    c.JSON(201, resource)
}
```

### Register Routes

In `internal/resources/routes.go`:

```go
type Routes struct {
    handler        *Handler
    authMiddleware *auth.Middleware
}

func NewRoutes(handler *Handler, authMiddleware *auth.Middleware) *Routes {
    return &Routes{handler: handler, authMiddleware: authMiddleware}
}

func (r *Routes) Register(router *gin.Engine) {
    apiGroup := router.Group("/api/resources")
    apiGroup.Use(r.authMiddleware.RequireAuth())
    apiGroup.Use(r.authMiddleware.RequireOrganization())
    {
        apiGroup.POST("",
            auth.RequirePermissionFunc("resource", "create"),
            r.handler.CreateResource)

        apiGroup.GET("/:id", r.handler.GetResource)
        apiGroup.GET("", r.handler.ListResources)
    }
}
```

## Step 6: Module Registration

### Create Module

In `internal/resources/module.go`:

```go
type Module struct {
    container *dig.Container
}

func NewModule(container *dig.Container) *Module {
    return &Module{container: container}
}

func (m *Module) RegisterDependencies() error {
    // Repository
    if err := m.container.Provide(func(store adapters.ResourceStore) domain.ResourceRepository {
        return repositories.NewResourceRepository(store)
    }); err != nil {
        return err
    }

    // Service
    if err := m.container.Provide(func(repo domain.ResourceRepository) services.ResourceService {
        return services.NewResourceService(repo)
    }); err != nil {
        return err
    }

    return nil
}
```

### Initialize Module

In `internal/resources/cmd/init.go`:

```go
func Init(container *dig.Container) error {
    module := NewModule(container)
    return module.RegisterDependencies()
}
```

### Register API

In `internal/resources/provider.go`:

```go
func RegisterDependencies(container *dig.Container) error {
    // Register handler
    if err := container.Provide(func(service services.ResourceService) *Handler {
        return NewHandler(service)
    }); err != nil {
        return err
    }

    // Register routes
    if err := container.Provide(func(
        handler *Handler,
        authMiddleware *auth.Middleware,
    ) *Routes {
        return NewRoutes(handler, authMiddleware)
    }); err != nil {
        return err
    }

    return nil
}
```

## Quick Reference

### File Structure

```
internal/resources/
├── domain/
│   ├── entity.go
│   ├── repository.go
│   └── errors.go
├── app/services/
│   ├── resource_service_interface.go
│   └── resource_service.go
├── infra/repositories/
│   └── resource_repository.go
├── cmd/init.go
└── module.go

internal/resources/
├── handler.go
├── routes.go
└── provider.go
```

### Common Response Codes

- `200` - Success
- `201` - Created
- `400` - Bad Request
- `401` - Unauthorized
- `403` - Forbidden
- `404` - Not Found
- `500` - Internal Server Error

## Next Steps

- **Add tests**: Unit tests for service, integration tests for repository
- **Add Swagger docs**: Document API with Swagger annotations
- **Add validation**: Request/response validation
- **Add events**: Publish domain events for cross-module communication
