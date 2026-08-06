import { useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { whatsappConfigRepository } from "@/lib/api/api/repositories/whatsapp-config-repository";
import { queryKeys } from "@/lib/hooks/queries/query-keys";
import type { WhatsAppConfigInput } from "@/lib/models/whatsapp-config.model";

export function useUpsertWhatsAppConfig() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (input: WhatsAppConfigInput) =>
      whatsappConfigRepository.upsertConfig(input),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.whatsappConfig.all });
      toast.success("WhatsApp configuration saved successfully");
    },
    onError: (error: Error) => {
      toast.error(error.message || "Failed to save WhatsApp configuration");
    },
  });
}
