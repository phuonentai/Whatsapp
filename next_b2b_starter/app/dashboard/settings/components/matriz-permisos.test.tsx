import { describe, it, expect, vi, beforeEach } from "vitest";
import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { renderWithProviders } from "@/test/render";
import type { RbacRole } from "@/lib/api/api/repositories/rbac-repository";

vi.mock("@/lib/hooks/queries/use-rbac-roles-query", () => ({
  useRbacRolesQuery: () => ({
    data: mocks.roles(),
    isLoading: false,
    isError: false,
    isRefetching: false,
    refetch: vi.fn(),
  }),
}));

vi.mock("@/lib/hooks/use-entitlement", () => ({
  useModules: () => mocks.modules(),
  useEntitlementQuery: () => ({ data: mocks.entitlement() }),
}));

vi.mock("@/lib/hooks/queries/use-modules-queries", () => ({
  useModulesCatalogQuery: () => ({ data: [], isLoading: false }),
}));

const mocks = vi.hoisted(() => ({
  roles: vi.fn(),
  modules: vi.fn(),
  entitlement: vi.fn(),
}));

import { MatrizPermisos } from "./matriz-permisos";

const ADMIN_ROLE: RbacRole = {
  id: "admin",
  name: "Admin",
  description: "Control total.",
  typicalUsers: "",
  permissions: [
    {
      id: "contact:view",
      resource: "contact",
      action: "view",
      displayName: "contact view",
      description: "Can view contact",
      category: "General",
    },
    {
      id: "contact:create",
      resource: "contact",
      action: "create",
      displayName: "contact create",
      description: "Can create contact",
      category: "General",
    },
  ],
};

const MEMBER_ROLE: RbacRole = {
  id: "member",
  name: "Member",
  description: "Acceso básico.",
  typicalUsers: "",
  permissions: [
    {
      id: "contact:view",
      resource: "contact",
      action: "view",
      displayName: "contact view",
      description: "Can view contact",
      category: "General",
    },
  ],
};

describe("MatrizPermisos", () => {
  beforeEach(() => {
    mocks.roles.mockReset();
    mocks.modules.mockReset();
    mocks.entitlement.mockReset();
  });

  it("renders ✓ / parcial / — states with text (never color-only)", () => {
    renderWithProviders(
      <MatrizPermisos
        roles={[ADMIN_ROLE, MEMBER_ROLE]}
        isLoading={false}
        isError={false}
        onRetry={vi.fn()}
      />
    );

    // Row resource
    expect(screen.getByText("contact")).toBeInTheDocument();

    // Admin column: both permissions → ✓
    const checkmarks = screen.getAllByText("✓");
    expect(checkmarks.length).toBe(1); // admin: full access → ✓

    // Member: only view → partial
    expect(screen.getByText("parcial")).toBeInTheDocument();
  });

  it("renders 'política no disponible' with retry for an EMPTY role list (never 'sin permisos')", async () => {
    const onRetry = vi.fn();
    renderWithProviders(
      <MatrizPermisos roles={[]} isLoading={false} isError={false} onRetry={onRetry} />
    );

    expect(screen.getByText("Política no disponible")).toBeInTheDocument();
    // The empty state must offer a retry action.
    const retry = screen.getByRole("button", { name: "Reintentar" });
    await userEvent.click(retry);
    expect(onRetry).toHaveBeenCalled();
  });

  it("renders skeleton while loading", () => {
    renderWithProviders(
      <MatrizPermisos roles={[]} isLoading={true} isError={false} onRetry={vi.fn()} />
    );
    expect(screen.getByLabelText("Matriz de permisos")).toBeInTheDocument();
  });

  it("filters rows by resource", async () => {
    const dealRole: RbacRole = {
      ...ADMIN_ROLE,
      permissions: [
        {
          id: "deal:view",
          resource: "deal",
          action: "view",
          displayName: "deal view",
          description: "Can view deal",
          category: "General",
        },
      ],
    };
    renderWithProviders(
      <MatrizPermisos
        roles={[dealRole]}
        isLoading={false}
        isError={false}
        onRetry={vi.fn()}
      />
    );

    expect(screen.getByText("deal")).toBeInTheDocument();

    await userEvent.type(screen.getByLabelText("Filtrar la matriz por recurso"), "contact");
    await waitFor(() => {
      expect(screen.queryByText("deal")).not.toBeInTheDocument();
    });
  });
});
