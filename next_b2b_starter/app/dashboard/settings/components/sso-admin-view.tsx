"use client";

import { AdminPortalSSO } from "@stytch/nextjs/b2b/adminPortal";

import { usePermissions } from "@/lib/hooks/use-permissions";
import { PERMISSIONS } from "@/lib/auth/permissions";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { ui } from "@/lib/copy/ui";

/**
 * Settings `?view=sso`: Stytch Admin Portal SSO management surface
 * (`enterprise-sso` capability).
 *
 * Renders the pre-built Admin Portal SSO component (`AdminPortalSSO` from
 * `@stytch/nextjs/b2b/adminPortal`, pinned `@stytch/nextjs@21.15.1` — verified
 * in task 3.2 to ship in the installed versions, so no version bump was
 * needed). Org admins create/edit/delete SAML and OIDC connections entirely
 * in the Stytch-hosted surface; connection secrets (certificates, client
 * secrets, issuer URLs) live only in Stytch and never transit or persist on
 * the platform's servers.
 *
 * Gated by `org:manage` here as defense in depth — the settings view
 * allowlist already blocks the route for non-admins.
 *
 * NOTE (task 3.2 / D8): the pinned Admin Portal mount options expose no
 * `locale`/strings override, so the Admin Portal renders in Stytch's default
 * locale. The login surface (`StytchB2B`) does support `locale: "es"`.
 */
export function SsoAdminView() {
  const { hasPermission, isInitialized: permissionsReady } = usePermissions();
  const canManage = hasPermission(PERMISSIONS.ORG_MANAGE);

  if (permissionsReady && !canManage) {
    return (
      <Alert variant="destructive" className="border border-red-200 bg-red-50">
        <AlertTitle>Acceso restringido</AlertTitle>
        <AlertDescription>
          No tienes permisos para administrar las conexiones SSO.
        </AlertDescription>
      </Alert>
    );
  }

  return (
    <div className="space-y-6">
      <div className="flex flex-col gap-1">
        <h2 className="text-xl font-semibold text-gray-900">SSO (SAML / OIDC)</h2>
        <p className="text-sm text-gray-600">
          {ui.enterpriseSso.description}
        </p>
      </div>

      <div
        data-testid="sso-admin-portal"
        className="rounded-xl border border-gray-200 bg-white p-4"
      >
        <AdminPortalSSO
          config={{
            testLoginRedirectURL: `/authenticate`,
            testSignupRedirectURL: `/authenticate`,
          }}
        />
      </div>

      <p className="text-xs text-muted-foreground">
        {ui.enterpriseSso.managedByStytch}
      </p>
    </div>
  );
}
