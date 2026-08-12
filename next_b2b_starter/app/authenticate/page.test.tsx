import { beforeEach, describe, expect, it, vi } from "vitest";
import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { renderWithProviders } from "@/test/render";

const mocks = vi.hoisted(() => ({
  push: vi.fn(),
  refresh: vi.fn(),
  searchParams: vi.fn(),
  consumeMagicLink: vi.fn(),
  authenticateTotp: vi.fn(),
}));

vi.mock("next/navigation", () => ({
  useRouter: () => ({ push: mocks.push, refresh: mocks.refresh }),
  useSearchParams: () => mocks.searchParams(),
  usePathname: () => "/authenticate",
}));

vi.mock("@/lib/actions/auth/consume-magic-link", () => ({
  consumeMagicLink: mocks.consumeMagicLink,
}));

vi.mock("@/lib/actions/auth/mfa", () => ({
  authenticateTotp: mocks.authenticateTotp,
}));

import AuthenticateRedirectPage from "./page";

const MFA_REQUIRED_RESULT = {
  success: true,
  data: {
    memberAuthenticated: false,
    intermediateSessionToken: "intermediate-1",
    mfaRequired: true,
    member: { member_id: "member-123", email_address: "m@example.com", name: "M" },
    organization: { organization_id: "org-456", organization_name: "Acme" },
  },
};

function paramsWith(token = "magic-token-1", returnTo = "/dashboard") {
  return new URLSearchParams({ stytch_token: token, returnTo });
}

beforeEach(() => {
  mocks.push.mockReset();
  mocks.refresh.mockReset();
  mocks.consumeMagicLink.mockReset();
  mocks.authenticateTotp.mockReset();
  mocks.searchParams.mockReturnValue(paramsWith());
});

describe("AuthenticateRedirectPage — MFA challenge step", () => {
  it("renders the TOTP challenge step instead of failing when MFA is required", async () => {
    mocks.consumeMagicLink.mockResolvedValue(MFA_REQUIRED_RESULT);

    renderWithProviders(<AuthenticateRedirectPage />);

    // Challenge step appears with code input + verify button.
    expect(
      await screen.findByPlaceholderText(/código de 6 dígitos/i)
    ).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: /verificar código/i })
    ).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: /usar un código de recuperación/i })
    ).toBeInTheDocument();
    // No redirect yet — session not minted.
    expect(mocks.push).not.toHaveBeenCalled();
  });

  it("shows a generic error on a wrong code and stays on the challenge step (no cookies, no redirect)", async () => {
    mocks.consumeMagicLink.mockResolvedValue(MFA_REQUIRED_RESULT);
    mocks.authenticateTotp.mockResolvedValue({
      success: false,
      error: "We couldn't verify that. Please try again later.",
    });

    const user = userEvent.setup();
    renderWithProviders(<AuthenticateRedirectPage />);

    const codeInput = await screen.findByPlaceholderText(/código de 6 dígitos/i);
    await user.type(codeInput, "000000");
    await user.click(screen.getByRole("button", { name: /verificar código/i }));

    // Generic error shown; user remains on the challenge step.
    expect(
      await screen.findByRole("alert")
    ).toHaveTextContent(/más tarde/i);
    expect(
      screen.getByRole("button", { name: /verificar código/i })
    ).toBeInTheDocument();
    expect(mocks.push).not.toHaveBeenCalled();
    expect(mocks.authenticateTotp).toHaveBeenCalledWith(
      expect.objectContaining({
        intermediateSessionToken: "intermediate-1",
        code: "000000",
        recoveryCode: undefined,
      })
    );
  });

  it("completes recovery-code sign-in in one exchange and redirects", async () => {
    mocks.consumeMagicLink.mockResolvedValue(MFA_REQUIRED_RESULT);
    mocks.authenticateTotp.mockResolvedValue({
      success: true,
      data: { memberAuthenticated: true },
    });

    const user = userEvent.setup();
    renderWithProviders(<AuthenticateRedirectPage />);

    await screen.findByPlaceholderText(/código de 6 dígitos/i);
    await user.click(
      screen.getByRole("button", { name: /usar un código de recuperación/i })
    );

    const recoveryInput = await screen.findByPlaceholderText(/código de recuperación/i);
    await user.type(recoveryInput, "abcd-efgh");
    await user.click(
      screen.getByRole("button", { name: /recuperar acceso/i })
    );

    await waitFor(() => expect(mocks.authenticateTotp).toHaveBeenCalledTimes(1));
    expect(mocks.authenticateTotp).toHaveBeenCalledWith({
      intermediateSessionToken: "intermediate-1",
      code: undefined,
      recoveryCode: "abcd-efgh",
      memberId: "member-123",
      organizationId: "org-456",
    });
    await waitFor(() => expect(mocks.push).toHaveBeenCalledWith("/dashboard"));
  });
});
