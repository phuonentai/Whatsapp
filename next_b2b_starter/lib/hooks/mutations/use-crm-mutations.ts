import { useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { crmRepository } from "../../api/api/repositories/crm-repository";
import { queryKeys } from "../queries/query-keys";
import { toSpanishMutationError } from "../../crm/errors";
import type { DealDto } from "../../api/api/dto/crm.dto";

function onMutationError(error: unknown) {
  toast.error(toSpanishMutationError(error));
}

export function useCreateContact() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (data: Parameters<typeof crmRepository.createContact>[0]) => crmRepository.createContact(data),
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.crm.all }),
    onError: onMutationError,
  });
}

export function useUpdateContact() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, data }: { id: number; data: Parameters<typeof crmRepository.updateContact>[1] }) =>
      crmRepository.updateContact(id, data),
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.crm.all }),
    onError: onMutationError,
  });
}

export function useDeleteContact() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: number) => crmRepository.deleteContact(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.crm.all }),
    onError: onMutationError,
  });
}

export function useCreateCompany() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (data: Parameters<typeof crmRepository.createCompany>[0]) => crmRepository.createCompany(data),
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.crm.all }),
    onError: onMutationError,
  });
}

export function useUpdateCompany() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, data }: { id: number; data: Parameters<typeof crmRepository.updateCompany>[1] }) =>
      crmRepository.updateCompany(id, data),
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.crm.all }),
    onError: onMutationError,
  });
}

export function useDeleteCompany() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: number) => crmRepository.deleteCompany(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.crm.all }),
    onError: onMutationError,
  });
}

export function useCreateDeal() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (data: Parameters<typeof crmRepository.createDeal>[0]) => crmRepository.createDeal(data),
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.crm.all }),
    onError: onMutationError,
  });
}

export function useUpdateDeal() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, data }: { id: number; data: Parameters<typeof crmRepository.updateDeal>[1] }) =>
      crmRepository.updateDeal(id, data),
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.crm.all }),
    onError: onMutationError,
  });
}

export function useMoveDealStage() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, data }: { id: number; data: Parameters<typeof crmRepository.moveDealStage>[1] }) =>
      crmRepository.moveDealStage(id, data),
    // Optimistic: move the card to the target stage in every cached deals
    // list immediately, snapshotting the previous state for rollback.
    onMutate: async ({ id, data }) => {
      await qc.cancelQueries({ queryKey: queryKeys.crm.deals() });
      const previous = qc.getQueriesData<DealDto[]>({ queryKey: queryKeys.crm.deals() });
      qc.setQueriesData<DealDto[]>({ queryKey: queryKeys.crm.deals() }, (old) =>
        old?.map((deal) => (deal.id === id ? { ...deal, stage_id: data.stage_id } : deal))
      );
      return { previous };
    },
    onError: (error, _vars, context) => {
      // Restore the snapshot so the card returns to its original stage.
      if (context?.previous) {
        for (const [key, deals] of context.previous) {
          qc.setQueryData(key, deals);
        }
      }
      onMutationError(error);
    },
    onSettled: () => qc.invalidateQueries({ queryKey: queryKeys.crm.all }),
  });
}

export function useCreateActivity() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (data: Parameters<typeof crmRepository.createActivity>[0]) => crmRepository.createActivity(data),
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.crm.all }),
    onError: onMutationError,
  });
}

export function useCreateTag() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (data: Parameters<typeof crmRepository.createTag>[0]) => crmRepository.createTag(data),
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.crm.all }),
    onError: onMutationError,
  });
}

export function useUpdateTag() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, data }: { id: number; data: Parameters<typeof crmRepository.updateTag>[1] }) =>
      crmRepository.updateTag(id, data),
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.crm.all }),
    onError: onMutationError,
  });
}

export function useTagEntity() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (args: { entityType: string; entityId: number; tagId: number }) =>
      crmRepository.tagEntity(args.entityType, args.entityId, args.tagId),
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.crm.all }),
    onError: onMutationError,
  });
}

export function useUntagEntity() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (args: { entityType: string; entityId: number; tagId: number }) =>
      crmRepository.untagEntity(args.entityType, args.entityId, args.tagId),
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.crm.all }),
    onError: onMutationError,
  });
}

export function useDeleteTag() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: number) => crmRepository.deleteTag(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.crm.all }),
    onError: onMutationError,
  });
}

export function useDeleteDeal() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: number) => crmRepository.deleteDeal(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.crm.all }),
    onError: onMutationError,
  });
}

export function useCreatePipeline() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (data: Parameters<typeof crmRepository.createPipeline>[0]) => crmRepository.createPipeline(data),
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.crm.all }),
    onError: onMutationError,
  });
}

export function useUpdatePipeline() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, data }: { id: number; data: Parameters<typeof crmRepository.updatePipeline>[1] }) =>
      crmRepository.updatePipeline(id, data),
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.crm.all }),
    onError: onMutationError,
  });
}

export function useCreateStage() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (args: { pipelineId: number; data: Parameters<typeof crmRepository.createStage>[1] }) =>
      crmRepository.createStage(args.pipelineId, args.data),
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.crm.all }),
    onError: onMutationError,
  });
}

export function useUpdateStage() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (args: { pipelineId: number; stageId: number; data: Parameters<typeof crmRepository.updateStage>[2] }) =>
      crmRepository.updateStage(args.pipelineId, args.stageId, args.data),
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.crm.all }),
    onError: onMutationError,
  });
}
