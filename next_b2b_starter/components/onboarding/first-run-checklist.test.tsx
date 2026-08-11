import { describe, it, expect, vi, beforeEach } from "vitest";
import { screen, fireEvent } from "@testing-library/react";

import { FirstRunChecklist } from "./first-run-checklist";
import { renderWithProviders } from "@/test/render";
import { useUIStore } from "@/stores/ui-store";

vi.mock("@/lib/hooks/queries/use-whatsapp-config-query", () => ({
  useWhatsAppConfigQuery: vi.fn(),
}));

vi.mock("@/lib/hooks/queries/use-subscription-query", () => ({
  useSubscriptionQuery: vi.fn(),
}));

vi.mock("@/lib/hooks/queries/use-conversations-query", () => ({
  useConversationsQuery: vi.fn(),
}));

import { useWhatsAppConfigQuery } from "@/lib/hooks/queries/use-whatsapp-config-query";
import { useSubscriptionQuery } from "@/lib/hooks/queries/use-subscription-query";
import { useConversationsQuery } from "@/lib/hooks/queries/use-conversations-query";

const mockWhatsAppConfig = vi.mocked(useWhatsAppConfigQuery);
const mockSubscription = vi.mocked(useSubscriptionQuery);
const mockConversations = vi.mocked(useConversationsQuery);

function renderChecklist() {
  return renderWithProviders(<FirstRunChecklist />);
}

describe("FirstRunChecklist", () => {
  beforeEach(() => {
    localStorage.clear();
    useUIStore.getState().reset();
    mockWhatsAppConfig.mockReturnValue({
      data: undefined,
      isLoading: false,
      isError: false,
    } as never);
    mockSubscription.mockReturnValue({
      data: { isActive: false },
      isLoading: false,
      isError: false,
    } as never);
    mockConversations.mockReturnValue({
      data: [],
      isLoading: false,
      isError: false,
    } as never);
  });

  it("renders all steps as pending by default", () => {
    renderChecklist();

    expect(screen.getByText("Primeros pasos")).toBeDefined();
    expect(screen.getByText("Conecta WhatsApp")).toBeDefined();
    expect(screen.getByText("Elige un plan")).toBeDefined();
    expect(screen.getByText("Conoce a tu asistente")).toBeDefined();
    expect(screen.getByText("Abre tu bandeja de entrada")).toBeDefined();
    expect(screen.getAllByText("Pendiente")).toHaveLength(4);
  });

  it("marks the WhatsApp step done when connected", () => {
    mockWhatsAppConfig.mockReturnValue({
      data: { id: 1, isActive: true } as never,
      isLoading: false,
      isError: false,
    } as never);

    renderChecklist();

    const doneLabels = screen.getAllByText("Completado");
    expect(doneLabels).toHaveLength(1);
    expect(screen.getByText("Conecta WhatsApp").closest("li")?.textContent).toContain(
      "Completado"
    );
  });

  it("marks the plan step done when subscribed", () => {
    mockSubscription.mockReturnValue({
      data: { isActive: true } as never,
      isLoading: false,
      isError: false,
    } as never);

    renderChecklist();

    const doneLabels = screen.getAllByText("Completado");
    expect(doneLabels).toHaveLength(1);
    expect(screen.getByText("Elige un plan").closest("li")?.textContent).toContain(
      "Completado"
    );
  });

  it("links each pending step to its surface", () => {
    renderChecklist();

    expect(screen.getByText("Conecta WhatsApp").closest("a")).toHaveAttribute(
      "href",
      "/dashboard/settings?view=whatsapp"
    );
    expect(screen.getByText("Conoce a tu asistente").closest("a")).toHaveAttribute(
      "href",
      "/dashboard/settings?view=ai"
    );
    expect(screen.getByText("Abre tu bandeja de entrada").closest("a")).toHaveAttribute(
      "href",
      "/dashboard/inbox"
    );
  });

  it("opens the plans modal from the plan step", () => {
    renderChecklist();

    fireEvent.click(screen.getByText("Elige un plan"));
    expect(useUIStore.getState().isPlansModalOpen).toBe(true);
  });

  it("hides entirely when all steps complete", () => {
    mockWhatsAppConfig.mockReturnValue({
      data: { id: 1, isActive: true } as never,
      isLoading: false,
      isError: false,
    } as never);
    mockSubscription.mockReturnValue({
      data: { isActive: true } as never,
      isLoading: false,
      isError: false,
    } as never);
    mockConversations.mockReturnValue({
      data: [{ id: 1 }] as never,
      isLoading: false,
      isError: false,
    } as never);
    localStorage.setItem("ai-onboarding.assistant-intro-dismissed", "true");

    const { container } = renderChecklist();
    expect(container.firstChild).toBeNull();
    expect(screen.queryByText("Primeros pasos")).toBeNull();
  });
});
