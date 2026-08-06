# Permission System Guide

This guide explains how to use the permission-driven UI system in the AP-Cash Frontend application.

## Overview

The permission system is built on top of Stytch B2B authentication and provides:
- Role-based access control (RBAC)
- Permission-based UI rendering
- Wildcard permission support (`resource:*`)
- Client and server-side guards

## Architecture

### 1. Stytch Integration
- Roles are stored in the Stytch member object at `member.roles[]`
- Each role is an object: `{ role_id: string, sources: [...] }`
- We extract `role_id` values (e.g., "member", "approver", "admin")
- Roles are mapped to permissions in our application
- Backend manages role definitions and assignments

### 2. Permission Flow
```
Stytch Session → Roles → Permissions → UI Components
```

## Files Structure

```
lib/auth/
├── permissions.ts          # Permission constants & role mappings
├── permission-utils.ts     # Permission check utilities
└── stytch/                 # Stytch integration

lib/hooks/
└── use-permissions.ts      # React hook for permissions

components/auth/
├── can.tsx                 # Inline permission wrapper
└── permission-gate.tsx     # Page-level permission guard

middleware.ts               # Route protection
```

## Usage Examples

### 1. Page-Level Protection

Protect entire pages using `PermissionGate`:

```tsx
// app/dashboard/approvals/page.tsx
import { PermissionGate } from "@/components/auth/permission-gate";
import { PERMISSIONS } from "@/lib/auth/permissions";

export default function ApprovalsPage() {
  return (
    <PermissionGate required={[PERMISSIONS.APPROVAL_VIEW]}>
      <ApprovalsPageContent />
    </PermissionGate>
  );
}
```

### 2. Conditional UI Rendering

Show/hide UI elements using the `Can` component:

```tsx
import { Can } from "@/components/auth/can";
import { PERMISSIONS } from "@/lib/auth/permissions";

function InvoiceActions() {
  return (
    <>
      {/* Single permission */}
      <Can permission={PERMISSIONS.INVOICE_CREATE}>
        <Button>Create Invoice</Button>
      </Can>

      {/* Multiple permissions (ANY - OR logic) */}
      <Can anyPermission={[PERMISSIONS.INVOICE_VIEW, PERMISSIONS.INVOICE_CREATE]}>
        <InvoiceList />
      </Can>

      {/* Multiple permissions (ALL - AND logic) */}
      <Can allPermissions={[PERMISSIONS.INVOICE_VIEW, PERMISSIONS.APPROVAL_APPROVE]}>
        <ComplexAction />
      </Can>

      {/* With fallback */}
      <Can
        permission={PERMISSIONS.INVOICE_DELETE}
        fallback={<DisabledButton />}
      >
        <DeleteButton />
      </Can>
    </>
  );
}
```

### 3. Using the Hook

Access permissions programmatically:

```tsx
import { usePermissions } from "@/lib/hooks/use-permissions";
import { PERMISSIONS } from "@/lib/auth/permissions";

function MyComponent() {
  const {
    hasPermission,
    hasAnyPermission,
    roles,
    permissions,
    isAuthenticated
  } = usePermissions();

  // Check single permission
  const canCreate = hasPermission(PERMISSIONS.INVOICE_CREATE);

  // Check multiple permissions
  const canViewOrEdit = hasAnyPermission([
    PERMISSIONS.INVOICE_VIEW,
    PERMISSIONS.INVOICE_CREATE
  ]);

  // Use in logic
  const handleAction = () => {
    if (!hasPermission(PERMISSIONS.APPROVAL_APPROVE)) {
      toast.error("You don't have permission to approve invoices");
      return;
    }
    // ... perform action
  };

  return <div>User has {permissions.length} permissions</div>;
}
```

### 4. Navigation Filtering

Filter navigation items by permissions:

```tsx
import { usePermissions } from "@/lib/hooks/use-permissions";
import { PERMISSIONS } from "@/lib/auth/permissions";

const navigation = [
  {
    name: "Invoices",
    href: "/dashboard/invoices",
    permission: PERMISSIONS.INVOICE_VIEW,
  },
  {
    name: "Approvals",
    href: "/dashboard/approvals",
    permission: PERMISSIONS.APPROVAL_VIEW,
  },
];

function Sidebar() {
  const { hasPermission } = usePermissions();

  const visibleNav = navigation.filter(item =>
    !item.permission || hasPermission(item.permission)
  );

  return (
    <nav>
      {visibleNav.map(item => (
        <Link key={item.href} href={item.href}>
          {item.name}
        </Link>
      ))}
    </nav>
  );
}
```

## Available Permissions

### Invoice Management
- `invoice:create` — Upload and create new invoice records
- `invoice:view` — View invoice details
- `invoice:delete` — Delete invoices from the system (admin only)

### Duplicate Handling
- `duplicate:view` — View duplicate detection results
- `duplicate:resolve` — Resolve duplicate flags (admin only)

### Approval Workflow
- `approval:view` — View pending approvals and history
- `approval:approve` — Approve or reject invoices in the workflow

