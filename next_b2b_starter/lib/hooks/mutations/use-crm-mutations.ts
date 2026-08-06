import { useMutation, useQueryClient } from "@tanstack/react-query";
import { crmRepository } from "../../api/api/repositories/crm-repository";
import { queryKeys } from "../queries/query-keys";

export function useCreateContact() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (data: Parameters<typeof crmRepository.createContact>[0]) => crmRepository.createContact(data),
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.crm.contacts() }),
  });
}

export function useUpdateContact() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, data }: { id: number; data: Parameters<typeof crmRepository.updateContact>[1] }) =>
      crmRepository.updateContact(id, data),
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.crm.contacts() }),
  });
}

export function useDeleteContact() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: number) => crmRepository.deleteContact(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.crm.contacts() }),
  });
}

export function useCreateDeal() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (data: Parameters<typeof crmRepository.createDeal>[0]) => crmRepository.createDeal(data),
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.crm.deals() }),
  });
}

export function useUpdateDeal() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, data }: { id: number; data: Parameters<typeof crmRepository.updateDeal>[1] }) =>
      crmRepository.updateDeal(id, data),
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.crm.deals() }),
  });
}

export function useMoveDealStage() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, data }: { id: number; data: Parameters<typeof crmRepository.moveDealStage>[1] }) =>
      crmRepository.moveDealStage(id, data),
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.crm.deals() }),
  });
}

export function useCreateActivity() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (data: Parameters<typeof crmRepository.createActivity>[0]) => crmRepository.createActivity(data),
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.crm.activities() }),
  });
}

export function useCreateCompany() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (data: Parameters<typeof crmRepository.createCompany>[0]) => crmRepository.createCompany(data),
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.crm.companies() }),
  });
}

export function useCreateTag() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (data: Parameters<typeof crmRepository.createTag>[0]) => crmRepository.createTag(data),
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.crm.tags() }),
  });
}

export function useDeleteTag() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: number) => crmRepository.deleteTag(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.crm.tags() }),
  });
}
