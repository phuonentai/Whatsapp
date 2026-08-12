// Procurement mutations (TanStack Query + Spanish toasts).

import { useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { procurementRepository } from "@/lib/api/api/repositories/procurement-repository";
import { queryKeys } from "../queries/query-keys";
import { toSpanishMutationError } from "@/lib/crm/errors";
import { ui } from "@/lib/copy/ui";
import type {
  CreateProductInput,
  CreateRunInput,
  CreateScheduleInput,
  CreateSupplierInput,
  PlaceOrderInput,
  UpdateFollowUpSettingsInput,
  UpdateProductInput,
  UpdateSupplierInput,
} from "@/lib/api/api/dto/procurement.dto";

function onError(error: unknown) {
  toast.error(toSpanishMutationError(error));
}

export function useCreateSupplier() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (data: CreateSupplierInput) => procurementRepository.createSupplier(data),
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.procurement.suppliers() }),
    onError: onError,
  });
}

export function useUpdateSupplier() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, data }: { id: number; data: UpdateSupplierInput }) =>
      procurementRepository.updateSupplier(id, data),
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.procurement.suppliers() }),
    onError: onError,
  });
}

export function useCreateProduct() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (data: CreateProductInput) => procurementRepository.createProduct(data),
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.procurement.products() }),
    onError: onError,
  });
}

export function useUpdateProduct() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, data }: { id: number; data: UpdateProductInput }) =>
      procurementRepository.updateProduct(id, data),
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.procurement.products() }),
    onError: onError,
  });
}

export function useCreateRun() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (data: CreateRunInput) => procurementRepository.createRun(data),
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.procurement.runs() }),
    onError: onError,
  });
}

export function useSendRun() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: number) => procurementRepository.sendRun(id),
    onSuccess: (_, id) => {
      qc.invalidateQueries({ queryKey: queryKeys.procurement.runs() });
      qc.invalidateQueries({ queryKey: queryKeys.procurement.board(id) });
    },
    onError: onError,
  });
}

export function usePlaceOrder() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ runId, data }: { runId: number; data: PlaceOrderInput }) =>
      procurementRepository.placeOrder(runId, data),
    onSuccess: (_, { runId }) => {
      qc.invalidateQueries({ queryKey: queryKeys.procurement.orders(runId) });
      qc.invalidateQueries({ queryKey: queryKeys.procurement.board(runId) });
    },
    onError: onError,
  });
}

// ---- schedules + follow-up settings (add-scheduled-inquiry-runs) ----

export function useCreateSchedule() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (data: CreateScheduleInput) => procurementRepository.createSchedule(data),
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.procurement.schedules() }),
    onError: onError,
  });
}

export function useUpdateSchedule() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, data }: { id: number; data: CreateScheduleInput }) =>
      procurementRepository.updateSchedule(id, data),
    onSuccess: (_, { id }) => {
      qc.invalidateQueries({ queryKey: queryKeys.procurement.schedules() });
      qc.invalidateQueries({ queryKey: queryKeys.procurement.scheduleDetail(id) });
    },
    onError: onError,
  });
}

export function usePauseSchedule() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: number) => procurementRepository.pauseSchedule(id),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: queryKeys.procurement.schedules() });
      toast.success(ui.procurement.schedulePausedToast);
    },
    onError: onError,
  });
}

export function useResumeSchedule() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: number) => procurementRepository.resumeSchedule(id),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: queryKeys.procurement.schedules() });
      toast.success(ui.procurement.scheduleResumedToast);
    },
    onError: onError,
  });
}

export function useDeleteSchedule() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: number) => procurementRepository.deleteSchedule(id),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: queryKeys.procurement.schedules() });
      toast.success(ui.procurement.scheduleDeleted);
    },
    onError: onError,
  });
}

export function useUpdateFollowUpSettings() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (data: UpdateFollowUpSettingsInput) =>
      procurementRepository.updateFollowUpSettings(data),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: queryKeys.procurement.followUpSettings() });
      toast.success(ui.procurement.followUpSaved);
    },
    onError: onError,
  });
}