### Payment Optimization
- `payment_optimization:schedule` — Schedule payment runs (admin only)
- `payment_optimization:export` — Export payment files (admin only)
- `payment_optimization:execute` — Execute or reschedule payments (admin only)

### Audit Trail
- `audit:view` — View audit log timeline and summaries

### Organization Management
- `org:view` — View organization settings and roster (admin only)
- `org:manage` — Manage organization settings and members (admin only)

## Role Permissions

### Member
- `invoice:create`
- `invoice:view`
- `duplicate:view`

### Approver
- All Member permissions
- `approval:view`
- `approval:approve`
- `duplicate:view`

### Admin
- All permissions listed in this guide, including organization management, payment optimization, and duplicate resolution

## Wildcard Permissions

The system supports wildcard permissions using `*`:

```tsx
// Grant all actions for a resource
const permissions = ['invoice:*'];

// This matches:
// - invoice:view
// - invoice:create
// - invoice:delete

// Check wildcard permission
hasPermission('invoice:create'); // true if user has 'invoice:*'
```

## Server-Side Protection

### Middleware (Route Protection)
The middleware automatically protects dashboard routes:

```typescript
// middleware.ts
// Automatically protects:
// - /dashboard/*
// - /settings
// - /metrics
// - /audit

// Redirects to /auth if no session found
```

### API Client (401 Handling)
The API client automatically handles 401 responses:

```typescript
// On 401:
// 1. Clears session cookies
// 2. Redirects to /auth?returnTo={currentPath}
```

## Best Practices

### 1. Always Use Constants
```tsx
// ✅ Good
import { PERMISSIONS } from "@/lib/auth/permissions";
<Can permission={PERMISSIONS.INVOICE_CREATE}>

// ❌ Bad
<Can permission="invoice:create">
```

### 2. Guard at Page Level
```tsx
// ✅ Good - Guard the entire page
export default function InvoicePage() {
  return (
    <PermissionGate required={[PERMISSIONS.INVOICE_VIEW]}>
      <InvoicePageContent />
    </PermissionGate>
  );
}

// ❌ Bad - Relying only on navigation filters
// Users can still access the URL directly
```

### 3. Show Helpful Fallbacks
```tsx
// ✅ Good
<Can
  permission={PERMISSIONS.INVOICE_DELETE}
  fallback={
    <Tooltip content="You don't have permission to delete">
      <Button disabled>Delete</Button>
    </Tooltip>
  }
>
  <Button>Delete</Button>
</Can>

// ❌ Bad - Hiding without explanation
<Can permission={PERMISSIONS.INVOICE_DELETE}>
  <Button>Delete</Button>
</Can>
```

### 4. Backend Validation
**Always validate permissions on the backend!**

Client-side permission checks are for UX only. The backend must enforce permissions.

## Troubleshooting

### Permissions Not Working
1. Check Stytch session is initialized: `isInitialized === true`
2. Verify roles exist: `console.log(member?.roles)` - should be array of `{ role_id, sources }`
3. Check role_id values: `console.log(member?.roles?.map(r => r.role_id))`
4. Check role mapping in `lib/auth/permissions.ts` matches your role_id values

### Navigation Items Not Showing
1. Ensure permission constant is imported correctly
2. Check `usePermissions` hook is available (client component)
3. Verify navigation filter logic in sidebar

### 401 Redirect Loop
1. Check middleware excludes public routes
2. Verify session cookies are being set correctly
3. Check API endpoints don't return 401 for valid sessions

## Adding New Permissions

1. Add to `PERMISSIONS` constant:
```typescript
// lib/auth/permissions.ts
export const PERMISSIONS = {
  // ... existing
  NEW_RESOURCE_VIEW: 'new_resource:view',
  NEW_RESOURCE_CREATE: 'new_resource:create',
} as const;
```

2. Update role mappings:
```typescript
export const ROLE_PERMISSIONS = {
  admin: [
    // ... existing admin permissions
    PERMISSIONS.NEW_RESOURCE_VIEW,
    PERMISSIONS.NEW_RESOURCE_CREATE,
  ],
  approver: [
    // ... approver permissions
    PERMISSIONS.NEW_RESOURCE_VIEW,
  ],
  member: [
    // ... member permissions
    PERMISSIONS.NEW_RESOURCE_VIEW,
  ],
};
```

3. Use in components:
```tsx
<Can permission={PERMISSIONS.NEW_RESOURCE_CREATE}>
  <CreateButton />
</Can>
```

## Testing Permissions

```tsx
// Test with different roles
function PermissionDebug() {
  const { roles, permissions, hasPermission } = usePermissions();

  return (
    <div>
      <h3>Current Roles: {roles.join(', ')}</h3>
      <h3>Permissions ({permissions.length}):</h3>
      <ul>
        {permissions.map(p => <li key={p}>{p}</li>)}
      </ul>
      <h3>Permission Checks:</h3>
      <p>Can create invoice: {hasPermission(PERMISSIONS.INVOICE_CREATE) ? '✅' : '❌'}</p>
      <p>Can approve: {hasPermission(PERMISSIONS.APPROVAL_APPROVE) ? '✅' : '❌'}</p>
    </div>
  );
}
```
