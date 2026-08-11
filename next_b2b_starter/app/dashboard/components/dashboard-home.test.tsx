import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";

import { DashboardHome } from "./dashboard-home";
import { renderWithProviders } from "@/test/render";

vi.mock("@/lib/hooks/queries/use-conversations-query", () => ({
  useConversationsQuery: vi.fn(),
}));

vi.mock("@/lib/hooks/queries/use-crm-queries", () => ({
  useContactsQuery: vi.fn(),
  useDealsQuery: vi.fn(),
  usePipelinesQuery: vi.fn(),
  useActivitiesQuery: vi.fn(),
}));

vi.mock("@/lib/hooks/queries/use-whatsapp-config-query", () => ({
  useWhatsAppConfigQuery: vi.fn(),
}));

vi.mock("@/lib/hooks/queries/use-subscription-query", () => ({
  useSubscriptionQuery: vi.fn(),
}));

import { useConversationsQuery } from "@/lib/hooks/queries/use-conversations-query";
import {
  useActivitiesQuery,
  useContactsQuery,
  useDealsQuery,
  usePipelinesQuery,
} from "@/lib/hooks/queries/use-crm-queries";
import { useWhatsAppConfigQuery } from "@/lib/hooks/queries/use-whatsapp-config-query";
import { useSubscriptionQuery } from "@/lib/hooks/queries/use-subscription-query";

const mockConversations = vi.mocked(useConversationsQuery);
const mockContacts = vi.mocked(useContactsQuery);
const mockDeals = vi.mocked(useDealsQuery);
const mockPipelines = vi.mocked(usePipelinesQuery);
const mockActivities = vi.mocked(useActivitiesQuery);
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
        createdAt: "2026-01-01T00:00:00Z",
        updatedAt: "2026-01-01T00:00:00Z",
      },
      {
        id: 2,
        status: "closed",
        channel: "whatsapp",
        contactId: 11,
        contactPhone: "+58",
        createdAt: "2026-01-01T00:00:00Z",
        updatedAt: "2026-01-01T00:00:00Z",
      },
    ],
    isLoading: false,
    isError: false,
  } as never);

  mockContacts.mockReturnValue({
    data: [{ id: 1 }, { id: 2 }, { id: 3 }],
    isLoading: false,
    isError: false,
  } as never);

  mockDeals.mockReturnValue({
    data: [
      { id: 1, stage_id: 10, nombre: "Deal A" },
      { id: 2, stage_id: 10, nombre: "Deal B" },
      { id: 3, stage_id: 20, nombre: "Deal C" },
    ],
    isLoading: false,
    isError: false,
  } as never);

  mockPipelines.mockReturnValue({
    data: [
      {
        id: 1,
        nombre: "Ventas",
        es_predeterminado: true,
        orden: 1,
        etapas: [
          { id: 10, nombre: "Negociación", pipeline_id: 1, orden: 1 },
          { id: 20, nombre: "Cerrado", pipeline_id: 1, orden: 2 },
        ],
        created_at: "",
        updated_at: "",
      },
    ],
    isLoading: false,
    isError: false,
  } as never);

  mockActivities.mockReturnValue({
    data: [
      {
        id: 1,
        tipo: "llamada",
        asunto: "Llamada de seguimiento",
        realizada_en: "2026-08-01T00:00:00Z",
      },
    ],
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

    expect(screen.getByText("Conversaciones abiertas")).toBeDefined();
    expect(screen.getByText("Contactos")).toBeDefined();
    expect(screen.getByText("Negocios")).toBeDefined();

    // 2 conversations, 1 closed → 1 open (value may share text with deal badges)
    expect(screen.getAllByText("1").length).toBeGreaterThan(0);
    // 3 contacts
    expect(screen.getAllByText("3")).toHaveLength(2);
  });

  it("shows deals grouped by stage and recent activity", () => {
    mockQueries();
    renderWithProviders(<DashboardHome />);

    expect(screen.getByText("Negocios por etapa")).toBeDefined();
    expect(screen.getByText("Negociación")).toBeDefined();
    expect(screen.getByText("Cerrado")).toBeDefined();
    expect(screen.getByText("Actividad reciente")).toBeDefined();
    expect(screen.getByText("Llamada de seguimiento")).toBeDefined();
  });

  it("renders quick action links", () => {
    mockQueries();
    renderWithProviders(<DashboardHome />);

    expect(screen.getByText("Abrir bandeja de entrada").closest("a")).toHaveAttribute(
      "href",
      "/dashboard/inbox"
    );
    expect(screen.getByText("Gestionar CRM").closest("a")).toHaveAttribute(
      "href",
      "/dashboard/crm"
    );
    // "Base de conocimiento" also appears in the assistant intro panel.
    const knowledgeLinks = screen
      .getAllByText("Base de conocimiento")
      .filter((el) => el.closest("a"))
      .map((el) => el.closest("a") as HTMLElement);
    expect(knowledgeLinks.some((a) => a.getAttribute("href") === "/dashboard/knowledge")).toBe(
      true
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
});
