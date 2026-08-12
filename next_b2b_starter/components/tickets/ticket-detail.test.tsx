import { describe, it, expect, vi, beforeEach } from "vitest";
import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { toast } from "sonner";
import { renderWithProviders } from "@/test/render";
import { ui } from "@/lib/copy/ui";

const mocks = vi.hoisted(() => ({
  list: vi.fn(),
  get: vi.fn(),
  create: vi.fn(),
  transition: vi.fn(),
  assign: vi.fn(),
  setPriority: vi.fn(),
  setTags: vi.fn(),
  addInternalNote: vi.fn(),
  aiTriage: vi.fn(),
}));

vi.mock("@/lib/api/api/repositories/ticket-repository", () => ({
  ticketRepository: {
    list: mocks.list,
    get: mocks.get,
    create: mocks.create,
    transition: mocks.transition,
    assign: mocks.assign,
    setPriority: mocks.setPriority,
    setTags: mocks.setTags,
    addInternalNote: mocks.addInternalNote,
    aiTriage: mocks.aiTriage,
  },
}));

vi.mock("@/lib/hooks/queries/use-modules-queries", () => ({
  useTicketQuery: () => ({
    data: {
      ticket: {
        id: 1,
        organization_id: 1,
        title: "Problema con factura",
        description: "No llegó la factura",
        status: "open",
        priority: "normal",
        tags: [],
        overdue: false,
        created_at: "2026-08-01T00:00:00Z",
        updated_at: "2026-08-01T00:00:00Z",
      },
      eventos: [],
    },
    isLoading: false,
    isError: false,
    refetch: vi.fn(),
    isRefetching: false,
  }),
}));

import { TicketDetail } from "./ticket-detail";

const NOTE_INPUT_PLACEHOLDER = "Nota interna (solo equipo)...";

describe("TicketDetail AI triage", () => {
  beforeEach(() => {
    Object.values(mocks).forEach((m) => m.mockReset());
  });

  it("clicking Redactar nota calls the triage mutation and fills the note draft", async () => {
    const user = userEvent.setup();
    mocks.aiTriage.mockResolvedValue({ note: "Borrador generado por la IA.", priority: "high" });

    renderWithProviders(<TicketDetail id={1} />);
    await user.click(screen.getByRole("button", { name: /Redactar nota/i }));

    expect(mocks.aiTriage).toHaveBeenCalledTimes(1);
    expect(mocks.aiTriage).toHaveBeenCalledWith(1);
    expect(await screen.findByDisplayValue("Borrador generado por la IA.")).toBeInTheDocument();
  });

  it("shows the priority suggestion chip and Apply calls SetPriority", async () => {
    const user = userEvent.setup();
    mocks.aiTriage.mockResolvedValue({ note: "Borrador", priority: "high" });
    mocks.setPriority.mockResolvedValue({});

    renderWithProviders(<TicketDetail id={1} />);
    await user.click(screen.getByRole("button", { name: /Redactar nota/i }));

    const chip = await screen.findByText(
      ui.tickets.triagePrioritySuggestion.replace("{priority}", "alta")
    );
    expect(chip).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: ui.tickets.triageApply }));
    expect(mocks.setPriority).toHaveBeenCalledWith(1, "high");
    await waitFor(() =>
      expect(
        screen.queryByText(ui.tickets.triagePrioritySuggestion.replace("{priority}", "alta"))
      ).not.toBeInTheDocument()
    );
  });

  it("dismisses the suggestion chip without applying", async () => {
    const user = userEvent.setup();
    mocks.aiTriage.mockResolvedValue({ note: "Borrador", priority: "low" });

    renderWithProviders(<TicketDetail id={1} />);
    await user.click(screen.getByRole("button", { name: /Redactar nota/i }));
    await screen.findByText(ui.tickets.triagePrioritySuggestion.replace("{priority}", "baja"));

    await user.click(screen.getByRole("button", { name: "Descartar sugerencia de prioridad" }));
    expect(mocks.setPriority).not.toHaveBeenCalled();
    expect(
      screen.queryByText(ui.tickets.triagePrioritySuggestion.replace("{priority}", "baja"))
    ).not.toBeInTheDocument();
  });

  it("failure keeps the form values and shows a toast (credits exhausted)", async () => {
    const user = userEvent.setup();
    mocks.aiTriage.mockRejectedValue(new Error("API Error 402: ai_credits_exhausted"));

    renderWithProviders(<TicketDetail id={1} />);
    const input = screen.getByPlaceholderText(NOTE_INPUT_PLACEHOLDER);
    await user.type(input, "Nota manual del agente");

    await user.click(screen.getByRole("button", { name: /Redactar nota/i }));
    await waitFor(() =>
      expect(toast.error).toHaveBeenCalledWith(ui.tickets.triageCreditsExhausted)
    );
    expect(input).toHaveValue("Nota manual del agente");
  });

  it("failure keeps the form values and shows a toast (generic error)", async () => {
    const user = userEvent.setup();
    mocks.aiTriage.mockRejectedValue(new Error("network down"));

    renderWithProviders(<TicketDetail id={1} />);
    const input = screen.getByPlaceholderText(NOTE_INPUT_PLACEHOLDER);
    await user.type(input, "Nota manual del agente");

    await user.click(screen.getByRole("button", { name: /Redactar nota/i }));
    await waitFor(() => expect(toast.error).toHaveBeenCalledWith(ui.tickets.triageError));
    expect(input).toHaveValue("Nota manual del agente");
  });

  it("nothing is saved automatically after a successful triage", async () => {
    const user = userEvent.setup();
    mocks.aiTriage.mockResolvedValue({ note: "Borrador generado.", priority: null });

    renderWithProviders(<TicketDetail id={1} />);
    await user.click(screen.getByRole("button", { name: /Redactar nota/i }));
    await screen.findByDisplayValue("Borrador generado.");

    expect(mocks.addInternalNote).not.toHaveBeenCalled();
    expect(mocks.setPriority).not.toHaveBeenCalled();
    expect(mocks.transition).not.toHaveBeenCalled();
    // No priority suggestion when the model returns an invalid/missing one.
    expect(screen.queryByText(/Sugerencia de prioridad/i)).not.toBeInTheDocument();
  });

  it("disables the action and shows a loading state while in flight", async () => {
    const user = userEvent.setup();
    const { promise, resolve } = Promise.withResolvers<{ note: string; priority: null }>();
    mocks.aiTriage.mockReturnValue(promise);

    renderWithProviders(<TicketDetail id={1} />);
    await user.click(screen.getByRole("button", { name: /Redactar nota/i }));

    const button = screen.getByRole("button", { name: "Generando…" });
    expect(button).toBeDisabled();

    resolve({ note: "Listo", priority: null });
    await waitFor(() => expect(screen.getByDisplayValue("Listo")).toBeInTheDocument());
  });
});
