# Permissions & Roles

The app uses **Role-Based Access Control (RBAC)**.

- **Frontend**: Uses server-computed permissions.
- **Backend**: The source of truth (`go-b2b-starter/src/pkg/auth/rbac.go`).

## Checking Permissions

### In React Components

**File**: `lib/hooks/use-permissions.ts`

```typescript
import { usePermissions } from '@/lib/hooks/use-permissions';

export function EditButton() {
  const { hasPermission, hasRole } = usePermissions();

  // Check Permission
  if (hasPermission('resource:edit')) {
    return <button>Edit</button>;
  }
  
  // Check Role
  if (hasRole('admin')) {
    return <AdminBadge />;
  }
  
  return null;
}
```

### In API Routes

**File**: `lib/auth/server-permissions.ts`

```typescript
import { getServerPermissions } from '@/lib/auth/server-permissions';

export async function POST(req) {
  const session = await requireMemberSession();
  const permissions = await getServerPermissions(session);
  
  if (!permissions.canCreateResources) {
    return NextResponse.json({ error: 'Forbidden' }, { status: 403 });
  }
}
```

## Available Roles

Defined in `go-b2b-starter/src/pkg/auth/roles.go`.

- **Admin**: Full access (`*`).
- **Manager**: Can view, create, edit, delete, approve.
- **Member**: Can view, create.

## Permissions Format

- `resource:view`
- `resource:create`
- `resource:edit`
- `resource:delete`
- `resource:approve`
- `org:manage`

## Next Steps

👉 **Learn about**: [Payments & Billing](./04-payments-and-billing.md)
