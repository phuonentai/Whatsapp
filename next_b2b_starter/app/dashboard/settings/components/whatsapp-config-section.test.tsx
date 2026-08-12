import { describe, it, expect, vi, beforeEach } from "vitest";
import { screen, fireEvent } from "@testing-library/react";
import { renderWithProviders } from "@/test/render";

const mocks = vi.hoisted(() => ({
  configQuery: vi.fn(),
  toggle: vi.fn(),
  upsert: vi.fn(),
  metaConfig: vi.fn(),
  signupStatus: vi.fn(),
  exchange: vi.fn(),
}));

vi.mock("@/lib/hooks/queries/use-whatsapp-config-query", () => ({
  useWhatsAppConfigQuery: () => mocks.configQuery(),
}));

vi.mock("@/lib/hooks/mutations/use-toggle-whatsapp-config", () => ({
  useToggleWhatsAppConfig: () => ({ isPending: false, mutateAsync: mocks.toggle }),
}));

vi.mock("@/lib/hooks/mutations/use-upsert-whatsapp-config", () => ({
  useUpsertWhatsAppConfig: () => ({ isPending: false, mutateAsync: mocks.upsert }),
}));

vi.mock("@/lib/hooks/queries/use-whatsapp-signup-query", () => ({
  useWhatsAppSignupMetaConfig: () => ({ data: undefined, refetch: mocks.metaConfig }),
  useWhatsAppSignupStatus: () => ({ data: undefined, refetch: vi.fn() }),
}));

vi.mock("@/lib/hooks/mutations/use-whatsapp-signup-mutation", () => ({
  useWhatsAppSignupExchange: () => ({ isPending: false, mutateAsync: mocks.exchange }),
}));

import { WhatsAppConfigSection } from "./whatsapp-config-section";
import { POST_CONNECT_DISMISS_KEY } from "@/components/whatsapp/post-connect-steps";

function config(isActive: boolean) {
  return {
    id: 1,
    organizationId: 1,
    phoneNumberId: "123456789012345",
    businessPhone: "+573001234567",
    webhookSecret: "secret",
    verifyToken: "token",
    appId: "app_id",
    wabaId: "waba_id",
    apiVersion: "v21.0",
    graphApiUrl: "https://graph.facebook.com",
    isActive,
    createdAt: new Date("2026-01-01T00:00:00Z"),
    updatedAt: new Date("2026-01-01T00:00:00Z"),
  };
}

function mockConfigQuery(isActive: boolean) {
  mocks.configQuery.mockReturnValue({
    data: config(isActive),
    isLoading: false,
    error: null,
    refetch: vi.fn(),
  });
}

describe("WhatsAppConfigSection post-connect state", () => {
  beforeEach(() => {
    localStorage.clear();
    vi.clearAllMocks();
    mocks.configQuery.mockReturnValue({
      data: undefined,
      isLoading: false,
      error: null,
      refetch: vi.fn(),
    });
  });

  it("renders the post-connect next-steps card when the config is active", () => {
    mockConfigQuery(true);

    renderWithProviders(<WhatsAppConfigSection />);

    expect(screen.getByText("Siguientes pasos")).toBeDefined();
    expect(screen.getByText("Envía un mensaje de prueba")).toBeDefined();
  });

  it("does not render the post-connect card when the config is inactive", () => {
    mockConfigQuery(false);

    renderWithProviders(<WhatsAppConfigSection />);

    expect(screen.queryByText("Siguientes pasos")).toBeNull();
  });

  it("dismisses the post-connect card and persists the flag", () => {
    mockConfigQuery(true);
    const { unmount } = renderWithProviders(<WhatsAppConfigSection />);

    fireEvent.click(screen.getByLabelText("Descartar"));

    expect(screen.queryByText("Siguientes pasos")).toBeNull();
    expect(localStorage.getItem(POST_CONNECT_DISMISS_KEY)).toBe("true");

    unmount();
    renderWithProviders(<WhatsAppConfigSection />);
    expect(screen.queryByText("Siguientes pasos")).toBeNull();
  });

  it("clears the dismissal when the config deactivates so the card reappears on reactivation", () => {
    mockConfigQuery(true);
    const { unmount, rerender } = renderWithProviders(<WhatsAppConfigSection />);

    fireEvent.click(screen.getByLabelText("Descartar"));
    expect(localStorage.getItem(POST_CONNECT_DISMISS_KEY)).toBe("true");

    // Deactivate: the stored dismissal is cleared and the card stays hidden.
    mockConfigQuery(false);
    rerender(<WhatsAppConfigSection />);
    expect(localStorage.getItem(POST_CONNECT_DISMISS_KEY)).toBeNull();
    expect(screen.queryByText("Siguientes pasos")).toBeNull();

    // Reactivate: the card shows again without any re-dismissal.
    mockConfigQuery(true);
    rerender(<WhatsAppConfigSection />);
    expect(screen.getByText("Siguientes pasos")).toBeDefined();
    unmount();
  });

  it("never invokes an API call while rendering or dismissing the post-connect flow", () => {
    const fetchSpy = vi.spyOn(globalThis, "fetch");
    mockConfigQuery(true);

    const { unmount } = renderWithProviders(<WhatsAppConfigSection />);
    fireEvent.click(screen.getByLabelText("Descartar"));

    expect(fetchSpy).not.toHaveBeenCalled();
    unmount();
  });
});
