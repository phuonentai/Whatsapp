import { useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { whatsappConfigRepository } from "@/lib/api/api/repositories/whatsapp-config-repository";
import { queryKeys } from "@/lib/hooks/queries/query-keys";

export function useToggleWhatsAppConfig() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: () => whatsappConfigRepository.toggleConfig(),
    onSuccess: (data) => {
      queryClient.invalidateQueries({ queryKey: queryKeys.whatsappConfig.all });
      toast.success(
        data.isActive
          ? "WhatsApp messaging is now active"
          : "WhatsApp messaging is now paused"
      );
    },
    onError: (error: Error) => {
      toast.error(error.message || "Failed to toggle WhatsApp configuration");
    },
  });
}
