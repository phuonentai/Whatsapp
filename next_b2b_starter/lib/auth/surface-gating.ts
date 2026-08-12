/**
 * Shared surface-gating logic — single source of truth for which product
 * surfaces a member can reach, derived from the SAME gates the navigation uses
 * (sidebar + settings view allowlist).
 *
 * The "Equipo y permisos" impact preview ("¿qué ve este miembro?") consumes
 * `computeSurfaceAccess` so it can never diverge from the real application
 * navigation: changing a gate here changes both the sidebar and the preview.
 *
 * Note on settings view gates: the `?view=` allowlist lives in
 * `settings-content.tsx` (gate `org:manage` for `view=access`, `audit:view`
 * for the audit log). This module documents the surface→gate mapping for the
 * navigation surfaces; view-level gates stay in the settings allowlist.
 */

import { PERMISSIONS } from "./permissions";

export type SurfaceKey =
  | "inbox"
  | "contacts"
  | "aiCopilot"
  | "knowledge"
  | "invoices"
  | "payments"
  | "analytics"
  | "suppliers"
  | "schedules"
  | "settings";

export interface SurfaceGate {
  /** Single required permission (e.g. `org:manage`). */
  permission?: string;
  /** Any of the listed permissions grants access. */
  anyPermissions?: string[];
  /** Single entitlement feature key (module gate). */
  entitlement?: string;
  /** Any of the listed entitlement feature keys grants access. */
  anyEntitlements?: string[];
}

/** Canonical surface → gate mapping (mirrors components/layout/sidebar.tsx). */
export const SURFACE_GATES: Record<SurfaceKey, SurfaceGate> = {
  // Inbox: members with inbox:view can read + reply manually; admins keep
  // org:manage. The page hides admin controls for non-managers.
  inbox: { anyPermissions: [PERMISSIONS.INBOX_VIEW, PERMISSIONS.ORG_MANAGE] },
  // CRM entries: visible when at least one CRM feature is entitled.
  contacts: {
    anyEntitlements: [
      "crm_contacts_manage",
      "crm_companies",
      "crm_deals",
      "crm_activities",
      "crm_tags",
    ],
  },
  // AI Copilot lives in settings (view=ai), gated by org:manage.
  aiCopilot: { permission: PERMISSIONS.ORG_MANAGE },
  // Knowledge base: everyone can access (no gate).
  knowledge: {},
  // Invoicing lives in settings (view=siigo); section enforces org:manage.
  invoices: { permission: PERMISSIONS.ORG_MANAGE },
  // Billing section enforces org:manage.
  payments: { permission: PERMISSIONS.ORG_MANAGE },
  // Module-gated: the analytics module grants the analytics_module feature.
  analytics: { entitlement: "analytics_module" },
  // Procurement page enforces org:manage.
  suppliers: { permission: PERMISSIONS.ORG_MANAGE },
  // Schedules page: same gating as the procurement section.
  schedules: { permission: PERMISSIONS.ORG_MANAGE },
  // Settings: everyone can open the settings overview.
  settings: {},
};

export interface SurfaceEntitlement {
  /** Mirrors the entitlement payload `funcionalidades` map. */
  funcionalidades?: Record<string, boolean>;
}

function evaluateGate(
  gate: SurfaceGate,
  permissions: string[],
  entitlement?: SurfaceEntitlement | null
): boolean {
  if (gate.permission) {
    return permissions.includes(gate.permission);
  }
  if (gate.anyPermissions) {
    return gate.anyPermissions.some((p) => permissions.includes(p));
  }
  if (gate.entitlement) {
    return entitlement?.funcionalidades?.[gate.entitlement] === true;
  }
  if (gate.anyEntitlements) {
    return gate.anyEntitlements.some(
      (key) => entitlement?.funcionalidades?.[key] === true
    );
  }
  // No gate → always accessible.
  return true;
}

/**
 * Compute per-surface access for a set of permissions (typically the effective
 * permissions of a member's role) plus the entitlement payload.
 */
export function computeSurfaceAccess(
  permissions: string[],
  entitlement?: SurfaceEntitlement | null
): Record<SurfaceKey, boolean> {
  const result = {} as Record<SurfaceKey, boolean>;
  for (const key of Object.keys(SURFACE_GATES) as SurfaceKey[]) {
    result[key] = evaluateGate(SURFACE_GATES[key], permissions, entitlement);
  }
  return result;
}
