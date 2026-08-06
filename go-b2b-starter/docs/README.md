# Go B2B SaaS Starter Kit

A production-ready Go backend for B2B SaaS applications with multi-tenant architecture, authentication, billing, and file management.

## Quick Start

```bash
make run-deps      # Start PostgreSQL & Redis
make migrateup     # Run migrations
make dev           # Start dev server with hot reload
```

## Documentation

### Core Systems
- **[Architecture](./architecture.md)** - Clean Architecture, dependency injection, module patterns
- **[Database](./database.md)** - SQLC workflow, migrations, store adapters
- **[Authentication](./authentication.md)** - Stytch integration, RBAC, middleware
- **[Billing](./billing.md)** - Polar.sh integration, subscriptions, paywall

### Infrastructure
- **[File Manager](./file-manager.md)** - R2 storage and file operations
- **[Event Bus](./event-bus.md)** - Event-driven architecture patterns
- **[API Development](./api-development.md)** - Guide to building new endpoints

## Project Structure

The codebase follows Clean Architecture with three main layers:

**API Layer** (`internal/`) - HTTP handlers and routes
**Application Layer** (`internal/`) - Business logic organized by modules
**Shared Layer** (`internal/`) - Reusable infrastructure packages

Each application module contains:
- `domain/` - Entities, interfaces, business rules
- `app/` - Services (use cases)
- `infra/` - Repository implementations
- `module.go` - Dependency injection setup

## Common Commands

```bash
# Development
make dev                # Run dev server with hot reload (Air)
make server             # Run server without hot reload
make build              # Build production binary

# Dependencies
make run-deps           # Start PostgreSQL & Redis in Docker
make stop-deps          # Stop and remove Docker containers

# Database
make migrateup          # Apply all migrations
make migratedown        # Rollback migrations
make sqlc               # Generate code from SQL
make create-migration   # Create new migration file

# Code Generation
make swagger            # Generate Swagger docs

# Testing
make test               # Run tests with coverage

# Utilities
make clear-rbac-cache   # Clear RBAC and JWKS caches from Redis
```

## Tech Stack

- **Language**: Go 1.25+
- **HTTP**: Gin framework
- **Database**: PostgreSQL with SQLC
- **Auth**: Stytch B2B
- **Payments**: Polar.sh
- **Storage**: Cloudflare R2
- **DI**: uber-go/dig

## Environment Setup

Copy `example.env` to `app.env` and configure:

```env
# Database
DATABASE_HOST=localhost
DATABASE_NAME=b2b_starter

# Authentication
STYTCH_PROJECT_ID=your-project-id
STYTCH_SECRET=your-secret

# Billing
POLAR_ACCESS_TOKEN=your-token
POLAR_WEBHOOK_SECRET=your-secret

# File Storage
R2_ACCOUNT_ID=your-account
R2_ACCESS_KEY_ID=your-key
R2_SECRET_ACCESS_KEY=your-secret
R2_BUCKET_NAME=files
```

See `example.env` for all configuration options.

## Getting Started

1. **Understand the architecture**: Read [Architecture](./architecture.md)
2. **Set up the database**: Follow [Database](./database.md)
3. **Configure authentication**: See [Authentication](./authentication.md)
4. **Build your first API**: Follow [API Development](./api-development.md)
