import { useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { campaignRepository } from "../../api/api/repositories/campaign-repository";
import { queryKeys } from "../queries/query-keys";
import { toSpanishMutationError } from "../../crm/errors";

function onMutationError(error: unknown) {
  toast.error(toSpanishMutationError(error));
}

export function useCreateSegment() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (data: { nombre: string; filter_spec: Parameters<typeof campaignRepository.createSegment>[0]["filter_spec"] }) =>
      campaignRepository.createSegment(data),
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.crm.segments() }),
    onError: onMutationError,
  });
}

export function useUpdateSegment() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, data }: { id: number; data: { nombre: string; filter_spec: unknown[] } }) =>
      campaignRepository.updateSegment(id, data as Parameters<typeof campaignRepository.updateSegment>[1]),
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.crm.segments() }),
    onError: onMutationError,
  });
}

export function useDeleteSegment() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: number) => campaignRepository.deleteSegment(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.crm.segments() }),
    onError: onMutationError,
  });
}

export function useAiBuild() {
  return useMutation({
    mutationFn: (descripcion: string) => campaignRepository.aiBuild(descripcion),
    onError: onMutationError,
  });
}

export function useCreateCampaign() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (data: { nombre: string; segment_id: number; mensaje?: string }) =>
      campaignRepository.createCampaign(data),
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.crm.campaigns() }),
    onError: onMutationError,
  });
}

export function useLaunchCampaign() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: number) => campaignRepository.launchCampaign(id),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: queryKeys.crm.campaigns() });
      qc.invalidateQueries({ queryKey: queryKeys.crm.segments() });
    },
    onError: onMutationError,
  });
}
