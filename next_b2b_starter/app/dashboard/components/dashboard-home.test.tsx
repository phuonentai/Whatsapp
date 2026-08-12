import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { DashboardHome } from "./dashboard-home";
import { renderWithProviders } from "@/test/render";

vi.mock("@/lib/hooks/queries/use-conversations-query", () => ({
  useConversationsQuery: vi.fn(),
}));

vi.mock("@/lib/hooks/queries/use-members-query", () => ({
  useMembersQuery: vi.fn(),
}));

vi.mock("@/lib/hooks/queries/use-agent-settings-query", () => ({
  useAgentSettingsQuery: vi.fn(),
}));

vi.mock("@/lib/hooks/queries/use-whatsapp-config-query", () => ({
  useWhatsAppConfigQuery: vi.fn(),
}));

vi.mock("@/lib/hooks/queries/use-subscription-query", () => ({
  useSubscriptionQuery: vi.fn(),
}));

import { useConversationsQuery } from "@/lib/hooks/queries/use-conversations-query";
import { useMembersQuery } from "@/lib/hooks/queries/use-members-query";
import { useAgentSettingsQuery } from "@/lib/hooks/queries/use-agent-settings-query";
import { useWhatsAppConfigQuery } from "@/lib/hooks/queries/use-whatsapp-config-query";
import { useSubscriptionQuery } from "@/lib/hooks/queries/use-subscription-query";

const mockConversations = vi.mocked(useConversationsQuery);
const mockMembers = vi.mocked(useMembersQuery);
const mockAgentSettings = vi.mocked(useAgentSettingsQuery);
const mockWhatsAppConfig = vi.mocked(useWhatsAppConfigQuery);
const mockSubscription = vi.mocked(useSubscriptionQuery);

function mockQueries() {
  mockConversations.mockReturnValue({
    data: [
      {
        id: 1,
        status: "active",
        channel: "whatsapp",
        contactId: 10,
        contactPhone: "+57",
        contactDisplayName: "Cliente Uno",
        createdAt: "2026-01-01T00:00:00Z",
        updatedAt: "2026-01-01T00:00:00Z",
        lastMessageAt: "2026-08-10T10:00:00Z",
      },
      {
        id: 2,
        status: "closed",
        channel: "whatsapp",
        contactId: 11,
        contactPhone: "+58",
        contactDisplayName: "Cliente Dos",
        createdAt: "2026-01-01T00:00:00Z",
        updatedAt: "2026-01-01T00:00:00Z",
        lastMessageAt: "2026-08-11T09:00:00Z",
      },
    ],
    isLoading: false,
    isError: false,
  } as never);

  mockMembers.mockReturnValue({
    data: { members: [], totalCount: 0, hasMore: false },
    isLoading: false,
    isError: false,
  } as never);

  mockAgentSettings.mockReturnValue({
    data: undefined,
    isLoading: false,
    isError: false,
  } as never);

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
}

describe("DashboardHome", () => {
  beforeEach(() => {
    localStorage.clear();
  });

  it("renders KPI cards from query data", () => {
    mockQueries();
    renderWithProviders(<DashboardHome />);

    expect(screen.getByText("Conversaciones activas")).toBeDefined();
    expect(screen.getByText("Ventas de la semana")).toBeDefined();
    expect(screen.getByText("Facturas emitidas")).toBeDefined();
    expect(screen.getByText("Tiempo respuesta IA")).toBeDefined();

    // 2 conversations, 1 closed → 1 open
    expect(screen.getAllByText("1").length).toBeGreaterThan(0);
    // KPIs sin fuente de datos → "—" (spec: never fabricate values)
    expect(screen.getAllByText("—").length).toBeGreaterThanOrEqual(2);
    // Sin comparación de periodo disponible → sin badge de delta inventado (los "%" solo existen en badges de delta)
    expect(screen.queryByText(/%$/)).toBeNull();
  });

  it("renders the recomposed panels with honest empty states", () => {
    mockQueries();
    renderWithProviders(<DashboardHome />);

    // Conversaciones recientes: snippet-level data (nombre + hora), sin cuerpos de mensaje
    expect(screen.getByText("Conversaciones recientes")).toBeDefined();
    expect(screen.getByText("Cliente Uno")).toBeDefined();
    expect(screen.getByText("Cliente Dos")).toBeDefined();

    // Rendimiento del equipo: sin permiso (ORG_MANAGE ausente en test) → estado vacío honesto, sin CTA
    expect(screen.getByText("Rendimiento del equipo")).toBeDefined();
    expect(screen.getByText("No hay datos de miembros disponibles para mostrar.")).toBeDefined();

    // Facturas Siigo: sin endpoint de lista → estado vacío honesto; CTA solo con invoice:view
    expect(screen.getByText("Facturas Siigo")).toBeDefined();
    expect(screen.getByText("Conecta Siigo para ver tus facturas aquí.")).toBeDefined();
    expect(screen.queryByText("Configurar Siigo")).toBeNull();

    // Banner Auto-Piloto: sin agent-settings confirmado → sugerencia estática, sin afirmar modo
    expect(screen.getByText("Auto-Piloto")).toBeDefined();
    expect(
      screen.getByText("Activa el Auto-Piloto para que el asistente te ayude a responder más rápido.")
    ).toBeDefined();
  });

  it("renders operational quick action links", () => {
    mockQueries();
    renderWithProviders(<DashboardHome />);

    // Broadcast → la superficie real de campañas (pestaña del CRM)
    expect(screen.getByText("Enviar broadcast").closest("a")).toHaveAttribute(
      "href",
      "/dashboard/crm?view=campa%C3%B1as"
    );
    expect(screen.getByText("Nueva factura").closest("a")).toHaveAttribute(
      "href",
      "/dashboard/settings?view=siigo"
    );
    expect(screen.getByText("Nuevo contacto").closest("a")).toHaveAttribute(
      "href",
      "/dashboard/crm"
    );
    expect(screen.getByText("Exportar").closest("a")).toHaveAttribute(
      "href",
      "/dashboard/reportes"
    );
  });

  it("renders the first-run checklist and assistant intro for incomplete orgs", () => {
    mockQueries();
    renderWithProviders(<DashboardHome />);

    expect(screen.getByText("Primeros pasos")).toBeDefined();
    expect(screen.getByText("Conecta WhatsApp")).toBeDefined();
    expect(screen.getByText("Elige un plan")).toBeDefined();
    expect(screen.getAllByText("Conoce a tu asistente").length).toBeGreaterThan(0);
    expect(screen.getByText("Abre tu bandeja de entrada")).toBeDefined();
  });

  it("folds the first-run checklist manually without breaking its state", async () => {
    const user = userEvent.setup();
    mockQueries();
    renderWithProviders(<DashboardHome />);

    const toggle = screen.getByRole("button", {
      name: "Mostrar u ocultar los pasos de inicio",
    });
    expect(toggle).toHaveAttribute("aria-expanded", "true");

    await user.click(toggle);

    expect(toggle).toHaveAttribute("aria-expanded", "false");
    expect(screen.queryByText("Conecta WhatsApp")).toBeNull();
    expect(localStorage.getItem("dashboard-home.onboarding-checklist-folded")).toBe("1");
  });
});
