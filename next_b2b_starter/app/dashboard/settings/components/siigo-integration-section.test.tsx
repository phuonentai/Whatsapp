import { describe, it, expect, vi, beforeEach } from "vitest";
import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { renderWithProviders } from "@/test/render";

const mocks = vi.hoisted(() => ({
  status: vi.fn(),
  numeration: vi.fn(),
  previewData: undefined as any,
  preview: vi.fn(),
  connect: vi.fn(),
  assisted: vi.fn(),
  confirmNumeration: vi.fn(),
  importConfirm: vi.fn(),
  testInvoice: vi.fn(),
  activate: vi.fn(),
  pause: vi.fn(),
  resume: vi.fn(),
}));

vi.mock("@/lib/hooks/queries/use-siigo-queries", () => ({
  useSiigoStatusQuery: () => ({ data: mocks.status(), isLoading: false, error: null, refetch: vi.fn() }),
  useSiigoNumerationQuery: () => ({ data: mocks.numeration(), isLoading: false, refetch: vi.fn() }),
  useImportPreviewQuery: () => ({
    data: mocks.previewData,
    isFetching: false,
    refetch: vi.fn().mockImplementation(async () => {
      const data = mocks.preview();
      mocks.previewData = data;
      return { data };
    }),
  }),
  useAdminConnectionsQuery: () => ({ data: [], isLoading: false, error: null, refetch: vi.fn() }),
}));

vi.mock("@/lib/hooks/mutations/use-siigo-mutations", () => ({
  useSiigoConnect: () => ({
    mutate: mocks.connect,
    mutateAsync: mocks.connect,
    isPending: false,
    error: null,
  }),
  useRequestAssistedSetup: () => ({ mutate: mocks.assisted, isPending: false, error: null }),
  useConfirmNumeration: () => ({ mutate: mocks.confirmNumeration, isPending: false, error: null }),
  useImportConfirm: () => ({
    mutateAsync: mocks.importConfirm,
    isPending: false,
    error: null,
  }),
  useTestInvoice: () => ({
    mutateAsync: mocks.testInvoice,
    isPending: false,
    error: null,
  }),
  useActivateInvoicing: () => ({ mutate: mocks.activate, isPending: false, error: null }),
  usePauseInvoicing: () => ({ mutate: mocks.pause, isPending: false, error: null }),
  useResumeInvoicing: () => ({ mutate: mocks.resume, isPending: false, error: null }),
  useSiigoSync: () => ({ mutate: vi.fn(), isPending: false, error: null }),
  useAdminProvision: () => ({ mutate: vi.fn(), mutateAsync: vi.fn(), isPending: false, error: null }),
}));

import { SiigoIntegrationSection } from "./siigo-integration-section";

function status(status: string) {
  return { organizationId: 5, provider: "siigo", status, nit: "900123" };
}

describe("SiigoIntegrationSection", () => {
  beforeEach(() => {
    mocks.status.mockReset();
    mocks.numeration.mockReset();
    mocks.preview.mockReset();
    mocks.previewData = undefined;
    mocks.connect.mockReset();
    mocks.confirmNumeration.mockReset();
    mocks.importConfirm.mockReset();
    mocks.testInvoice.mockReset();
    mocks.activate.mockReset();
    mocks.pause.mockReset();
    mocks.resume.mockReset();
  });

  it("shows connect invitation and form when status is none", () => {
    mocks.status.mockReturnValue(status("none"));
    renderWithProviders(<SiigoIntegrationSection />);
    expect(screen.getByText("Conecta Siigo para facturar")).toBeInTheDocument();
    expect(screen.getByPlaceholderText("client_id")).toBeInTheDocument();
    expect(screen.getByPlaceholderText("900.123.456-7")).toBeInTheDocument();
  });

  it("submits connect with credentials", async () => {
    const user = userEvent.setup();
    mocks.status.mockReturnValue(status("none"));
    renderWithProviders(<SiigoIntegrationSection />);
    await user.type(screen.getByPlaceholderText("client_id"), "cid");
    await user.type(screen.getByPlaceholderText("client_secret"), "csec");
    await user.type(screen.getByPlaceholderText("900.123.456-7"), "9001234567");
    await user.click(screen.getByRole("button", { name: "Conectar Siigo" }));
    expect(mocks.connect).toHaveBeenCalledWith({
      client_id: "cid",
      client_secret: "csec",
      nit: "9001234567",
    });
  });

  it("shows assisted setup banner for awaiting_setup", () => {
    mocks.status.mockReturnValue(status("awaiting_setup"));
    renderWithProviders(<SiigoIntegrationSection />);
    expect(screen.getByText("Tu equipo está configurando tu facturación")).toBeInTheDocument();
  });

  it("shows numeration confirmation step when connected", async () => {
    const user = userEvent.setup();
    mocks.status.mockReturnValue(status("connected"));
    mocks.numeration.mockReturnValue({ mode: "auto" });
    renderWithProviders(<SiigoIntegrationSection />);
    expect(screen.getByText("Paso 2 — Confirma tu numeración DIAN")).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Confirmar numeración" }));
    expect(mocks.confirmNumeration).toHaveBeenCalled();
  });

  it("import step requires preview before confirm and shows counts", async () => {
    const user = userEvent.setup();
    mocks.status.mockReturnValue(status("numeracion_ok"));
    mocks.preview.mockReturnValue({
      total: 5,
      nuevos: 2,
      existentes: 1,
      duplicados: 1,
      sin_nit: 1,
      sin_nombre: 0,
      contactos: 0,
      sin_contacto: 0,
    });
    mocks.importConfirm.mockResolvedValue({
      total: 5,
      nuevos: 2,
      existentes: 1,
      duplicados: 1,
      sin_nit: 1,
      sin_nombre: 0,
      contactos: 0,
      sin_contacto: 0,
    });
    renderWithProviders(<SiigoIntegrationSection />);
    expect(screen.getByText("Paso 3 — Importa tus clientes")).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Ver vista previa" }));
    expect(await screen.findByText("5 clientes encontrados")).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Confirmar importación" }));
    expect(mocks.importConfirm).toHaveBeenCalled();
    expect(await screen.findByText(/Importación completada: 2 nuevos/)).toBeInTheDocument();
  });

  it("shows sandbox test and activation steps when sandbox_ok", () => {
    mocks.status.mockReturnValue(status("sandbox_ok"));
    renderWithProviders(<SiigoIntegrationSection />);
    expect(screen.getByText("Paso 4 — Prueba en sandbox")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Activar facturación" })).toBeInTheDocument();
  });

  it("shows live banner with pause kill-switch", async () => {
    const user = userEvent.setup();
    mocks.status.mockReturnValue(status("live"));
    renderWithProviders(<SiigoIntegrationSection />);
    expect(screen.getByText("Facturación activa")).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Pausar" }));
    expect(mocks.pause).toHaveBeenCalled();
  });

  it("shows paused banner with resume kill-switch", () => {
    mocks.status.mockReturnValue(status("paused"));
    renderWithProviders(<SiigoIntegrationSection />);
    expect(screen.getByText("Facturación pausada")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Reanudar" })).toBeInTheDocument();
  });

  it("shows explicit disabled notice for invoicing_disabled", () => {
    mocks.status.mockReturnValue(status("invoicing_disabled"));
    renderWithProviders(<SiigoIntegrationSection />);
    expect(
      screen.getByText(/Facturación desactivada — activa con Siigo/),
    ).toBeInTheDocument();
  });
});
