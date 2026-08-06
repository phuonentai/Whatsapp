# CRM Module Design

Status: Draft (exploration complete, not yet implemented)

## Overview

Transform the existing WhatsApp-focused CRM module into a modern, industry-standard CRM with contacts, companies, deals, pipelines, activities, and tags. Build on the existing architecture — evolve the Contact entity, keep Conversations/Messages as messaging infrastructure, and add the relationship-management layer on top.

```
┌──────────────────────────────────────────────────────────────────────┐
│                     MODERN CRM CORE MODEL                            │
├──────────────────────────────────────────────────────────────────────┤
│                                                                      │
│    ┌──────────┐         ┌──────────┐         ┌──────────┐           │
│    │ CONTACT  │────────▶│ COMPANY  │         │ PIPELINE │           │
│    │   (who)  │    *    │  (where) │         │ (config) │           │
│    └────┬─────┘         └────┬─────┘         └────┬─────┘           │
│         │                    │                    │                  │
│         │              ┌─────┴─────┐        ┌─────┴─────┐           │
│         │              │    DEAL   │───────▶│   STAGE   │           │
│         │              │  (what)   │        │ (kanban)  │           │
│         └──────────────┴─────┬─────┘        └───────────┘           │
│                              │                                       │
│                              ▼                                       │
│    ┌──────────────────────────────────────────────────┐             │
│    │                ACTIVITY TIMELINE                  │             │
│    │   notes │ calls │ emails │ meetings │ messages    │             │
│    │   tasks │ system events │ deal changes           │             │
│    └──────────────────────────────────────────────────┘             │
│                              │                                       │
│                              ▼                                       │
│    ┌──────────┐         ┌──────────┐                                │
│    │   TAG    │────────▶│ENTITY TAG│  (M2M on contacts/companies/   │
│    │ (label)  │    *    │ (M2M)    │   deals)                       │
│    └──────────┘         └──────────┘                                │
│                                                                      │
└──────────────────────────────────────────────────────────────────────┘
```

## What Exists vs What's New

### Existing (keep, evolve)

| Asset | Action | Notes |
|-------|--------|-------|
| `Contact` entity | Evolve | Add email, company_id, source, lead_status, job_title, assigned_to |
| `Conversation` entity | Unchanged | Messaging sessions (active/closed/archived) |
| `Message` entity | Unchanged | Channel-specific payloads (WhatsApp fields) |
| `CRMService` | Evolve | Add Activity creation when inbound message is processed |
| `MessageListener` | Unchanged | Event bus subscriber for `whatsapp.message.received` |
| Contact/Conversation/Message repos | Extended | New methods added, existing methods unchanged |
| SQLC queries (crm.sql) | Extended | New queries in separate file `crm_extended.sql` |
| Event bus | Unchanged | CRM publishes new domain events |
| RBAC framework | Extended | New permissions added to `AllPermissions` |
| Clean Architecture | Unchanged | New entities follow the same pattern |
| DI (uber-go/dig) | Extended | New providers are additive |

### New

| Asset | Notes |
|-------|-------|
| `Company` entity | CRM company (NOT tenant organization). Name, domain, industry, size, phone, website, address, notes, metadata, owner_account_id |
| `Deal` entity | Sales opportunities. Name, amount, currency, expected_close_date, status (open/won/lost/abandoned), probability, notes, metadata. FK to contact, company, pipeline, stage |
| `Pipeline` entity | Configurable per-tenant. Name, is_default, sort_order |
| `PipelineStage` entity | Within a pipeline. Name, sort_order, color, probability |
| `Activity` entity | Timeline entries. Type (note/call/email/meeting/task/whatsapp_message/system), subject, content, status, due_date. FK to contact, company, deal, conversation. performed_by (account_id) |
| `Tag` entity | Labels per tenant. Name, color |
| `EntityTag` entity | M2M junction. tag_id, entity_type (contact/company/deal), entity_id |
| 6 new repositories | CompanyRepo, DealRepo, PipelineRepo, ActivityRepo, TagRepo, EntityTagRepo |
| 5 new services | ContactService, CompanyService, DealService, PipelineService, ActivityService, TagService |
| ~25 HTTP endpoints | Full CRUD for contacts, companies, deals, pipelines, activities, tags |
| Frontend pages | `/dashboard/crm/` with contacts list, company detail, deal kanban, activity timeline, pipeline editor |
| New permissions | contact:view/manage/delete, deal:view/manage, pipeline:view/manage |

## Key Design Decisions

### 1. Contact Identity Model

Contact is phone-centric (`UNIQUE(org_id, phone_number)`). New fields are additive with defaults.

