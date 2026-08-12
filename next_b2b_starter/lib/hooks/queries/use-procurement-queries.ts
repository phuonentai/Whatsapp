// Procurement queries (TanStack Query).

import { useQuery } from "@tanstack/react-query";
import { procurementRepository } from "@/lib/api/api/repositories/procurement-repository";
import { queryKeys } from "./query-keys";

export function useSuppliersQuery() {
  return useQuery({
    queryKey: queryKeys.procurement.suppliers(),
    queryFn: () => procurementRepository.listSuppliers(),
  });
}

export function useProductsQuery() {
  return useQuery({
    queryKey: queryKeys.procurement.products(),
    queryFn: () => procurementRepository.listProducts(),
  });
}

export function useRunsQuery() {
  return useQuery({
    queryKey: queryKeys.procurement.runs(),
    queryFn: () => procurementRepository.listRuns(),
  });
}

export function useRunBoardQuery(runId: number) {
  return useQuery({
    queryKey: queryKeys.procurement.board(runId),
    queryFn: () => procurementRepository.getRunBoard(runId),
    enabled: !!runId,
  });
}

export function useRunOrdersQuery(runId: number) {
  return useQuery({
    queryKey: queryKeys.procurement.orders(runId),
    queryFn: () => procurementRepository.listRunOrders(runId),
    enabled: !!runId,
  });
}

// ---- schedules + follow-up settings (add-scheduled-inquiry-runs) ----

export function useSchedulesQuery() {
  return useQuery({
    queryKey: queryKeys.procurement.schedules(),
    queryFn: () => procurementRepository.listSchedules(),
  });
}

export function useScheduleDetailQuery(scheduleId: number) {
  return useQuery({
    queryKey: queryKeys.procurement.scheduleDetail(scheduleId),
    queryFn: () => procurementRepository.getSchedule(scheduleId),
    enabled: !!scheduleId,
  });
}

export function useFollowUpSettingsQuery() {
  return useQuery({
    queryKey: queryKeys.procurement.followUpSettings(),
    queryFn: () => procurementRepository.getFollowUpSettings(),
  });
}
