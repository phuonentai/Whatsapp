import { describe, it, expect } from "vitest";
import {
  computeSurfaceAccess,
  SURFACE_GATES,
} from "./surface-gating";
import { PERMISSIONS } from "./permissions";

describe("computeSurfaceAccess", () => {
  it("derives surfaces from the SAME gates as the navigation", () => {
    // Admin-like permission set.
    const access = computeSurfaceAccess(
      [PERMISSIONS.ORG_MANAGE, "contact:view"],
      { funcionalidades: { analytics_module: true } }
    );

    expect(access.inbox).toBe(true); // org:manage
    expect(access.aiCopilot).toBe(true); // org:manage
    expect(access.invoices).toBe(true); // org:manage
    expect(access.payments).toBe(true); // org:manage
    expect(access.suppliers).toBe(true); // org:manage
    expect(access.schedules).toBe(true); // org:manage
    expect(access.analytics).toBe(true); // entitlement
    expect(access.knowledge).toBe(true); // no gate
    expect(access.settings).toBe(true); // no gate
  });

  it("denies gated surfaces without the permission/entitlement", () => {
    const access = computeSurfaceAccess([], {});

    expect(access.inbox).toBe(false);
    expect(access.aiCopilot).toBe(false);
    expect(access.invoices).toBe(false);
    expect(access.payments).toBe(false);
    expect(access.suppliers).toBe(false);
    expect(access.schedules).toBe(false);
    expect(access.analytics).toBe(false);
    expect(access.knowledge).toBe(true);
    expect(access.settings).toBe(true);
  });

  it("keeps sidebar and preview gates in sync via SURFACE_GATES", () => {
    expect(SURFACE_GATES.inbox.permission).toBe(PERMISSIONS.ORG_MANAGE);
    expect(SURFACE_GATES.aiCopilot.permission).toBe(PERMISSIONS.ORG_MANAGE);
    expect(SURFACE_GATES.analytics.entitlement).toBe("analytics_module");
    expect(SURFACE_GATES.contacts.anyEntitlements).toContain("crm_deals");
  });
});