```
Before:                              After:
phone_number (identity key)          phone_number (unchanged, still identity)
display_name, avatar_url             email (NEW, nullable, partial unique index)
metadata (JSONB)                     company_id (NEW, FK to crm.companies)
is_blocked, last_message_at          source (NEW, 'whatsapp'|'manual'|'import'|'api')
                                     lead_status (NEW, 'new'|'contacted'|'qualified'|'unqualified'|'customer')
                                     job_title (NEW)
                                     assigned_to (NEW, FK to organizations.accounts)
```

- Existing rows default to: `source='whatsapp'`, `lead_status='new'`
- WhatsApp flow unchanged: still upserts by `phone_number`
- Email uniqueness via partial unique index: `WHERE email IS NOT NULL`

### 2. Activity vs Message — Two-Layer Model

Activity and Message serve different purposes. They are NOT unified.

```
Activity                       Message
────────                       ───────
CRM timeline entry             Channel-specific payload
Type-agnostic                  WhatsApp-specific fields
Links to ALL CRM entities      Links to conversation only

User creates a note:           Activity only (no Message)
WhatsApp message arrives:      Activity CREATED + Message CREATED
User logs a call:              Activity only
Future email integration:      Activity + email-specific table
```

An Activity has FK to `conversation_id` and `metadata {message_id: ...}` for the link back.

### 3. WhatsApp → CRM Bridge

When a WhatsApp message arrives via the event bus:

```
MessageListener
  │
  ▼
CRMService.ProcessInboundMessage()
  │
  ├── upsert Contact           ← UNCHANGED
  ├── upsert Conversation      ← UNCHANGED
  ├── insert Message           ← UNCHANGED
  │
  └── NEW: insert Activity({
        ContactID:      contact.ID,
        ConversationID: conv.ID,
        Type:           "whatsapp_message",
        Subject:        "WhatsApp message from " + phone,
        Content:        event.Content (truncated),
        PerformedAt:    event.WhatsAppTimestamp,
        Metadata:       {message_id: msg.ID, direction: "inbound"}
      })
      │
      └── publish "crm.activity.created" event
```

Activity creation failure is logged and does NOT block message processing.

### 4. Naming: Company ≠ Organization

The tenant table is `organizations.organizations`. CRM companies use the name `Company` to avoid catastrophic confusion.

```
organizations.organizations   ← Tenant (auth/billing scope)
crm.companies                 ← CRM record (sales/relationship scope)

Contact.company_id → crm.companies.id  (clear, obvious)
```

### 5. Pipeline Seeding

On first access for a tenant, auto-create a default pipeline:

```
Default Sales Pipeline:
  Stage          Order   Color     Probability
  Lead            1      #6B7280   10%
  Qualified       2      #3B82F6   25%
  Proposal        3      #8B5CF6   50%
  Negotiation     4      #F59E0B   75%
  Closed Won      5      #10B981   100%   (positive exit)
  Closed Lost     6      #EF4444   0%     (negative exit)
```

Tenants can add, edit, reorder, and delete their own pipelines and stages via API/UI.

### 6. Permissions

| Permission | Member | Manager | Admin |
|-----------|--------|---------|-------|
| `contact:view` | yes | yes | yes |
| `contact:manage` | yes | yes | yes |
| `contact:delete` | no | yes | yes |
| `deal:view` | yes | yes | yes |
| `deal:manage` | no | yes | yes |
| `pipeline:view` | yes | yes | yes |
| `pipeline:manage` | no | yes | yes |

Added to `AllPermissions` and `AllRoles` in `auth/rbac.go`.

## Database Schema

### Contact Evolution (ALTER)

```sql
ALTER TABLE crm.contacts
  ADD COLUMN email VARCHAR(255),
  ADD COLUMN company_id INTEGER REFERENCES crm.companies(id) ON DELETE SET NULL,
  ADD COLUMN source VARCHAR(50) DEFAULT 'whatsapp',
  ADD COLUMN lead_status VARCHAR(50) DEFAULT 'new',
  ADD COLUMN job_title VARCHAR(255),
  ADD COLUMN assigned_to INTEGER REFERENCES organizations.accounts(id) ON DELETE SET NULL;

CREATE UNIQUE INDEX idx_contacts_org_email
  ON crm.contacts(organization_id, email) WHERE email IS NOT NULL;

CREATE INDEX idx_contacts_company ON crm.contacts(company_id);
CREATE INDEX idx_contacts_source ON crm.contacts(organization_id, source);
CREATE INDEX idx_contacts_lead_status ON crm.contacts(organization_id, lead_status);
CREATE INDEX idx_contacts_assigned ON crm.contacts(assigned_to);
```

