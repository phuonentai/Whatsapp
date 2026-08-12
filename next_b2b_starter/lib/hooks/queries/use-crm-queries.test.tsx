import { describe, it, expect, vi, beforeEach } from "vitest";
import { renderHook, waitFor, act } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import type { ReactNode } from "react";
import { useContactsQuery } from "./use-crm-queries";
import { crmRepository } from "@/lib/api/api/repositories/crm-repository";
import type { ContactDto } from "@/lib/api/api/dto/crm.dto";

vi.mock("@/lib/api/api/repositories/crm-repository", () => ({
  crmRepository: { listContacts: vi.fn() },
}));

const CONTACT_1: ContactDto = {
  id: 1,
  organization_id: 1,
  phone_number: "+573000000001",
  display_name: "Uno",
  source: "manual",
  lead_status: "nuevo",
  is_blocked: false,
  created_at: "2026-01-01T00:00:00Z",
  updated_at: "2026-01-01T00:00:00Z",
};

const CONTACT_2: ContactDto = {
  ...CONTACT_1,
  id: 2,
  phone_number: "+573000000002",
  display_name: "Dos",
};

function makeWrapper(client: QueryClient) {
  return function Wrapper({ children }: { children: ReactNode }) {
    return <QueryClientProvider client={client}>{children}</QueryClientProvider>;
  };
}

describe("useContactsQuery pagination", () => {
  const listContacts = vi.mocked(crmRepository.listContacts);

  beforeEach(() => {
    listContacts.mockReset();
  });

  it("maps page/pageSize to limit/offset in the API request", async () => {
    listContacts.mockResolvedValue({ items: [], total: 0 });
    const client = new QueryClient();
    const { result } = renderHook(() => useContactsQuery({ page: 2, pageSize: 25 }), {
      wrapper: makeWrapper(client),
    });
    await waitFor(() => {
      expect(listContacts).toHaveBeenCalledWith({ limit: 25, offset: 25 });
    });
    expect(result.current.total).toBe(0);
  });

  it("keeps the previous page data while the next page is fetching", async () => {
    const client = new QueryClient();
    let resolveNext!: (value: { items: ContactDto[]; total: number }) => void;
    listContacts
      .mockResolvedValueOnce({ items: [CONTACT_1], total: 50 })
      .mockImplementationOnce(
        () => new Promise<{ items: ContactDto[]; total: number }>((resolve) => {
          resolveNext = resolve;
        })
      );

    const { result, rerender } = renderHook(
      ({ page }: { page: number }) => useContactsQuery({ page, pageSize: 25 }),
      { wrapper: makeWrapper(client), initialProps: { page: 1 } }
    );
    await waitFor(() => {
      expect(result.current.data).toEqual([CONTACT_1]);
    });

    // Switch to page 2 without resolving its request: the hook must keep
    // exposing page 1's data (placeholder) instead of blanking out.
    rerender({ page: 2 });
    expect(result.current.data).toEqual([CONTACT_1]);
    expect(result.current.isPlaceholderData).toBe(true);

    await act(async () => {
      resolveNext({ items: [CONTACT_2], total: 50 });
    });
    await waitFor(() => {
      expect(result.current.data).toEqual([CONTACT_2]);
    });
    expect(result.current.isPlaceholderData).toBe(false);
  });
});
