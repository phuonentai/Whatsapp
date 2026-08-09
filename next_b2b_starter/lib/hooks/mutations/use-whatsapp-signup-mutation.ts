import { useMutation, useQueryClient } from "@tanstack/react-query";
import { whatsappSignupRepository } from "@/lib/api/api/repositories/whatsapp-signup-repository";
import { queryKeys } from "@/lib/hooks/queries/query-keys";

export function useWhatsAppSignupExchange() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (code: string) => whatsappSignupRepository.exchange(code),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.whatsappConfig.all });
      queryClient.invalidateQueries({ queryKey: queryKeys.whatsappSignup.all });
    },
    onError: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.whatsappSignup.all });
    },
  });
}