### New Tables

```sql
CREATE TABLE crm.companies (
    id SERIAL PRIMARY KEY,
    organization_id INTEGER NOT NULL REFERENCES organizations.organizations(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    domain VARCHAR(255),
    industry VARCHAR(100),
    size VARCHAR(50),
    phone VARCHAR(20),
    website VARCHAR(500),
    address TEXT,
    notes TEXT,
    metadata JSONB DEFAULT '{}',
    owner_account_id INTEGER REFERENCES organizations.accounts(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(organization_id, name)
);

CREATE TABLE crm.pipelines (
    id SERIAL PRIMARY KEY,
    organization_id INTEGER NOT NULL REFERENCES organizations.organizations(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    is_default BOOLEAN DEFAULT false,
    sort_order INTEGER DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE crm.pipeline_stages (
    id SERIAL PRIMARY KEY,
    pipeline_id INTEGER NOT NULL REFERENCES crm.pipelines(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    sort_order INTEGER DEFAULT 0,
    color VARCHAR(7),
    probability INTEGER,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE crm.deals (
    id SERIAL PRIMARY KEY,
    organization_id INTEGER NOT NULL REFERENCES organizations.organizations(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    contact_id INTEGER REFERENCES crm.contacts(id) ON DELETE SET NULL,
    company_id INTEGER REFERENCES crm.companies(id) ON DELETE SET NULL,
    pipeline_id INTEGER NOT NULL REFERENCES crm.pipelines(id) ON DELETE RESTRICT,
    stage_id INTEGER REFERENCES crm.pipeline_stages(id) ON DELETE SET NULL,
    amount DECIMAL(12,2),
    currency VARCHAR(3) DEFAULT 'USD',
    expected_close_date DATE,
    status VARCHAR(20) DEFAULT 'open' CHECK (status IN ('open', 'won', 'lost', 'abandoned')),
    probability INTEGER,
    notes TEXT,
    metadata JSONB DEFAULT '{}',
    assigned_to INTEGER REFERENCES organizations.accounts(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE crm.activities (
    id SERIAL PRIMARY KEY,
    organization_id INTEGER NOT NULL REFERENCES organizations.organizations(id) ON DELETE CASCADE,
    contact_id INTEGER REFERENCES crm.contacts(id) ON DELETE SET NULL,
    company_id INTEGER REFERENCES crm.companies(id) ON DELETE SET NULL,
    deal_id INTEGER REFERENCES crm.deals(id) ON DELETE SET NULL,
    conversation_id INTEGER REFERENCES crm.conversations(id) ON DELETE SET NULL,
    type VARCHAR(30) NOT NULL CHECK (type IN ('note', 'call', 'email', 'meeting', 'task', 'whatsapp_message', 'system')),
    subject VARCHAR(500),
    content TEXT,
    status VARCHAR(20),
    due_date TIMESTAMPTZ,
    performed_by INTEGER REFERENCES organizations.accounts(id) ON DELETE SET NULL,
    performed_at TIMESTAMPTZ DEFAULT NOW(),
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE crm.tags (
    id SERIAL PRIMARY KEY,
    organization_id INTEGER NOT NULL REFERENCES organizations.organizations(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    color VARCHAR(7),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(organization_id, name)
);

CREATE TABLE crm.entity_tags (
    id SERIAL PRIMARY KEY,
    tag_id INTEGER NOT NULL REFERENCES crm.tags(id) ON DELETE CASCADE,
    entity_type VARCHAR(20) NOT NULL CHECK (entity_type IN ('contact', 'company', 'deal')),
    entity_id INTEGER NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(tag_id, entity_type, entity_id)
);

-- Indexes
CREATE INDEX idx_companies_org ON crm.companies(organization_id);
CREATE INDEX idx_pipelines_org ON crm.pipelines(organization_id);
CREATE INDEX idx_pipeline_stages_pipeline ON crm.pipeline_stages(pipeline_id);
CREATE INDEX idx_deals_org ON crm.deals(organization_id);
CREATE INDEX idx_deals_pipeline ON crm.deals(pipeline_id);
CREATE INDEX idx_deals_stage ON crm.deals(stage_id);
CREATE INDEX idx_deals_contact ON crm.deals(contact_id);
CREATE INDEX idx_deals_company ON crm.deals(company_id);
CREATE INDEX idx_deals_status ON crm.deals(organization_id, status);
CREATE INDEX idx_activities_org ON crm.activities(organization_id);
CREATE INDEX idx_activities_contact ON crm.activities(contact_id);
CREATE INDEX idx_activities_company ON crm.activities(company_id);
CREATE INDEX idx_activities_deal ON crm.activities(deal_id);
CREATE INDEX idx_activities_type ON crm.activities(organization_id, type);
CREATE INDEX idx_activities_performed ON crm.activities(organization_id, performed_at DESC);
CREATE INDEX idx_tags_org ON crm.tags(organization_id);
CREATE INDEX idx_entity_tags_entity ON crm.entity_tags(entity_type, entity_id);

-- Triggers
CREATE TRIGGER trigger_companies_updated_at
    BEFORE UPDATE ON crm.companies FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER trigger_pipelines_updated_at
    BEFORE UPDATE ON crm.pipelines FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER trigger_pipeline_stages_updated_at
    BEFORE UPDATE ON crm.pipeline_stages FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER trigger_deals_updated_at
    BEFORE UPDATE ON crm.deals FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER trigger_activities_updated_at
    BEFORE UPDATE ON crm.activities FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER trigger_tags_updated_at
    BEFORE UPDATE ON crm.tags FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
```

