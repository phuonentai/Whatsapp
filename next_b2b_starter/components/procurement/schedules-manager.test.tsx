import { describe, it, expect, vi, beforeEach } from "vitest";
import { screen } from "@testing-library/react";
import { renderWithProviders } from "@/test/render";
import { SchedulesManager } from "./schedules-manager";
import type { ScheduleStatusDto } from "@/lib/api/api/dto/procurement.dto";

const SCHEDULE: ScheduleStatusDto = {
  Schedule: {
    ID: 1,
    OrganizationID: 42,
    Name: "Matinal",
    RunTime: "08:00",
    DaysOfWeek: [1, 2, 3, 4, 5],
    ProductIDs: [10],
    SupplierIDs: [1],
    IsActive: true,
    NextRunAt: "2026-08-13T13:00:00Z",
    CreatedAt: "2026-08-11T00:00:00Z",
    UpdatedAt: "2026-08-11T00:00:00Z",
  },
  LastRunID: 7,
  LastRunStatus: "awaiting_responses",
  HasLastRun: true,
};

const pause = vi.fn();
const resume = vi.fn();
const remove = vi.fn();

vi.mock("@/lib/hooks/queries/use-procurement-queries", () => ({
  useSchedulesQuery: () => ({
    data: [SCHEDULE],
    isLoading: false,
    isError: false,
    refetch: vi.fn(),
    isRefetching: false,
  }),
  useScheduleDetailQuery: () => ({
    data: null,
    isLoading: false,
    isError: false,
    refetch: vi.fn(),
    isRefetching: false,
  }),
  useSuppliersQuery: () => ({ data: [], isLoading: false, isError: false, refetch: vi.fn(), isRefetching: false }),
  useProductsQuery: () => ({ data: [], isLoading: false, isError: false, refetch: vi.fn(), isRefetching: false }),
}));

vi.mock("@/lib/hooks/mutations/use-procurement-mutations", () => ({
  useCreateSchedule: () => ({ mutateAsync: vi.fn(), isPending: false }),
  useUpdateSchedule: () => ({ mutateAsync: vi.fn(), isPending: false }),
  usePauseSchedule: () => ({ mutate: pause, isPending: false }),
  useResumeSchedule: () => ({ mutate: resume, isPending: false }),
  useDeleteSchedule: () => ({ mutate: remove, isPending: false }),
}));

beforeEach(() => {
  vi.clearAllMocks();
});

describe("SchedulesManager", () => {
  it("renders the schedule list with next run and last status", () => {
    renderWithProviders(<SchedulesManager />);
    expect(screen.getByText("Matinal")).toBeInTheDocument();
    expect(screen.getByText("08:00")).toBeInTheDocument();
    expect(screen.getByText("awaiting_responses")).toBeInTheDocument();
    expect(screen.getByText("Activa")).toBeInTheDocument();
    expect(screen.getByText("Programaciones de cotizaciones")).toBeInTheDocument();
  });
});
