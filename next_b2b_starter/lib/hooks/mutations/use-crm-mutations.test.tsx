import { describe, it, expect, vi, beforeEach } from "vitest";
import { renderHook, waitFor, act } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import type { ReactNode } from "react";
import { useMoveDealStage } from "./use-crm-mutations";
import { crmRepository } from "@/lib/api/api/repositories/crm-repository";
import { queryKeys } from "../queries/query-keys";
import type { DealDto } from "@/lib/api/api/dto/crm.dto";

vi.mock("@/lib/api/api/repositories/crm-repository", () => ({
  crmRepository: {
    moveDealStage: vi.fn(),
    // Keep refetches after invalidation from hitting the network.
    listDeals: vi.fn(() => new Promise(() => {})),
  },
}));

const DEAL: DealDto = {
  id: 10,
  organization_id: 1,
  nombre: "Trato grande",
  moneda: "COP",
  pipeline_id: 1,
  stage_id: 1,
  estado: "abierto",
  created_at: "2026-01-01T00:00:00Z",
  updated_at: "2026-01-01T00:00:00Z",
};

function makeWrapper(client: QueryClient) {
  return function Wrapper({ children }: { children: ReactNode }) {
    return <QueryClientProvider client={client}>{children}</QueryClientProvider>;
  };
}

describe("useMoveDealStage optimistic updates", () => {
  const moveDealStage = vi.mocked(crmRepository.moveDealStage);

  beforeEach(() => {
    moveDealStage.mockReset();
  });

  it("moves the card to the target stage before the request resolves", async () => {
    moveDealStage.mockImplementationOnce(() => new Promise(() => {}));
    const client = new QueryClient();
    const key = queryKeys.crm.deals({ pipeline_id: 1 });
    client.setQueryData(key, [DEAL]);

    const { result } = renderHook(() => useMoveDealStage(), { wrapper: makeWrapper(client) });

    await act(async () => {
      result.current.mutate({
        id: 10,
        data: { stage_id: 2, old_stage_name: "Prospección", new_stage_name: "Negociación" },
      });
    });

    // Server has not answered yet — the cache must already show stage 2.
    expect(client.getQueryData<DealDto[]>(key)?.[0].stage_id).toBe(2);
  });

  it("rolls the card back to its original stage when the request fails", async () => {
    let rejectMove!: (reason: Error) => void;
    moveDealStage.mockImplementationOnce(
      () =>
        new Promise((_resolve, reject) => {
          rejectMove = reject;
        })
    );
    const client = new QueryClient();
    const key = queryKeys.crm.deals({ pipeline_id: 1 });
    client.setQueryData(key, [DEAL]);

    const { result } = renderHook(() => useMoveDealStage(), { wrapper: makeWrapper(client) });

    await act(async () => {
      result.current.mutate({
        id: 10,
        data: { stage_id: 2, old_stage_name: "Prospección", new_stage_name: "Negociación" },
      });
    });
    expect(client.getQueryData<DealDto[]>(key)?.[0].stage_id).toBe(2);

    await act(async () => {
      rejectMove(new Error("API Error 500: boom"));
    });
    await waitFor(() => {
      expect(client.getQueryData<DealDto[]>(key)?.[0].stage_id).toBe(1);
    });
  });
});
