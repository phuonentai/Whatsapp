import { beforeEach, describe, expect, it, vi } from "vitest";
import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { renderWithProviders } from "@/test/render";

const mocks = vi.hoisted(() => ({
  createTotp: vi.fn(),
  verifyTotpEnrollment: vi.fn(),
  rotateRecoveryCodes: vi.fn(),
}));

vi.mock("@/lib/actions/auth/mfa", () => ({
  createTotp: mocks.createTotp,
  verifyTotpEnrollment: mocks.verifyTotpEnrollment,
  rotateRecoveryCodes: mocks.rotateRecoveryCodes,
}));

import { SecuritySection } from "./security-section";

const CREATED_RESULT = {
  success: true,
  data: {
    status: "created" as const,
    totpRegistrationId: "totp-new-1",
    qrCode: "data:image/png;base64,QRPLACEHOLDER",
    secret: "MANUAL-SECRET-ABCD",
    recoveryCodes: ["code-1111", "code-2222"],
  },
};

beforeEach(() => {
  mocks.createTotp.mockReset();
  mocks.verifyTotpEnrollment.mockReset();
  mocks.rotateRecoveryCodes.mockReset();
});

describe("SecuritySection — TOTP enrollment", () => {
  it("shows the Stytch-issued QR code and manual secret after createTotp (no QR library)", async () => {
    mocks.createTotp.mockResolvedValue(CREATED_RESULT);
    const user = userEvent.setup();

    renderWithProviders(<SecuritySection />);

    await user.click(
      screen.getByRole("button", { name: /configurar app de autenticación/i })
    );

    await waitFor(() => expect(mocks.createTotp).toHaveBeenCalledTimes(1));
    const qr = await screen.findByAltText(/código qr/i);
    expect(qr).toHaveAttribute("src", "data:image/png;base64,QRPLACEHOLDER");
    expect(screen.getByText("MANUAL-SECRET-ABCD")).toBeInTheDocument();
  });

  it("shows one-time recovery codes after a successful verify and confirms enrollment", async () => {
    mocks.createTotp.mockResolvedValue(CREATED_RESULT);
    mocks.verifyTotpEnrollment.mockResolvedValue({
      success: true,
      data: { totpRegistrationId: "totp-new-1" },
    });
    const user = userEvent.setup();

    renderWithProviders(<SecuritySection />);

    await user.click(
      screen.getByRole("button", { name: /configurar app de autenticación/i })
    );
    await screen.findByAltText(/código qr/i);

    await user.type(
      screen.getByPlaceholderText(/código de 6 dígitos/i),
      "123456"
    );
    await user.click(screen.getByRole("button", { name: /verificar e inscribir/i }));

    // Recovery codes shown exactly once.
    expect(await screen.findByText("code-1111")).toBeInTheDocument();
    expect(screen.getByText("code-2222")).toBeInTheDocument();
    expect(mocks.verifyTotpEnrollment).toHaveBeenCalledWith({ code: "123456" });

    await user.click(screen.getByRole("button", { name: /guardé estos códigos/i }));
    expect(
      await screen.findByText(/verificación en dos pasos activada/i)
    ).toBeInTheDocument();
  });

  it("surfaces an existing registration for management instead of creating a duplicate", async () => {
    mocks.createTotp.mockResolvedValue({
      success: true,
      data: { status: "existing", totpRegistrationId: "totp-existing-9" },
    });
    mocks.rotateRecoveryCodes.mockResolvedValue({
      success: true,
      data: { recoveryCodes: ["rotated-aaaa", "rotated-bbbb"] },
    });
    const user = userEvent.setup();

    renderWithProviders(<SecuritySection />);

    await user.click(
      screen.getByRole("button", { name: /configurar app de autenticación/i })
    );

    expect(
      await screen.findByText(/ya tienes una app de autenticación configurada/i)
    ).toBeInTheDocument();
    expect(screen.getByText("totp-existing-9")).toBeInTheDocument();

    await user.click(
      screen.getByRole("button", { name: /rotar códigos de recuperación/i })
    );
    expect(await screen.findByText("rotated-aaaa")).toBeInTheDocument();
  });
});
