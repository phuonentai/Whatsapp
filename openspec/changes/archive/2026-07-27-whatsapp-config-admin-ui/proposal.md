## Why

The WhatsApp Cloud API webhook ingress and CRM storage are already implemented — organizations can receive and store inbound WhatsApp messages. However, there is no self-service way for organizations to configure their WhatsApp phone number, webhook secret, or verify token. Currently, the only way to set up a WhatsApp connection is a manual `INSERT` into the `whatsapp.whatsapp_configs` database table. This blocks customer onboarding and prevents the platform from being a true self-service B2B SaaS product.

## What Changes

- Add backend API endpoints for CRUD management of WhatsApp configuration records under auth middleware with `org:manage` permission gating
- Add a `List` method to the WhatsApp config repository (currently missing)
- Wire WhatsApp routes into `internal/api/provider.go` (currently unregistered, meaning even the existing webhook endpoints are unreachable in production)
- Fix the `GET /api/v1/webhooks/whatsapp` verification endpoint to validate the `hub.verify_token` against the stored config rather than accepting any non-empty token
- Add a frontend admin UI integrated into the workspace settings page at `/dashboard/settings?view=whatsapp`, following the existing view stack pattern (profile, members, subscription)
- Add a new `WhatsAppConfigSection` component with form fields for phone number ID, business phone, webhook secret, verify token, app ID, and an active/inactive toggle

## Capabilities

### New Capabilities

- `whatsapp-config-api`: Backend API endpoints for managing per-organization WhatsApp configuration — get, upsert, and toggle active status. Auth-gated via existing middleware chain (auth + org_context + subscription) with `org:manage` permission check.
- `whatsapp-config-frontend`: Frontend admin UI within workspace settings that displays current WhatsApp connection status and provides a form to configure a WhatsApp Business phone number. Follows the existing settings view stack pattern and repository-based API client communication.

### Modified Capabilities

<!-- No existing capabilities are modified. The webhook ingress and CRM storage behaviors remain unchanged. -->

## Impact

- **Go backend** (`internal/modules/whatsapp/`): New config service, extended handler, modified routes, modified repository interface
- **Go backend** (`internal/api/provider.go`): WhatsApp routes wired into API route registration
- **Go backend** (`internal/db/postgres/sqlc/query/whatsapp.sql`): New list query for paginated config lookup
- **Next.js frontend** (`app/dashboard/settings/`): New `SettingsView` entry, new component, new query/mutation hooks
- **Next.js frontend** (`lib/api/api/`): New repository, new DTO for WhatsApp config