## Module Structure

```
go-b2b-starter/internal/modules/crm/
├── domain/
│   ├── contact.go              → EVOLVE: add new fields
│   ├── conversation.go         → UNCHANGED
│   ├── message.go              → UNCHANGED
│   ├── company.go              → NEW
│   ├── deal.go                 → NEW
│   ├── pipeline.go             → NEW
│   ├── activity.go             → NEW
│   ├── tag.go                  → NEW
│   ├── repository.go           → EVOLVE: add new interfaces
│   ├── errors.go               → EVOLVE: add new sentinels
│   └── events/
│       ├── contact_created.go  → NEW
│       ├── deal_stage_changed.go → NEW
│       └── activity_created.go → NEW
├── app/
│   └── services/
│       ├── crm_service.go      → EVOLVE: add Activity creation
│       ├── message_listener.go → UNCHANGED
│       ├── contact_service.go  → NEW
│       ├── company_service.go  → NEW
│       ├── deal_service.go     → NEW
│       ├── pipeline_service.go → NEW
│       ├── activity_service.go → NEW
│       └── tag_service.go      → NEW
├── infra/
│   └── repositories/
│       ├── contact_repository.go      → EVOLVE: map new columns
│       ├── conversation_repository.go → UNCHANGED
│       ├── message_repository.go      → UNCHANGED
│       ├── company_repository.go      → NEW
│       ├── deal_repository.go         → NEW
│       ├── pipeline_repository.go     → NEW
│       ├── activity_repository.go     → NEW
│       └── tag_repository.go          → NEW
├── cmd/
│   └── init.go                 → EVOLVE: subscribe to new events
├── module.go                   → EVOLVE: provide new services
├── provider.go                 → EVOLVE: provide new handlers+routes
├── handler.go                  → NEW: all CRM HTTP handlers
├── routes.go                   → NEW: all CRM API routes
```

## API Routes

```
POST   /api/crm/contacts                           → CreateContact
GET    /api/crm/contacts                           → ListContacts (paginated + filters)
GET    /api/crm/contacts/:id                       → GetContact
PUT    /api/crm/contacts/:id                       → UpdateContact
DELETE /api/crm/contacts/:id                       → DeleteContact

POST   /api/crm/companies                          → CreateCompany
GET    /api/crm/companies                          → ListCompanies
GET    /api/crm/companies/:id                      → GetCompany
PUT    /api/crm/companies/:id                      → UpdateCompany
DELETE /api/crm/companies/:id                      → DeleteCompany

POST   /api/crm/deals                              → CreateDeal
GET    /api/crm/deals                              → ListDeals (filter by pipeline/stage/status)
GET    /api/crm/deals/:id                          → GetDeal
PUT    /api/crm/deals/:id                          → UpdateDeal
PUT    /api/crm/deals/:id/stage                    → MoveDealStage (stage transition)

GET    /api/crm/pipelines                          → ListPipelines (with stages)
POST   /api/crm/pipelines                          → CreatePipeline
PUT    /api/crm/pipelines/:id                      → UpdatePipeline
POST   /api/crm/pipelines/:id/stages               → CreateStage
PUT    /api/crm/pipelines/:id/stages/:stageId      → UpdateStage

GET    /api/crm/activities                         → ListActivities (filter by type/entity)
POST   /api/crm/activities                         → CreateActivity (note, call, log, task)
GET    /api/crm/contacts/:id/activities            → ListContactActivities
GET    /api/crm/deals/:id/activities               → ListDealActivities

GET    /api/crm/tags                               → ListTags
POST   /api/crm/tags                               → CreateTag
POST   /api/crm/:entityType/:entityId/tags         → TagEntity
DELETE /api/crm/:entityType/:entityId/tags/:tagId  → UntagEntity

GET    /api/crm/conversations                      → ListConversations
GET    /api/crm/conversations/:id                  → GetConversation
GET    /api/crm/conversations/:id/messages         → ListMessages
```

