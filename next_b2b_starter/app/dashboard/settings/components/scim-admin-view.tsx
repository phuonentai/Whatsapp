"use client";

import { AdminPortalSCIM } from "@stytch/nextjs/b2b/adminPortal";

import { usePermissions } from "@/lib/hooks/use-permissions";
import { PERMISSIONS } from "@/lib/auth/permissions";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { ui } from "@/lib/copy/ui";

/**
 * Settings `?view=scim`: Stytch Admin Portal SCIM provisioning surface
 * (`scim-provisioning` capability).
 *
 * Renders the pre-built Admin Portal SCIM component (`AdminPortalSCIM` from
 * `@stytch/nextjs/b2b/adminPortal`, pinned `@stytch/nextjs@21.15.1` — verified
 * in task 3.2 to ship in the installed versions). Org admins create, view and
 * delete SCIM connections, see the per-connection SCIM base URL, and manage
 * group→role mapping entirely in the Stytch-hosted surface. SCIM bearer
 * tokens are shown only by Stytch and are never transmitted to or stored by
 * the platform's servers.
 *
 * Gated by `org:manage` here as defense in depth — the settings view
 * allowlist already blocks the route for non-admins.
 */
export function ScimAdminView() {
  const { hasPermission, isInitialized: permissionsReady } = usePermissions();
  const canManage = hasPermission(PERMISSIONS.ORG_MANAGE);

  if (permissionsReady && !canManage) {
    return (
      <Alert variant="destructive" className="border border-red-200 bg-red-50">
        <AlertTitle>Acceso restringido</AlertTitle>
        <AlertDescription>
          No tienes permisos para administrar el aprovisionamiento SCIM.
        </AlertDescription>
      </Alert>
    );
  }

  return (
    <div className="space-y-6">
      <div className="flex flex-col gap-1">
        <h2 className="text-xl font-semibold text-gray-900">SCIM</h2>
        <p className="text-sm text-gray-600">
          {ui.enterpriseScim.description}
        </p>
      </div>

      <div
        data-testid="scim-admin-portal"
        className="rounded-xl border border-gray-200 bg-white p-4"
      >
        <AdminPortalSCIM />
      </div>

      <p className="text-xs text-muted-foreground">
        {ui.enterpriseScim.managedByStytch}
      </p>
    </div>
  );
}
