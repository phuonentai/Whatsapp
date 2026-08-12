import { beforeEach, describe, expect, it, vi } from "vitest";
import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { renderWithProviders } from "@/test/render";
import type { UserProfile } from "@/lib/models/member.model";

const mocks = vi.hoisted(() => ({
  updateMfaPolicy: vi.fn(),
}));

vi.mock("@/lib/api/api/repositories/organization-repository", () => ({
  organizationRepository: { updateMfaPolicy: mocks.updateMfaPolicy },
}));

import { MfaPolicySection } from "./mfa-policy-section";

const PROFILE: UserProfile = {
  id: "member-1",
  email: "admin@example.com",
  role: "admin",
  organizationId: "org-456",
  organizationName: "Acme",
  organizationMfaPolicy: "OPTIONAL",
  organizationMfaMethods: "RESTRICTED",
  organizationAllowedMfaMethods: ["totp"],
};

beforeEach(() => {
  mocks.updateMfaPolicy.mockReset();
  mocks.updateMfaPolicy.mockResolvedValue(undefined);
});

describe("MfaPolicySection", () => {
  it("reflects the profile mirror (display-only) as the initial selection", () => {
    renderWithProviders(<MfaPolicySection profile={PROFILE} />);
    expect(screen.getByText("Política MFA de la organización")).toBeInTheDocument();
    // OPTIONAL is the active (default-styled) option.
    expect(screen.getByRole("button", { name: "Opcional" })).toHaveClass(
      "bg-slate-900"
    );
    expect(
      screen.getByRole("button", { name: "Obligatorio para todos" })
    ).not.toHaveClass("bg-slate-900");
  });

  it("posts the policy change to PUT /api/organizations/mfa-policy with totp allowlist", async () => {
    const user = userEvent.setup();
    renderWithProviders(<MfaPolicySection profile={PROFILE} />);

    await user.click(
      screen.getByRole("button", { name: "Obligatorio para todos" })
    );
    await user.click(screen.getByRole("button", { name: "Guardar política" }));

    await waitFor(() => expect(mocks.updateMfaPolicy).toHaveBeenCalledTimes(1));
    expect(mocks.updateMfaPolicy).toHaveBeenCalledWith({
      mfa_policy: "REQUIRED_FOR_ALL",
      mfa_methods: "RESTRICTED",
      allowed_mfa_methods: ["totp"],
    });
  });

  it("shows the 503 structured error when the backend reports auth-provider unavailability", async () => {
    mocks.updateMfaPolicy.mockRejectedValue(
      new Error(
        "API Error 503: The MFA policy service is temporarily unavailable. Your organization's policy was not changed."
      )
    );
    const user = userEvent.setup();
    renderWithProviders(<MfaPolicySection profile={PROFILE} />);

    await user.click(screen.getByRole("button", { name: "Guardar política" }));

    expect(
      await screen.findByRole("alert")
    ).toHaveTextContent(/temporalmente no disponible/i);
    expect(screen.getByRole("alert")).toHaveTextContent(/503/i);
  });
});