All routes require `auth` + `org_context` middleware. Write operations require appropriate permissions.

## Frontend Surface

```
next_b2b_starter/app/dashboard/crm/
├── layout.tsx              # CRM layout (auth guard)
├── page.tsx                # Redirects to ?view=contacts

Components:
├── contacts/
│   ├── contact-table.tsx   # Data table with search, sort, filter
│   └── contact-detail.tsx  # Profile + activity timeline + deals
├── companies/
│   ├── company-table.tsx   # Data table
│   └── company-detail.tsx  # Detail + associated contacts/deals
├── deals/
│   ├── deal-kanban.tsx     # Drag-drop board (pipeline columns)
│   ├── deal-card.tsx       # Card in kanban column
│   └── deal-detail.tsx     # Detail + activity timeline
├── pipelines/
│   └── pipeline-editor.tsx # Drag-reorder stages, edit metadata
├── activities/
│   ├── activity-timeline.tsx # Chronological feed
│   └── activity-form.tsx     # Create note/call/log/task
├── tags/
│   ├── tag-badge.tsx       # Chip display
│   └── tag-selector.tsx    # Multi-select picker

Data layer:
├── lib/api/api/dto/crm.dto.ts
├── lib/api/api/repositories/crm-repository.ts
├── lib/models/crm.model.ts
├── lib/hooks/queries/
│   ├── use-contacts-query.ts
│   ├── use-companies-query.ts
│   ├── use-deals-query.ts
│   ├── use-pipelines-query.ts
│   ├── use-activities-query.ts
│   └── use-tags-query.ts
└── lib/hooks/mutations/
    ├── use-create-contact.ts
    ├── use-update-deal.ts
    └── use-move-deal-stage.ts
```

## Implementation Strata

### Stratum 1: Database Foundation
- New migration with all tables + contact evolution
- SQLC queries for new tables in `crm_extended.sql`
- Run `make sqlc` to regenerate

### Stratum 2: Domain Layer
- New domain entities (Company, Deal, Pipeline, Activity, Tag)
- New repository interfaces
- New infra implementations
- DI wiring in `inject.go`
- Evolve Contact entity + Contact repository for new columns

### Stratum 3: Services
- ContactService, CompanyService, DealService, PipelineService, ActivityService, TagService
- Extend CRMService to auto-create Activity on inbound message
- Pipeline seeding logic

### Stratum 4: HTTP + Permissions
- Handler methods (~25 endpoints)
- Routes with auth + permission middleware
- Provider DI wiring
- Update `api/provider.go` to register CRM routes
- New permissions in `auth/rbac.go`

### Stratum 5: Frontend
- DTOs → Repository → Models → Query Hooks → Mutations
- CRM dashboard page (SPA pattern like settings)
- Contact list, company list, deal kanban, activity timeline
- Pipeline editor, tag management
- Sidebar entry with permission check

## Risk Assessment

| Risk | Severity | Mitigation |
|------|----------|------------|
| ALTER contact table breaks WhatsApp flow | Medium | New columns are nullable with defaults. Existing SQL unchanged. WhatsApp upsert by phone unchanged. |
| Activity creation fails in message handler | Low | ActivityRepo.Create failure is logged, not propagated. Message saving continues. |
| SQLC regenerate breaks existing gen code | Low | New queries in separate file `crm_extended.sql`. Existing `crm.sql` unchanged. |
| DI wiring conflicts | Low | New providers are additive. dig resolves by type. All new types are distinct. |
| Frontend bundle size | Low | CRM page is lazy-loaded. No new heavy dependencies (reuses shadcn/ui). |
| Email duplicate contact creation | Low | Partial unique index on `(org_id, email)` handles it at DB level. |

## Constraints

- **Multi-tenant**: All CRM data scoped by `organization_id` FK to `organizations.organizations(id)`.
- **No new dependencies**: Reuses existing SQLC, PostgreSQL, gin, dig, shadcn/ui, TanStack Query, react-hook-form, zod.
- **Backward compatibility**: Existing WhatsApp flow unchanged. MessageListener, CRMService.ProcessInboundMessage both continue to work.
- **Incremental**: Each stratum can be deployed independently. Stratum 1 has zero impact on running code.
