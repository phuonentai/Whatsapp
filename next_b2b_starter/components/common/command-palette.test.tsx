import { describe, it, expect, beforeEach, vi } from "vitest";
import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { CommandPalette } from "./command-palette";
import { renderWithProviders } from "@/test/render";
import { crmRepository } from "@/lib/api/api/repositories/crm-repository";
import { useCommandPaletteStore } from "@/lib/stores/command-palette-store";
import type { ContactDto } from "@/lib/api/api/dto/crm.dto";

const pushMock = vi.fn();

vi.mock("next/navigation", () => ({
  useRouter: () => ({ push: pushMock }),
  usePathname: () => "/dashboard",
  useSearchParams: () => new URLSearchParams(),
}));

vi.mock("@/lib/api/api/repositories/crm-repository", () => ({
  crmRepository: {
    searchContacts: vi.fn(),
  },
}));

const mockSearchContacts = vi.mocked(crmRepository.searchContacts);

function openPalette(mode: "command" | "search" = "command") {
  useCommandPaletteStore.getState().openPalette(mode);
}

function closePalette() {
  useCommandPaletteStore.getState().closePalette();
}

describe("CommandPalette", () => {
  beforeEach(() => {
    pushMock.mockReset();
    mockSearchContacts.mockReset();
    closePalette();
  });

  it("opens and focuses the input", async () => {
    const user = userEvent.setup();
    renderWithProviders(<CommandPalette />);

    openPalette();

    const input = await screen.findByPlaceholderText(
      "Type a command or search…"
    );
    expect(input).toBeDefined();
    expect(useCommandPaletteStore.getState().open).toBe(true);
  });

  it("filters navigation targets by fuzzy text match", async () => {
    const user = userEvent.setup();
    renderWithProviders(<CommandPalette />);
    openPalette();

    const input = await screen.findByPlaceholderText(
      "Type a command or search…"
    );
    await user.type(input, "contactos");

    expect(screen.getByText("CRM")).toBeDefined();
    expect(screen.queryByText("Inbox")).toBeNull();
  });

  it("navigates to the selected destination on Enter", async () => {
    const user = userEvent.setup();
    renderWithProviders(<CommandPalette />);
    openPalette();

    const input = await screen.findByPlaceholderText(
      "Type a command or search…"
    );
    await user.type(input, "inbox");
    await user.keyboard("{Enter}");

    await waitFor(() => {
      expect(pushMock).toHaveBeenCalledWith("/dashboard/inbox");
    });
    expect(useCommandPaletteStore.getState().open).toBe(false);
  });

  it("searches contacts in search mode and opens the contact on Enter", async () => {
    const user = userEvent.setup();
    const contact: ContactDto = {
      id: 42,
      organization_id: 7,
      phone_number: "+573001112233",
      display_name: "Maria Lopez",
      source: "whatsapp",
      lead_status: "nuevo",
      is_blocked: false,
      created_at: "2026-01-01T00:00:00Z",
      updated_at: "2026-01-01T00:00:00Z",
    };
    mockSearchContacts.mockResolvedValue({ items: [contact], total: 1 });

    renderWithProviders(<CommandPalette />);
    openPalette("search");

    const input = await screen.findByPlaceholderText("Search contacts…");
    await user.type(input, "maria");

    await waitFor(() => {
      expect(screen.getByText("Maria Lopez")).toBeDefined();
    });

    await user.keyboard("{Enter}");

    await waitFor(() => {
      expect(pushMock).toHaveBeenCalledWith(
        "/dashboard/crm?view=contactos&id=42"
      );
    });
    expect(mockSearchContacts).toHaveBeenCalledWith(
      "maria",
      expect.objectContaining({ limit: 20 })
    );
  });

  it("shows a no-results state when the search returns nothing", async () => {
    const user = userEvent.setup();
    mockSearchContacts.mockResolvedValue({ items: [], total: 0 });

    renderWithProviders(<CommandPalette />);
    openPalette("search");

    const input = await screen.findByPlaceholderText("Search contacts…");
    await user.type(input, "zzzzzz");

    await waitFor(() => {
      expect(screen.getByText("No results")).toBeDefined();
    });
  });
});
