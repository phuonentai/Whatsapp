import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { whatsappTemplateRepository } from "@/lib/api/api/repositories/whatsapp-template-repository";
import {
  WhatsAppTemplate,
  WhatsAppTemplateInput,
} from "@/lib/models/whatsapp-template.model";
import { queryKeys } from "./query-keys";

export function useWhatsAppTemplatesQuery(options?: { enabled?: boolean }) {
  return useQuery({
    queryKey: queryKeys.whatsappTemplates.list(),
    queryFn: () => whatsappTemplateRepository.list(),
    retry: false,
    staleTime: 30 * 1000,
    ...options,
  });
}

export function useCreateWhatsAppTemplate() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (input: WhatsAppTemplateInput) =>
      whatsappTemplateRepository.create(input),
    onSuccess: () =>
      qc.invalidateQueries({ queryKey: queryKeys.whatsappTemplates.list() }),
  });
}

export function useUpdateWhatsAppTemplate() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, input }: { id: number; input: WhatsAppTemplateInput }) =>
      whatsappTemplateRepository.update(id, input),
    onSuccess: () =>
      qc.invalidateQueries({ queryKey: queryKeys.whatsappTemplates.list() }),
  });
}

export function useDeleteWhatsAppTemplate() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: number) => whatsappTemplateRepository.remove(id),
    onSuccess: () =>
      qc.invalidateQueries({ queryKey: queryKeys.whatsappTemplates.list() }),
  });
}

export function useSubmitWhatsAppTemplate() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: number) => whatsappTemplateRepository.submit(id),
    onSuccess: (updated: WhatsAppTemplate) => {
      qc.invalidateQueries({ queryKey: queryKeys.whatsappTemplates.list() });
      qc.setQueryData(queryKeys.whatsappTemplates.detail(updated.id), updated);
    },
  });
}

export function useSyncWhatsAppTemplate() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: number) => whatsappTemplateRepository.sync(id),
    onSuccess: (updated: WhatsAppTemplate) => {
      qc.invalidateQueries({ queryKey: queryKeys.whatsappTemplates.list() });
      qc.setQueryData(queryKeys.whatsappTemplates.detail(updated.id), updated);
    },
  });
}
