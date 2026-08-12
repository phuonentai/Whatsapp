import { beforeEach, describe, expect, it, vi } from "vitest";
import { screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { renderWithProviders } from "@/test/render";
import type { AuthPolicyMirror } from "@/lib/api/api/repositories/organization-repository";

const mocks = vi.hoisted(() => ({
  getAuthPolicy: vi.fn(),
  updateAuthPolicy: vi.fn(),
  hasPermission: vi.fn(),
}));

vi.mock("@/lib/api/api/repositories/organization-repository", () => ({
  organizationRepository: {
    getAuthPolicy: mocks.getAuthPolicy,
    updateAuthPolicy: mocks.updateAuthPolicy,
  },
}));

vi.mock("@/lib/hooks/use-permissions", () => ({
  usePermissions: () => ({
    hasPermission: mocks.hasPermission,
    isInitialized: true,
    permissions: [],
    roles: [],
    profile: null,
    isAuthenticated: true,
    hasAnyPermission: () => false,
    hasAllPermissions: () => false,
    hasRole: () => false,
    hasAnyRole: () => false,
    hasAllRoles: () => false,
    updateAuthState: () => {},
  }),
}));

import { AuthPolicySection } from "./auth-policy-section";

const DEFAULT_MIRROR: AuthPolicyMirror = {
  email_jit_provisioning: "DISABLED",
  email_allowed_domains: [],
  auth_methods_restricted: false,
  allowed_auth_methods: [],
  sso_jit_provisioning: "DISABLED",
  sso_jit_provisioning_allowed_connections: [],
  sso_default_connection_id: "",
  sso_active_connection_ids: [],
};

beforeEach(() => {
  mocks.getAuthPolicy.mockReset();
  mocks.updateAuthPolicy.mockReset();
  mocks.hasPermission.mockReset();
  mocks.hasPermission.mockReturnValue(true);
  mocks.getAuthPolicy.mockResolvedValue(DEFAULT_MIRROR);
  mocks.updateAuthPolicy.mockResolvedValue(undefined);
});

describe("AuthPolicySection", () => {
  it("renders the disabled default: no domain join, no SSO-JIT toggle without connections", async () => {
    renderWithProviders(<AuthPolicySection />);

    await waitFor(() =>
      expect(screen.getByTestId("auth-policy-section")).toBeInTheDocument()
    );

    // Default off: domain switch unchecked, domain input hidden.
    const domainSwitch = screen.getByRole("switch", {
      name: "Permitir que compañeros con el dominio se unan automáticamente",
    });
    expect(domainSwitch).not.toBeChecked();
    expect(screen.queryByLabelText(/Dominios permitidos/)).not.toBeInTheDocument();

    // No active SSO connections → SSO-JIT toggle is NOT rendered.
    expect(
      screen.queryByRole("switch", {
        name: "Permitir aprovisionamiento automático por SSO",
      })
    ).not.toBeInTheDocument();

    // magic_link is always checked and disabled.
    expect(screen.getByRole("checkbox", { name: "magic_link (siempre permitido)" })).toBeChecked();
  });

  it("posts the domain-restricted save payload with enforced-list methods", async () => {
    const user = userEvent.setup();
    renderWithProviders(<AuthPolicySection />);

    await waitFor(() => expect(screen.getByTestId("auth-policy-section")).toBeInTheDocument());

    await user.click(
      screen.getByRole("switch", {
        name: "Permitir que compañeros con el dominio se unan automáticamente",
      })
    );
    const domainInput = screen.getByLabelText(/Dominios permitidos/);
    await user.type(domainInput, "acme.com");

    // First-write preservation: an org on ALL_ALLOWED keeps its full effective
    // method set (magic_link, email_otp, google_oauth, microsoft_oauth; no sso
    // without connections) — email_otp is already checked by default.
    expect(
      screen.getByRole("checkbox", { name: "email_otp" })
    ).toBeChecked();

    await user.click(screen.getByRole("button", { name: "Guardar política" }));

    await waitFor(() => expect(mocks.updateAuthPolicy).toHaveBeenCalledTimes(1));
    expect(mocks.updateAuthPolicy).toHaveBeenCalledWith({
      email_jit_provisioning: "DOMAIN_RESTRICTED",
      email_allowed_domains: ["acme.com"],
      allowed_auth_methods: ["magic_link", "email_otp", "google_oauth", "microsoft_oauth"],
      sso_jit_provisioning: "DISABLED",
      sso_jit_provisioning_allowed_connections: [],
      sso_default_connection_id: "",
    });
  });

  it("renders the SSO-JIT toggle and posts RESTRICTED + the org's connection ids", async () => {
    mocks.getAuthPolicy.mockResolvedValue({
      ...DEFAULT_MIRROR,
      sso_active_connection_ids: ["conn-1", "conn-2"],
    });

    const user = userEvent.setup();
    renderWithProviders(<AuthPolicySection />);

    const ssoSwitch = await screen.findByRole("switch", {
      name: "Permitir aprovisionamiento automático por SSO",
    });
    expect(ssoSwitch).toBeInTheDocument();
    expect(ssoSwitch).not.toBeChecked();

    await user.click(ssoSwitch);
    await user.click(screen.getByRole("button", { name: "Guardar política" }));

    await waitFor(() => expect(mocks.updateAuthPolicy).toHaveBeenCalledTimes(1));
    expect(mocks.updateAuthPolicy).toHaveBeenCalledWith(
      expect.objectContaining({
        sso_jit_provisioning: "CONNECTION_RESTRICTED",
        sso_jit_provisioning_allowed_connections: ["conn-1", "conn-2"],
      })
    );
  });

  it("shows the structured 503 read error (auth_policy_unavailable) with retry", async () => {
    mocks.getAuthPolicy.mockRejectedValueOnce(
      new Error("API Error 503: auth_policy_unavailable")
    );

    renderWithProviders(<AuthPolicySection />);

    const alert = await screen.findByRole("alert");
    expect(within(alert).getByText(/503/)).toBeInTheDocument();
    expect(within(alert).getByText(/auth_policy_unavailable/)).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Reintentar" })).toBeInTheDocument();
  });

  it("shows the structured 503 write error and does not optimistically change state", async () => {
    mocks.getAuthPolicy.mockResolvedValue({
      ...DEFAULT_MIRROR,
      email_jit_provisioning: "DOMAIN_RESTRICTED",
      email_allowed_domains: ["acme.com"],
    });
    mocks.updateAuthPolicy.mockRejectedValueOnce(
      new Error("API Error 503: auth_policy_update_unavailable")
    );

    const user = userEvent.setup();
    renderWithProviders(<AuthPolicySection />);

    await waitFor(() =>
      expect(
        screen.getByRole("switch", {
          name: "Permitir que compañeros con el dominio se unan automáticamente",
        })
      ).toBeChecked()
    );

    await user.click(screen.getByRole("button", { name: "Guardar política" }));

    const alert = await screen.findByRole("alert");
    expect(within(alert).getByText(/auth_policy_update_unavailable/)).toBeInTheDocument();

    // No optimistic state change: the mirror still reflects the saved policy
    // (the write failed, so the reload never happened and state stays on the
    // mirrored values).
    expect(
      screen.getByRole("switch", {
        name: "Permitir que compañeros con el dominio se unan automáticamente",
      })
    ).toBeChecked();
    expect(mocks.updateAuthPolicy).toHaveBeenCalledTimes(1);
  });

  it("hides the whole section for members without org:manage", async () => {
    mocks.hasPermission.mockReturnValue(false);

    renderWithProviders(<AuthPolicySection />);

    await waitFor(() =>
      expect(
        screen.getByText("No tienes permisos para gestionar la política de autenticación.")
      ).toBeInTheDocument()
    );
    expect(screen.queryByTestId("auth-policy-section")).not.toBeInTheDocument();
  });
});
